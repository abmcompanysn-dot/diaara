package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/diarra/backend/internal/payment"
	"github.com/go-chi/chi/v5"
)

// WebinarHandler — pilotage admin des webinaires YES Business (page
// Admin → Webinaires). DIARRA crée/démarre/termine le webinaire et lit ses
// statistiques ; la diffusion, le tchat, les Q/R, les sondages, l'inscription
// et le replay restent entièrement côté interface YES (voir yes_webinar.go).
type WebinarHandler struct {
	yes *payment.YesBusinessClient // nil si YES_BUSINESS_API_KEY/SECRET absents
}

func NewWebinarHandler(yes *payment.YesBusinessClient) *WebinarHandler {
	return &WebinarHandler{yes: yes}
}

// writeYesError traduit une erreur YES en réponse HTTP propre, sans jamais
// faire planter l'appelant : 401 (clé/HMAC refusés — problème de config
// DIARRA, renvoyé en 502 pour ne pas déconnecter l'admin côté frontend),
// 422 (champs invalides, détail YES relayé), 502 (YES indisponible).
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

// yesErrorDetails relaie le corps d'erreur YES (JSON si possible) — utile à
// l'admin pour corriger un champ refusé (422).
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

// List — GET /api/admin/webinars
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

// maxWebinarCoverUploadBytes — limite de la lecture du champ "file" (doc YES
// Business : 6 Mo max, voir payment.MaxWebinarCoverImageBytes). Légèrement
// au-dessus pour laisser respirer l'overhead multipart avant le rejet exact
// côté client YES (UploadWebinarCoverImage revérifie la taille exacte).
const maxWebinarCoverUploadBytes = payment.MaxWebinarCoverImageBytes + (64 << 10)

// UploadCoverImage — POST /api/admin/webinars/cover-image (multipart/form-data,
// champ "file"). Sniffe le contenu réel (même principe que
// ProductHandler.validCoverImage — jamais l'extension/Content-Type déclaré
// par le client) avant de relayer à YES, pour ne jamais transmettre un
// fichier qui ne soit pas réellement une image raster. Renvoie
// {"cover_image_url": "..."} à reposer tel quel dans CreateWebinarRequest.
func (h *WebinarHandler) UploadCoverImage(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxWebinarCoverUploadBytes)
	if err := r.ParseMultipartForm(maxWebinarCoverUploadBytes); err != nil {
		http.Error(w, `{"error":"file_too_large_or_invalid"}`, http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"file_required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, payment.MaxWebinarCoverImageBytes+1))
	if err != nil {
		http.Error(w, `{"error":"read_failed"}`, http.StatusInternalServerError)
		return
	}
	if len(data) > payment.MaxWebinarCoverImageBytes {
		http.Error(w, `{"error":"file_too_large"}`, http.StatusBadRequest)
		return
	}
	contentType := http.DetectContentType(data)
	if !allowedWebinarCoverImageTypes[contentType] {
		http.Error(w, `{"error":"invalid_image_type"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := yesCtx(r)
	defer cancel()
	resp, err := h.yes.UploadWebinarCoverImage(ctx, header.Filename, contentType, data)
	if err != nil {
		writeYesError(w, "upload-cover-image", err)
		return
	}
	writeWebinarJSON(w, http.StatusOK, map[string]string{"cover_image_url": resp.CoverImageURL})
}

// allowedWebinarCoverImageTypes — PNG, JPEG ou WEBP (doc YES Business
// 2026-10-02, §6.0) ; GIF exclu ici contrairement à
// ProductHandler.allowedCoverImageTypes, YES n'en fait pas mention.
var allowedWebinarCoverImageTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/webp": true,
}

// Create — POST /api/admin/webinars. Validation minimale côté DIARRA pour
// renvoyer une erreur claire avant l'aller-retour YES ; YES reste l'arbitre
// final (422 relayé).
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
	input.CoverImageURL = strings.TrimSpace(input.CoverImageURL)
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

// Start — POST /api/admin/webinars/{id}/start, corps optionnel
// {enable_recording}. Renvoie le webinaire + host_join_link (bouton
// « Rejoindre en tant qu'hôte », valable 10 min, jamais stocké) et
// registration_link.
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
	// Seulement les liens (doc YES §6.3) : host_token/websocket_token sont
	// des jetons LiveKit bruts sans usage dans l'admin DIARRA, ils ne
	// quittent pas le serveur.
	writeWebinarJSON(w, http.StatusOK, map[string]interface{}{
		"webinar":           resp.Webinar,
		"host_join_link":    resp.HostJoinLink,
		"registration_link": resp.RegistrationLink,
	})
}

// End — POST /api/admin/webinars/{id}/end
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

// Registrations — GET /api/admin/webinars/{id}/registrations : inscrits
// (email, prénom, nom, téléphone, entreprise, présent au live, date).
func (h *WebinarHandler) Registrations(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	ctx, cancel := yesCtx(r)
	defer cancel()
	regs, err := h.yes.ListWebinarRegistrations(ctx, chi.URLParam(r, "id"))
	if err != nil {
		writeYesError(w, "registrations", err)
		return
	}
	writeWebinarJSON(w, http.StatusOK, map[string]interface{}{"registrations": regs})
}

// ResendRegistration — POST /api/admin/webinars/{id}/registrations/{regId}/resend :
// renvoie l'email d'accès à UN inscrit (bouton « Relancer »). Le contenu
// ("ça commence bientôt" / "c'est en direct") est choisi par YES selon le
// statut du webinaire.
func (h *WebinarHandler) ResendRegistration(w http.ResponseWriter, r *http.Request) {
	if !h.ready(w) {
		return
	}
	ctx, cancel := yesCtx(r)
	defer cancel()
	err := h.yes.ResendWebinarRegistration(ctx, chi.URLParam(r, "id"), chi.URLParam(r, "regId"))
	if err != nil {
		// 404 sur ce endpoint = inscription introuvable (doc YES §6.7), pas
		// webinaire introuvable comme ailleurs.
		var apiErr *payment.YesAPIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
			log.Printf("yes webinar resend: %v", err)
			http.Error(w, `{"error":"registration_not_found"}`, http.StatusNotFound)
			return
		}
		writeYesError(w, "resend", err)
		return
	}
	writeWebinarJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

// Stats — GET /api/admin/webinars/{id}/stats
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
