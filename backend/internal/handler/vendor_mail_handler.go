package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/vendormail"
	"github.com/go-chi/chi/v5"
)

const vendorMailFetchEvery = 5 * time.Minute

// VendorMailHandler pilote la boucle de discussion email avec les vendeurs
// depuis atekossibrunel@diarra.app (demandé le 2026-10-05) : lecture
// périodique des réponses par IMAP, file de brouillons sortants à valider
// par un admin avant tout envoi réel. Volontairement SANS envoi ni réponse
// automatique — voir vendormail.Sender, qui n'est appelé que depuis
// ApproveDraft une fois la validation humaine faite.
type VendorMailHandler struct {
	repo      *repository.VendorMailRepo
	userRepo  *repository.UserRepo
	sender    *vendormail.Sender // nil si non configuré (dev local)
	readerCfg vendormail.ReaderConfig
	imapReady bool
}

func NewVendorMailHandler(repo *repository.VendorMailRepo, userRepo *repository.UserRepo, sender *vendormail.Sender, readerCfg vendormail.ReaderConfig, imapReady bool) *VendorMailHandler {
	return &VendorMailHandler{repo: repo, userRepo: userRepo, sender: sender, readerCfg: readerCfg, imapReady: imapReady}
}

// RunFetchLoop interroge périodiquement la boîte IMAP pour rattacher les
// réponses des vendeurs à leur fil. Ne touche jamais à l'envoi.
func (h *VendorMailHandler) RunFetchLoop(ctx context.Context) {
	if !h.imapReady {
		log.Printf("vendor-mail: lecture IMAP désactivée (non configurée)")
		return
	}
	ticker := time.NewTicker(vendorMailFetchEvery)
	defer ticker.Stop()
	h.fetchPass(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.fetchPass(ctx)
		}
	}
}

const vendorMailMailbox = "INBOX"

func (h *VendorMailHandler) fetchPass(ctx context.Context) {
	lastUID, err := h.repo.GetLastUID(ctx, vendorMailMailbox)
	if err != nil {
		log.Printf("vendor-mail: lecture curseur IMAP échouée: %v", err)
		return
	}

	messages, maxUID, err := vendormail.FetchNewMessages(h.readerCfg, lastUID)
	if err != nil {
		log.Printf("vendor-mail: récupération IMAP échouée: %v", err)
		return
	}

	for _, msg := range messages {
		vendor, err := h.userRepo.FindByEmail(ctx, msg.FromEmail)
		if err != nil || vendor == nil {
			continue // email reçu d'une adresse qui n'est pas un compte vendeur connu
		}
		thread, err := h.repo.FindOrCreateThread(ctx, vendor.ID, msg.Subject)
		if err != nil {
			log.Printf("vendor-mail: fil introuvable pour %s: %v", vendor.ID, err)
			continue
		}
		var inReplyTo *string
		if msg.InReplyTo != "" {
			inReplyTo = &msg.InReplyTo
		}
		if _, err := h.repo.CreateInbound(ctx, thread.ID, msg.Subject, msg.Body, msg.MessageID, inReplyTo); err != nil {
			log.Printf("vendor-mail: enregistrement message entrant échoué: %v", err)
			continue
		}
		_ = h.repo.TouchThread(ctx, thread.ID, msg.MessageID)
	}

	if maxUID > lastUID {
		if err := h.repo.SetLastUID(ctx, vendorMailMailbox, maxUID); err != nil {
			log.Printf("vendor-mail: mise à jour curseur IMAP échouée: %v", err)
		}
	}
}

// ListThreads — GET /api/admin/vendor-mail/threads
func (h *VendorMailHandler) ListThreads(w http.ResponseWriter, r *http.Request) {
	threads, err := h.repo.ListThreads(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"threads": threads})
}

// GetThread — GET /api/admin/vendor-mail/threads/{id}
func (h *VendorMailHandler) GetThread(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	thread, err := h.repo.FindThreadByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	messages, err := h.repo.ListMessages(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"thread": thread, "messages": messages})
}

// ListDrafts — GET /api/admin/vendor-mail/drafts : la file à valider.
func (h *VendorMailHandler) ListDrafts(w http.ResponseWriter, r *http.Request) {
	drafts, err := h.repo.ListDraftMessages(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"drafts": drafts})
}

// StartThread — POST /api/admin/vendor-mail/start : crée un brouillon de
// premier message vers un vendeur choisi par l'admin (jamais automatique).
func (h *VendorMailHandler) StartThread(w http.ResponseWriter, r *http.Request) {
	var input model.StartVendorMailThreadInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	input.Subject = strings.TrimSpace(input.Subject)
	input.Body = strings.TrimSpace(input.Body)
	if input.VendorID == "" || input.Subject == "" || input.Body == "" {
		http.Error(w, `{"error":"vendor_id_subject_and_body_required"}`, http.StatusBadRequest)
		return
	}

	thread, err := h.repo.FindOrCreateThread(r.Context(), input.VendorID, input.Subject)
	if err != nil {
		http.Error(w, `{"error":"thread_failed"}`, http.StatusInternalServerError)
		return
	}
	draft, err := h.repo.CreateDraft(r.Context(), thread.ID, input.Subject, input.Body, nil)
	if err != nil {
		http.Error(w, `{"error":"draft_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"draft": draft})
}

// DraftReply — POST /api/admin/vendor-mail/threads/{id}/reply : prépare un
// brouillon de réponse dans un fil existant.
func (h *VendorMailHandler) DraftReply(w http.ResponseWriter, r *http.Request) {
	threadID := chi.URLParam(r, "id")
	thread, err := h.repo.FindThreadByID(r.Context(), threadID)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	var input model.DraftVendorMailReplyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	input.Body = strings.TrimSpace(input.Body)
	if input.Body == "" {
		http.Error(w, `{"error":"body_required"}`, http.StatusBadRequest)
		return
	}
	draft, err := h.repo.CreateDraft(r.Context(), thread.ID, thread.Subject, input.Body, thread.LastMessageID)
	if err != nil {
		http.Error(w, `{"error":"draft_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"draft": draft})
}

// UpdateDraft — PUT /api/admin/vendor-mail/drafts/{id} : l'admin corrige le
// texte avant validation.
func (h *VendorMailHandler) UpdateDraft(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var input model.DraftVendorMailReplyInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	input.Body = strings.TrimSpace(input.Body)
	if input.Body == "" {
		http.Error(w, `{"error":"body_required"}`, http.StatusBadRequest)
		return
	}
	if err := h.repo.UpdateDraftBody(r.Context(), id, input.Body); err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ApproveDraft — POST /api/admin/vendor-mail/drafts/{id}/approve : SEUL
// point d'entrée qui envoie réellement un email à un vendeur. Appelé
// uniquement depuis l'action explicite d'un admin dans /admin/vendor-mail.
func (h *VendorMailHandler) ApproveDraft(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if h.sender == nil {
		http.Error(w, `{"error":"mail_sender_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	draft, err := h.repo.FindMessageByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if draft.Status != "draft" {
		http.Error(w, `{"error":"already_processed"}`, http.StatusConflict)
		return
	}
	thread, err := h.repo.FindThreadByID(r.Context(), draft.ThreadID)
	if err != nil {
		http.Error(w, `{"error":"thread_not_found"}`, http.StatusNotFound)
		return
	}
	vendor, err := h.userRepo.FindByID(r.Context(), thread.VendorID)
	if err != nil || vendor.Email == "" {
		http.Error(w, `{"error":"vendor_email_not_found"}`, http.StatusNotFound)
		return
	}

	adminID := middleware.GetUserID(r.Context())
	if err := h.repo.ApproveDraft(r.Context(), id, adminID); err != nil {
		http.Error(w, `{"error":"approve_failed"}`, http.StatusInternalServerError)
		return
	}

	sentCtx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	messageID, err := h.sender.Send(sentCtx, vendor.Email, draft.Subject, draft.Body, draft.InReplyTo)
	if err != nil {
		log.Printf("vendor-mail: envoi échoué pour le brouillon %s: %v", id, err)
		http.Error(w, `{"error":"send_failed"}`, http.StatusBadGateway)
		return
	}
	if err := h.repo.MarkSent(r.Context(), id, messageID); err != nil {
		log.Printf("vendor-mail: marquage envoyé échoué pour %s: %v", id, err)
	}
	_ = h.repo.TouchThread(r.Context(), thread.ID, messageID)

	w.WriteHeader(http.StatusNoContent)
}

// Broadcast — POST /api/admin/vendor-mail/broadcast : crée un brouillon
// personnalisé pour chaque compte des rôles demandés ("vendeur", "closer").
// Ne crée QUE des brouillons ('draft') — aucun envoi ici, voir ApproveDraft,
// seul point qui envoie réellement un email.
func (h *VendorMailHandler) Broadcast(w http.ResponseWriter, r *http.Request) {
	var input model.BroadcastVendorMailInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	input.Subject = strings.TrimSpace(input.Subject)
	input.Body = strings.TrimSpace(input.Body)
	if input.Subject == "" || input.Body == "" || len(input.Roles) == 0 {
		http.Error(w, `{"error":"subject_body_and_roles_required"}`, http.StatusBadRequest)
		return
	}
	for _, role := range input.Roles {
		if role != "vendeur" && role != "closer" {
			http.Error(w, `{"error":"invalid_role"}`, http.StatusBadRequest)
			return
		}
	}

	seen := map[string]bool{}
	created := 0
	for _, role := range input.Roles {
		recipients, err := h.repo.ListByRole(r.Context(), role)
		if err != nil {
			http.Error(w, `{"error":"list_recipients_failed"}`, http.StatusInternalServerError)
			return
		}
		for _, rec := range recipients {
			if seen[rec.UserID] {
				continue // un compte peut avoir plusieurs rôles — un seul message, pas un par rôle
			}
			seen[rec.UserID] = true

			thread, err := h.repo.FindOrCreateThread(r.Context(), rec.UserID, input.Subject)
			if err != nil {
				log.Printf("vendor-mail broadcast: fil introuvable pour %s: %v", rec.UserID, err)
				continue
			}
			if _, err := h.repo.CreateDraft(r.Context(), thread.ID, input.Subject, personalize(input.Body, rec.Name), nil); err != nil {
				log.Printf("vendor-mail broadcast: brouillon échoué pour %s: %v", rec.UserID, err)
				continue
			}
			created++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"created": created})
}

// personalize remplace le placeholder littéral "{{nom}}" par le nom du
// destinataire, avec un espace avant pour rester naturel ("Bonjour{{nom}},"
// -> "Bonjour Nom,"), ou le retire proprement (double espace nettoyé) si ce
// destinataire n'a pas de nom connu.
func personalize(body, name string) string {
	if name == "" {
		result := strings.ReplaceAll(body, "{{nom}}", "")
		return strings.ReplaceAll(result, "  ", " ")
	}
	return strings.ReplaceAll(body, "{{nom}}", name)
}

// RejectDraft — POST /api/admin/vendor-mail/drafts/{id}/reject : l'admin
// écarte un brouillon sans l'envoyer.
func (h *VendorMailHandler) RejectDraft(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.RejectDraft(r.Context(), id); err != nil {
		http.Error(w, `{"error":"reject_failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
