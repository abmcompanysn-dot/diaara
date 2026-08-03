package handler

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"net/http"

	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
	"github.com/go-chi/chi/v5"
)

const slugCharset = "abcdefghijklmnopqrstuvwxyz0123456789"

// newSlug génère un slug court (8 caractères) lisible, sans caractères ambigus.
func newSlug() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	slug := make([]byte, 8)
	for i, v := range b {
		slug[i] = slugCharset[int(v)%len(slugCharset)]
	}
	return string(slug), nil
}

type CloserHandler struct {
	referralRepo *repository.ReferralRepo
	productRepo  *repository.ProductRepo
	frontendURL  string
}

func NewCloserHandler(
	referralRepo *repository.ReferralRepo,
	productRepo *repository.ProductRepo,
	frontendURL string,
) *CloserHandler {
	return &CloserHandler{
		referralRepo: referralRepo,
		productRepo:  productRepo,
		frontendURL:  frontendURL,
	}
}

// CreateLink — POST /api/closer/links
func (h *CloserHandler) CreateLink(w http.ResponseWriter, r *http.Request) {
	closerID := middleware.GetUserID(r.Context())
	if closerID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var input model.CreateReferralLinkInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if input.ProductID == "" || input.CommissionPct <= 0 {
		http.Error(w, `{"error":"product_and_commission_required"}`, http.StatusBadRequest)
		return
	}

	product, err := h.productRepo.FindByID(r.Context(), input.ProductID)
	if err != nil {
		http.Error(w, `{"error":"product_not_found"}`, http.StatusNotFound)
		return
	}

	if product.ModerationStatus != "approved" {
		http.Error(w, `{"error":"product_not_available"}`, http.StatusBadRequest)
		return
	}
	if !product.AffiliateEnabled {
		http.Error(w, `{"error":"affiliate_disabled"}`, http.StatusBadRequest)
		return
	}

	// Anti-fraude : un closer ne promeut pas son propre produit.
	if product.VendorID == closerID {
		http.Error(w, `{"error":"cannot_promote_own_product"}`, http.StatusBadRequest)
		return
	}

	// Commission plafonnée par le vendeur (max_closer_commission_pct).
	if input.CommissionPct > product.MaxCloserCommissionPct {
		http.Error(w, `{"error":"commission_exceeds_cap"}`, http.StatusBadRequest)
		return
	}

	// Slug unique avec retry en cas de collision.
	for attempt := 0; attempt < 5; attempt++ {
		slug, err := newSlug()
		if err != nil {
			http.Error(w, `{"error":"slug_generation_failed"}`, http.StatusInternalServerError)
			return
		}
		available, err := h.referralRepo.SlugAvailable(r.Context(), slug)
		if err != nil {
			http.Error(w, `{"error":"slug_check_failed"}`, http.StatusInternalServerError)
			return
		}
		if available {
			input.Slug = slug
			break
		}
	}
	if input.Slug == "" {
		http.Error(w, `{"error":"slug_generation_failed"}`, http.StatusInternalServerError)
		return
	}

	link, err := h.referralRepo.Create(r.Context(), closerID, input)
	if err != nil {
		if err == repository.ErrReferralExists {
			http.Error(w, `{"error":"link_already_exists"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"link_creation_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"link": link})
}

// ListLinks — GET /api/closer/links
func (h *CloserHandler) ListLinks(w http.ResponseWriter, r *http.Request) {
	closerID := middleware.GetUserID(r.Context())
	if closerID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	links, err := h.referralRepo.ListByCloser(r.Context(), closerID)
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"links": links})
}

// Redirect — GET /r/{slug} (public). Compte le clic puis redirige vers la
// fiche produit du frontend.
func (h *CloserHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	link, err := h.referralRepo.FindBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, `{"error":"link_not_found"}`, http.StatusNotFound)
		return
	}

	// Le comptage de clics ne doit pas bloquer la redirection.
	go func() {
		_ = h.referralRepo.IncrementClicks(context.Background(), link.ID)
	}()

	target := "/product?id=" + link.ProductID
	if h.frontendURL != "" {
		target = h.frontendURL + "/product?id=" + link.ProductID
	}
	http.Redirect(w, r, target, http.StatusFound)
}
