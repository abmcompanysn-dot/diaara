package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/service"
	"github.com/diarra/backend/internal/storage"
	"github.com/go-chi/chi/v5"
)

// AnnouncementHandler — "Dernières mises à jour" côté admin -> dashboard
// vendeur (voir migrations/040_announcements.sql, 043_announcements_image_and_push.sql).
// Chaque création envoie une notification in-app + push à tous les
// vendeurs, avec l'image jointe si présente (voir Image, comme
// ProductHandler.Cover).
type AnnouncementHandler struct {
	repo             *repository.AnnouncementRepo
	userRepo         *repository.UserRepo
	notificationRepo *repository.NotificationRepo
	storage          *storage.S3Storage
	pushSvc          *service.PushService
	apiURL           string
}

func NewAnnouncementHandler(
	repo *repository.AnnouncementRepo,
	userRepo *repository.UserRepo,
	notificationRepo *repository.NotificationRepo,
	storageSvc *storage.S3Storage,
	pushSvc *service.PushService,
	apiURL string,
) *AnnouncementHandler {
	return &AnnouncementHandler{
		repo: repo, userRepo: userRepo, notificationRepo: notificationRepo,
		storage: storageSvc, pushSvc: pushSvc, apiURL: apiURL,
	}
}

// Create — POST /api/admin/announcements (scope "users" ou équivalent large,
// voir routing dans main.go).
func (h *AnnouncementHandler) Create(w http.ResponseWriter, r *http.Request) {
	adminID := middleware.GetUserID(r.Context())
	var input model.CreateAnnouncementInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if input.Title == "" || input.Body == "" {
		http.Error(w, `{"error":"title_and_body_required"}`, http.StatusBadRequest)
		return
	}
	a, err := h.repo.Create(r.Context(), adminID, input)
	if err != nil {
		http.Error(w, `{"error":"create_failed"}`, http.StatusInternalServerError)
		return
	}
	go h.notifyVendors(context.Background(), a)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"announcement": a})
}

// notifyVendors — notification in-app + push à tous les vendeurs, en tâche
// de fond (ne doit jamais retarder la réponse à l'admin qui vient de
// publier). Best-effort comme le reste du système de notifications (voir
// WebhookHandler.notify) : une erreur d'envoi pour un vendeur n'empêche pas
// les autres.
func (h *AnnouncementHandler) notifyVendors(ctx context.Context, a *model.Announcement) {
	vendorIDs, err := h.userRepo.ListVendorIDs(ctx)
	if err != nil {
		log.Printf("announcement notify: liste vendeurs échouée: %v", err)
		return
	}
	imageURL := h.imageURL(a)
	for _, vendorID := range vendorIDs {
		if h.notificationRepo != nil {
			h.notificationRepo.Create(ctx, vendorID, "announcement", a.Title, a.Body, "/vendor")
		}
		if h.pushSvc != nil {
			h.pushSvc.NotifyUserWithImage(ctx, vendorID, a.Title, a.Body, "/vendor", "announcement", imageURL)
		}
	}
}

// imageURL — URL publique de l'image jointe (voir Image ci-dessous), vide si
// aucune image ou stockage non configuré.
func (h *AnnouncementHandler) imageURL(a *model.Announcement) string {
	if a.ImageKey == nil || *a.ImageKey == "" || h.apiURL == "" {
		return ""
	}
	return h.apiURL + "/api/announcements/" + a.ID + "/image"
}

// Image — GET /api/announcements/{id}/image (public, pas d'auth) : sert
// l'image jointe en direct, même logique que ProductHandler.Cover (le
// Content-Type doit être forcé, l'objet S3 n'en porte aucun à l'upload).
func (h *AnnouncementHandler) Image(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	items, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	var key string
	for _, a := range items {
		if a.ID == id && a.ImageKey != nil {
			key = *a.ImageKey
			break
		}
	}
	if key == "" {
		http.Error(w, `{"error":"no_image"}`, http.StatusNotFound)
		return
	}
	if h.storage == nil {
		http.Error(w, `{"error":"storage_not_configured"}`, http.StatusServiceUnavailable)
		return
	}
	data, err := h.storage.Download(r.Context(), key)
	if err != nil {
		http.Error(w, `{"error":"image_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", imageContentType(key, data))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(data)
}

// List — GET /api/admin/announcements (vue admin, pour gérer/supprimer) ET
// GET /api/vendor/announcements (vue vendeur, lecture seule) — même contenu,
// deux routes distinctes pour ne pas exposer les routes admin (/api/admin/*)
// à un compte vendeur non-admin.
func (h *AnnouncementHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context())
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"announcements": items})
}

// Delete — DELETE /api/admin/announcements/{id}.
func (h *AnnouncementHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.repo.Delete(r.Context(), id); err != nil {
		http.Error(w, `{"error":"delete_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
