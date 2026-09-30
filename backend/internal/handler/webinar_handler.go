package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/diarra/backend/internal/payment"
	"github.com/go-chi/chi/v5"
)

// WebinarHandler â€” pilotage admin des webinaires YES Business (page
// Admin â†’ Webinaires). DIARRA crÃ©e/dÃ©marre/termine le webinaire et lit ses
// statistiques ; la diffusion, le tchat, les Q/R, les sondages, l'inscription
// et le replay restent entiÃ¨rement cÃ´tÃ© interface YES (voir yes_webinar.go).
type WebinarHandler struct {
	yes *payment.YesBusinessClient // nil si YES_BUSINESS_API_KEY/SECRET absents
}

func NewWebinarHandler(yes *payment.YesBusinessClient) *WebinarHandler {
	return &WebinarHandler{yes: yes}
}

// writeYesError traduit une erreur YES en rÃ©ponse HTTP propre, sans jamais
// faire planter l'appelant : 401 (clÃ©/HMAC refusÃ©s â€” problÃ¨me de config
// DIARRA, renvoyÃ© en 502 pour ne pas dÃ©connecter l'admin cÃ´tÃ© frontend),
// 422 (champs invalides, dÃ©tail YES relayÃ©), 502 (YES indisponible).
func writeYesError(w http.ResponseWriter, op string, err error) {
	log.Printf("yes webinar %s: %v", op, err)
	w.Header().Set("Content-Type", "application/json")
	var apiErr *payment.YesAPIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden:
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(map[string]string{"error": "yes_auth_failed"})
			return
		case apiErr.StatusCode == http.StatusUnprocessableEntity || apiErr.StatusCode == http.StatusBadRequest:
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "webinar_invalid",
				"details": yesErrorDetails(apiErr.Body),
			})
			return
		case apiErr.StatusCode == http.StatusNotFound:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "webinar_not_found"})
			return
		case apiErr.StatusCode == http.StatusConflict:
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "webinar_state_conflict",
				"details": yesErrorDetails(apiErr.Body),
			})
			return
		}
	}
	w.WriteHeader(http.StatusBadGateway)
	json.NewEncoder(w).Encode(map[string]string{"error": "yes_unavailable"})
}

// yesErrorDetails relaie le corps d'erreur YES (JSON si possible) â€” utile Ã
// l'admin pour corriger un champ refusÃ© (422).
func yesErrorDetails(body string) interface{} {
	var parsed interface{}
	if json.Unmarshal([]byte(body), &parsed) == nil {
		return parsed
	}
	if len(body) > 300 {
		body = body[:300]
	}
	return body
}

func (h *WebinarHandler) ready(w http.ResponseWriter) bool {
	if h.yes == nil {
		http.Error(w, `{"error":"yes_not_configured"}`, http.StatusServiceUnavailable)
		return false
	}
	return true
}

func writeWebinarJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func yesCtx(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(r.Context(), 20*time.Second)
}

// List â€” GET /api/admin/webinars
func (h *WebinarHandler) List(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	ctx, cancel := yesCtx(r)
	defer cancel()
	webinars, err := h.yes.ListWebinars(ctx)
	if err != nil {
		writeYesError(w, "list", err)
		return
	}
	writeWebinarJSON(w, http.StatusOK, map[string]interface{}{"webinars": webinars})
}

// Create â€” POST /api/admin/webinars. Validation minimale cÃ´tÃ© DIARRA pour
// renvoyer une erreur claire avant l'aller-retour YES ; YES reste l'arbitre
// final (422 relayÃ©).
func (h *WebinarHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	var input payment.CreateWebinarRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if input.Title == "" {
		http.Error(w, `{"error":"title_required"}`, http.StatusBadRequest)
		return
	}
	start, err := time.Parse(time.RFC3339, input.ScheduledStartAt)
	if err != nil {
		http.Error(w, `{"error":"invalid_start_date"}`, http.StatusBadRequest)
		return
	}
	input.ScheduledStartAt = start.UTC().Format(time.RFC3339)
	if input.EstimatedDurationMinutes <= 0 {
		input.EstimatedDurationMinutes = 60
	}
	if input.AccessType != "private" {
		input.AccessType = "public"
	}
	fields := input.CustomRegistrationFields[:0]
	for _, f := range input.CustomRegistrationFields {
		f.Key = strings.TrimSpace(f.Key)
		f.Label = strings.TrimSpace(f.Label)
		if f.Key != "" && f.Label != "" {
			fields = append(fields, f)
		}
	}
	input.CustomRegistrationFields = fields

	ctx, cancel := yesCtx(r)
	defer cancel()
	webinar, err := h.yes.CreateWebinar(ctx, input)
	if err != nil {
		writeYesError(w, "create", err)
		return
	}
	writeWebinarJSON(w, http.StatusCreated, map[string]interface{}{"webinar": webinar})
}

// Start â€” POST /api/admin/webinars/{id}/start, corps optionnel
// {enable_recording}. Les jetons host/websocket renvoyÃ©s par YES servent Ã
// leur interface d'hÃ©bergement : relayÃ©s Ã  l'admin pour qu'il ouvre la
// salle, jamais stockÃ©s ni utilisÃ©s par DIARRA.
func (h *WebinarHandler) Start(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	var input struct {
		EnableRecording bool `json:"enable_recording"`
	}
	_ = json.NewDecoder(r.Body).Decode(&input) // corps optionnel
	ctx, cancel := yesCtx(r)
	defer cancel()
	resp, err := h.yes.StartWebinar(ctx, chi.URLParam(r, "id"), input.EnableRecording)
	if err != nil {
		writeYesError(w, "start", err)
		return
	}
	writeWebinarJSON(w, http.StatusOK, resp)
}

// End â€” POST /api/admin/webinars/{id}/end
func (h *WebinarHandler) End(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	ctx, cancel := yesCtx(r)
	defer cancel()
	webinar, err := h.yes.EndWebinar(ctx, chi.URLParam(r, "id"))
	if err != nil {
		writeYesError(w, "end", err)
		return
	}
	writeWebinarJSON(w, http.StatusOK, map[string]interface{}{"webinar": webinar})
}

// Stats â€” GET /api/admin/webinars/{id}/stats
func (h *WebinarHandler) Stats(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	ctx, cancel := yesCtx(r)
	defer cancel()
	stats, err := h.yes.GetWebinarStats(ctx, chi.URLParam(r, "id"))
	if err != nil {
		writeYesError(w, "stats", err)
		return
	}
	writeWebinarJSON(w, http.StatusOK, stats)
}
