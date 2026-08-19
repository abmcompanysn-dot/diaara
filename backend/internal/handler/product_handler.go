package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/diarra/backend/internal/cache"
	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/preview"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/service"
	"github.com/diarra/backend/internal/storage"
	"github.com/go-chi/chi/v5"
)

type ProductHandler struct {
	productRepo *repository.ProductRepo
	userRepo    *repository.UserRepo
	saleRepo    *repository.SaleRepo
	storage     StorageService
	frontendURL string
	cache       *cache.Client
}

// StorageService est l'interface minimale dont le handler a besoin.
type StorageService interface {
	Upload(ctx context.Context, key string, data []byte) error
	Download(ctx context.Context, key string) ([]byte, error)
	GenerateSignedURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

func NewProductHandler(productRepo *repository.ProductRepo, userRepo *repository.UserRepo, saleRepo *repository.SaleRepo, storage StorageService, frontendURL string, cacheClient *cache.Client) *ProductHandler {
	return &ProductHandler{productRepo: productRepo, userRepo: userRepo, saleRepo: saleRepo, storage: storage, frontendURL: frontendURL, cache: cacheClient}
}

// Shop — GET /api/vendors/{id}/shop (public). Boutique publique d'un
// vendeur : son nom de boutique + ses produits approuvés, pour la page
// partageable via QR code depuis son espace vendeur.
func (h *ProductHandler) Shop(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	vendor, err := h.userRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}

	products, err := h.productRepo.ListApprovedByVendor(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	// Niveau/trophée basé sur le chiffre d'affaires cumulé — seul le palier
	// est exposé publiquement, jamais le montant exact.
	tier := ""
	if totalEarned, err := h.saleRepo.TotalEarnedByVendor(r.Context(), id); err == nil {
		tier = model.VendorTier(totalEarned)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"vendor": map[string]interface{}{
			"id":           vendor.ID,
			"display_name": vendor.DisplayName,
			"shop_name":    vendor.ShopName,
			"tier":         tier,
		},
		"products": products,
	})
}

// List — public, produits approuvés
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	search := r.URL.Query().Get("search")

	limit := 50
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	// Page la plus visitée du site : mise en cache 60s. Pas d'invalidation à
	// l'approbation d'un nouveau produit — délai volontairement accepté pour
	// rester simple, le catalogue n'est pas temps-réel critique.
	cacheKey := fmt.Sprintf("catalog:%s:%s:%d:%d", category, search, limit, offset)
	var products []*model.Product
	hit, _ := h.cache.GetJSON(r.Context(), cacheKey, &products)
	if !hit {
		var err error
		products, err = h.productRepo.ListApproved(r.Context(), category, search, limit, offset)
		if err != nil {
			http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
			return
		}
		h.cache.SetJSON(r.Context(), cacheKey, products, 60*time.Second)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"products": products})
}

// ListVendor — vendeur authentifié, ses propres produits
func (h *ProductHandler) ListVendor(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	products, err := h.productRepo.FindByVendor(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"list_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"products": products})
}

// Get — public, fiche détaillée
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	product, err := h.productRepo.FindByID(r.Context(), id)
	if err != nil {
		if err == repository.ErrProductNotFound {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error":"get_failed"}`, http.StatusInternalServerError)
		return
	}

	// Un produit non approuvé n'est visible que par son vendeur ou un admin
	if product.ModerationStatus != "approved" {
		userID := middleware.GetUserID(r.Context())
		isAdmin := middleware.GetIsAdmin(r.Context())
		if userID != product.VendorID && !isAdmin {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"product": product})
}

// Upload — vendeur authentifié, fichier multipart → R2
func (h *ProductHandler) Upload(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if h.storage == nil {
		http.Error(w, `{"error":"storage_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	if err := r.ParseMultipartForm(50 << 20); err != nil { // 50MB max
		http.Error(w, `{"error":"file_too_large"}`, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"file_required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, `{"error":"read_failed"}`, http.StatusInternalServerError)
		return
	}

	key := storage.NewFileKey(userID, header.Filename)
	// Champ "type" : "cover" → image de couverture (préfixe covers/), sinon fichier vendu.
	if r.FormValue("type") == "cover" {
		if !validCoverImage(data) {
			http.Error(w, `{"error":"invalid_cover_image_type"}`, http.StatusBadRequest)
			return
		}
		data = compressCoverImage(data)
		key = "covers/" + key
	}
	if err := h.storage.Upload(r.Context(), key, data); err != nil {
		http.Error(w, `{"error":"upload_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"file_key": key,
		"size":     strconv.Itoa(len(data)),
	})
}

// Cover — public. Sert directement les octets de l'image de couverture
// (plutôt qu'une redirection vers une URL signée) : Upload() n'enregistre
// aucun Content-Type sur l'objet S3/MinIO, donc une redirection expose le
// type générique renvoyé par le stockage — les navigateurs l'ignorent et
// devinent quand même, mais les robots WhatsApp/Facebook le respectent
// strictement et refusent d'afficher l'image dans la carte de partage si ce
// n'est pas "image/...". Servir en direct permet de forcer le bon type.
func (h *ProductHandler) Cover(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	product, err := h.productRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if product.CoverImageKey == nil || *product.CoverImageKey == "" {
		http.Error(w, `{"error":"no_cover"}`, http.StatusNotFound)
		return
	}

	if h.storage == nil {
		http.Error(w, `{"error":"storage_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	data, err := h.storage.Download(r.Context(), *product.CoverImageKey)
	if err != nil {
		http.Error(w, `{"error":"cover_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", imageContentType(*product.CoverImageKey, data))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(data)
}

// imageContentType devine le type MIME d'une image à partir de son
// extension (rapide, couvre le cas normal) puis, à défaut, en reniflant les
// premiers octets du fichier (couvre les clés sans extension exploitable).
func imageContentType(key string, data []byte) string {
	if ct := mime.TypeByExtension(filepath.Ext(key)); ct != "" {
		return ct
	}
	return http.DetectContentType(data)
}

// allowedCoverImageTypes — liste blanche appliquée à tout upload servant de
// couverture produit. SVG exclu volontairement : un SVG peut embarquer du
// <script>, exécuté si l'image est ouverte en navigation directe ou via
// <object>/<iframe> (le Content-Type est déterminé par imageContentType à
// partir de l'extension du fichier stocké — un SVG téléversé serait donc
// resservi en image/svg+xml et son script exécuté dans l'origine DIARRA).
var allowedCoverImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,
}

// validCoverImage sniffe le contenu réel du fichier (pas l'extension ni le
// Content-Type déclaré par le client) et rejette tout ce qui n'est pas un
// des formats raster autorisés.
func validCoverImage(data []byte) bool {
	return allowedCoverImageTypes[http.DetectContentType(data)]
}

// Preview — public (produits approuvés) ou vendeur/admin (produit en
// attente). Redirige vers une URL signée (1 h) de l'aperçu filigrané
// d'indice donné (0 = premier).
func (h *ProductHandler) Preview(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	idxStr := chi.URLParam(r, "index")
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 {
		http.Error(w, `{"error":"invalid_index"}`, http.StatusBadRequest)
		return
	}

	product, err := h.productRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if product.ModerationStatus != "approved" {
		userID := middleware.GetUserID(r.Context())
		isAdmin := middleware.GetIsAdmin(r.Context())
		if userID != product.VendorID && !isAdmin {
			http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
			return
		}
	}
	if idx >= len(product.PreviewKeys) {
		http.Error(w, `{"error":"no_preview"}`, http.StatusNotFound)
		return
	}

	if h.storage == nil {
		http.Error(w, `{"error":"storage_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	data, err := h.storage.Download(r.Context(), product.PreviewKeys[idx])
	if err != nil {
		http.Error(w, `{"error":"preview_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", imageContentType(product.PreviewKeys[idx], data))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(data)
}

// socialCrawlerUA reconnaît les robots des réseaux sociaux qui n'exécutent pas
// de JavaScript et ont donc besoin des balises Open Graph directement dans le HTML.
var socialCrawlerUA = []string{
	"facebookexternalhit", "twitterbot", "whatsapp", "telegrambot",
	"linkedinbot", "slackbot", "discordbot", "pinterest", "skypeuripreview",
}

func isSocialCrawler(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	for _, needle := range socialCrawlerUA {
		if strings.Contains(ua, needle) {
			return true
		}
	}
	return false
}

// Share — GET /p/{id}, lien public à partager (WhatsApp, Facebook, etc.).
// Sert une carte Open Graph aux robots des réseaux sociaux, redirige les
// visiteurs humains vers la fiche produit réelle du frontend.
func (h *ProductHandler) Share(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	product, err := h.productRepo.FindByID(r.Context(), id)
	if err != nil || product.ModerationStatus != "approved" {
		http.NotFound(w, r)
		return
	}

	productRef := product.Slug
	if productRef == "" {
		productRef = id
	}
	productURL := fmt.Sprintf("%s/product?id=%s", strings.TrimSuffix(h.frontendURL, "/"), productRef)

	if !isSocialCrawler(r.Header.Get("User-Agent")) {
		http.Redirect(w, r, productURL, http.StatusFound)
		return
	}

	scheme := "https"
	if r.Header.Get("X-Forwarded-Proto") != "" {
		scheme = r.Header.Get("X-Forwarded-Proto")
	} else if r.TLS == nil {
		scheme = "http"
	}
	shareURL := fmt.Sprintf("%s://%s/p/%s", scheme, r.Host, id)

	// WhatsApp/Facebook n'ont pas d'emplacement dédié pour product:price:*
	// dans un aperçu de lien classique (ces balises servent à leurs systèmes
	// de catalogue, pas à l'aperçu affiché en discussion) — le prix doit
	// donc être visible dans le texte de la description pour apparaître.
	description := ""
	if product.Description != nil {
		description = *product.Description
	}
	if len(description) > 180 {
		description = description[:180] + "…"
	}
	priceLine := fmt.Sprintf("%d FCFA", product.PriceCFA)
	if description == "" {
		description = fmt.Sprintf("%s — produit numérique sur DIARRA", priceLine)
	} else {
		description = fmt.Sprintf("%s — %s", priceLine, description)
	}

	var imageTag string
	if product.CoverImageKey != nil && *product.CoverImageKey != "" {
		imageURL := fmt.Sprintf("%s://%s/api/products/%s/cover", scheme, r.Host, id)
		imageTag = fmt.Sprintf(`<meta property="og:image" content="%s">`, html.EscapeString(imageURL))
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="fr" prefix="og: https://ogp.me/ns# product: https://ogp.me/ns/product#">
<head>
<meta charset="utf-8">
<title>%s</title>
<meta property="og:type" content="product">
<meta property="og:site_name" content="DIARRA">
<meta property="og:title" content="%s">
<meta property="og:description" content="%s">
<meta property="og:url" content="%s">
<meta property="product:price:amount" content="%d">
<meta property="product:price:currency" content="XOF">
%s
<meta name="twitter:card" content="summary_large_image">
<meta http-equiv="refresh" content="0; url=%s">
</head>
<body></body>
</html>`,
		html.EscapeString(product.Title),
		html.EscapeString(product.Title),
		html.EscapeString(description),
		html.EscapeString(shareURL),
		product.PriceCFA,
		imageTag,
		html.EscapeString(productURL),
	)
}

// Create — vendeur authentifié
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var input model.CreateProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if input.Title == "" || input.PriceCFA < 0 || input.Category == "" || input.FileKey == "" {
		http.Error(w, `{"error":"title_price_category_file_required"}`, http.StatusBadRequest)
		return
	}

	if !validAffiliateConfig(input.AffiliateEnabled, input.MaxCloserCommissionPct) {
		http.Error(w, `{"error":"invalid_affiliate_config"}`, http.StatusBadRequest)
		return
	}

	if input.PriceMode == "flexible" && (input.MinPriceCFA == nil || *input.MinPriceCFA <= 0) {
		http.Error(w, `{"error":"min_price_required_for_flexible_pricing"}`, http.StatusBadRequest)
		return
	}
	if input.PriceMode != "" && input.PriceMode != "fixed" && input.PriceMode != "flexible" {
		http.Error(w, `{"error":"invalid_price_mode"}`, http.StatusBadRequest)
		return
	}

	product, err := h.productRepo.Create(r.Context(), input, userID)
	if err != nil {
		http.Error(w, `{"error":"creation_failed"}`, http.StatusInternalServerError)
		return
	}

	if h.storage != nil {
		go h.generatePreview(userID, product.ID, product.FileKey)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"product": product})
}

// AutoCreate — POST /api/admin/products/auto (admin authentifié uniquement).
// Endpoint pensé pour une automatisation externe (script / IA) qui génère le
// texte de la fiche (et éventuellement un prompt d'image) sans forcément
// avoir de fichier prêt tout de suite. Le fichier est optionnel : s'il n'est
// pas fourni, le produit est créé avec le prompt d'image renseigné, et un
// modérateur l'attachera plus tard via AttachFile avant d'approuver. S'il
// est fourni, il sert à la fois de couverture et de fichier livré (vente de
// visuels/templates où l'image EST le produit). Dans tous les cas, le
// produit créé passe par la modération normale (moderation_status =
// "pending"), comme n'importe quel produit.
//
// Champs multipart attendus :
//   - file          : optionnel, le fichier (image/template)
//   - image_prompt  : optionnel (mais requis si "file" est absent), prompt à
//     donner à un outil de génération d'image pour créer le visuel plus tard
//   - vendor_id     : optionnel, compte vendeur cible (défaut : l'admin appelant)
//   - title         : requis
//   - description   : optionnel
//   - price_cfa     : requis, entier
//   - category      : requis (voir CATEGORY_LABELS côté frontend pour les valeurs valides)
func (h *ProductHandler) AutoCreate(w http.ResponseWriter, r *http.Request) {
	// adminID est vide quand l'appel est authentifié par clé d'automatisation
	// (voir middleware.RequireAutomation) plutôt que par session admin — dans
	// ce cas vendor_id doit être fourni explicitement dans la requête.
	adminID := middleware.GetUserID(r.Context())

	if err := r.ParseMultipartForm(50 << 20); err != nil { // 50MB max
		http.Error(w, `{"error":"file_too_large"}`, http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	description := r.FormValue("description")
	category := r.FormValue("category")
	priceStr := r.FormValue("price_cfa")
	imagePrompt := r.FormValue("image_prompt")
	vendorID := r.FormValue("vendor_id")
	if vendorID == "" {
		vendorID = adminID
	}
	if vendorID == "" {
		http.Error(w, `{"error":"vendor_id_required"}`, http.StatusBadRequest)
		return
	}

	priceCFA, err := strconv.Atoi(priceStr)
	if title == "" || category == "" || err != nil || priceCFA < 0 {
		http.Error(w, `{"error":"title_price_category_required"}`, http.StatusBadRequest)
		return
	}

	input := model.CreateProductInput{
		Title:    title,
		Category: category,
		PriceCFA: priceCFA,
	}
	if description != "" {
		input.Description = &description
	}

	file, header, ferr := r.FormFile("file")
	if ferr == nil {
		defer file.Close()
		if h.storage == nil {
			http.Error(w, `{"error":"storage_not_configured"}`, http.StatusServiceUnavailable)
			return
		}
		data, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, `{"error":"read_failed"}`, http.StatusInternalServerError)
			return
		}
		key := storage.NewFileKey(vendorID, header.Filename)
		if err := h.storage.Upload(r.Context(), key, data); err != nil {
			http.Error(w, `{"error":"upload_failed"}`, http.StatusInternalServerError)
			return
		}
		coverKey := "covers/" + key
		if err := h.storage.Upload(r.Context(), coverKey, compressCoverImage(data)); err != nil {
			http.Error(w, `{"error":"upload_failed"}`, http.StatusInternalServerError)
			return
		}
		input.FileKey = key
		input.CoverImageKey = coverKey
	} else if imagePrompt != "" {
		input.ImagePrompt = &imagePrompt
	} else {
		http.Error(w, `{"error":"file_or_image_prompt_required"}`, http.StatusBadRequest)
		return
	}

	product, err := h.productRepo.Create(r.Context(), input, vendorID)
	if err != nil {
		http.Error(w, `{"error":"creation_failed"}`, http.StatusInternalServerError)
		return
	}

	if product.FileKey != "" && h.storage != nil {
		go h.generatePreview(vendorID, product.ID, product.FileKey)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"product": product})
}

// AttachFile — PUT /api/admin/products/{id}/file (admin authentifié
// uniquement). Deuxième étape pour un produit créé sans fichier via
// AutoCreate (avec un image_prompt) : le modérateur génère l'image à partir
// du prompt puis l'attache ici, avant d'approuver le produit via Moderate.
// Le fichier envoyé sert à la fois de couverture et de fichier livré.
func (h *ProductHandler) AttachFile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if h.storage == nil {
		http.Error(w, `{"error":"storage_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	product, err := h.productRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}

	if err := r.ParseMultipartForm(50 << 20); err != nil { // 50MB max
		http.Error(w, `{"error":"file_too_large"}`, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"file_required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, `{"error":"read_failed"}`, http.StatusInternalServerError)
		return
	}

	// AttachFile ne sert que la seconde étape du flux image_prompt (voir
	// commentaire de fonction) : le fichier envoyé est toujours censé être
	// l'image générée, jamais un livrable arbitraire — validation appliquée.
	if !validCoverImage(data) {
		http.Error(w, `{"error":"invalid_cover_image_type"}`, http.StatusBadRequest)
		return
	}

	key := storage.NewFileKey(product.VendorID, header.Filename)
	if err := h.storage.Upload(r.Context(), key, data); err != nil {
		http.Error(w, `{"error":"upload_failed"}`, http.StatusInternalServerError)
		return
	}
	coverKey := "covers/" + key
	if err := h.storage.Upload(r.Context(), coverKey, compressCoverImage(data)); err != nil {
		http.Error(w, `{"error":"upload_failed"}`, http.StatusInternalServerError)
		return
	}

	updated, err := h.productRepo.Update(r.Context(), id, model.UpdateProductInput{
		FileKey:       &key,
		CoverImageKey: &coverKey,
	})
	if err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}

	go h.generatePreview(product.VendorID, id, key)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"product": updated})
}

// UpdateAutomation — PUT /api/automation/products/{id}. Modifie les champs
// d'un produit déjà créé via l'automatisation (titre, prix, description,
// catégorie...) — mêmes règles qu'une modification vendeur normale : un
// produit déjà approuvé repasse en attente (le contenu a changé, il doit
// être revalidé avant de redevenir visible).
func (h *ProductHandler) UpdateAutomation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	product, err := h.productRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}

	var input model.UpdateProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	if input.PriceMode != nil && *input.PriceMode == "flexible" {
		minVal := product.MinPriceCFA
		if input.MinPriceCFA != nil {
			minVal = input.MinPriceCFA
		}
		if minVal == nil || *minVal <= 0 {
			http.Error(w, `{"error":"min_price_required_for_flexible_pricing"}`, http.StatusBadRequest)
			return
		}
	}

	updated, err := h.productRepo.Update(r.Context(), id, input)
	if err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}

	if product.ModerationStatus == "approved" {
		if err := h.productRepo.UpdateModerationStatus(r.Context(), id, "pending", nil); err != nil {
			http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
			return
		}
		updated.ModerationStatus = "pending"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"product": updated})
}

// UpdateAutomationCover — PUT /api/automation/products/{id}/cover. Change
// l'image de couverture indépendamment du fichier livré à l'acheteur
// (contrairement à AttachFile, qui définit les deux depuis le même upload) —
// utile pour remplacer juste le visuel marketing sans toucher au fichier vendu.
func (h *ProductHandler) UpdateAutomationCover(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if h.storage == nil {
		http.Error(w, `{"error":"storage_not_configured"}`, http.StatusServiceUnavailable)
		return
	}

	product, err := h.productRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}

	if err := r.ParseMultipartForm(50 << 20); err != nil { // 50MB max
		http.Error(w, `{"error":"file_too_large"}`, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error":"file_required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, `{"error":"read_failed"}`, http.StatusInternalServerError)
		return
	}

	if !validCoverImage(data) {
		http.Error(w, `{"error":"invalid_cover_image_type"}`, http.StatusBadRequest)
		return
	}

	coverKey := "covers/" + storage.NewFileKey(product.VendorID, header.Filename)
	if err := h.storage.Upload(r.Context(), coverKey, compressCoverImage(data)); err != nil {
		http.Error(w, `{"error":"upload_failed"}`, http.StatusInternalServerError)
		return
	}

	updated, err := h.productRepo.Update(r.Context(), id, model.UpdateProductInput{
		CoverImageKey: &coverKey,
	})
	if err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}

	if product.ModerationStatus == "approved" {
		if err := h.productRepo.UpdateModerationStatus(r.Context(), id, "pending", nil); err != nil {
			http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
			return
		}
		updated.ModerationStatus = "pending"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"product": updated})
}

// generatePreview construit les aperçus filigranés en tâche de fond (peut
// prendre de quelques secondes à ~1 min selon le fichier) puis enregistre
// le résultat. Appelé après la réponse HTTP, sans bloquer le vendeur.
// RecoverStuckPreviews relance la génération d'aperçu pour les produits
// restés bloqués en "pending" (interrompus par un redéploiement pendant la
// génération — voir ListPendingPreviews). Appelé une fois au démarrage du
// serveur.
func (h *ProductHandler) RecoverStuckPreviews(ctx context.Context) {
	if h.storage == nil {
		return
	}
	products, err := h.productRepo.ListPendingPreviews(ctx)
	if err != nil {
		log.Printf("preview: échec de la liste des aperçus bloqués: %v", err)
		return
	}
	if len(products) == 0 {
		return
	}
	log.Printf("preview: relance de %d aperçu(s) bloqué(s)", len(products))
	for _, p := range products {
		go h.generatePreview(p.VendorID, p.ID, p.FileKey)
	}
}

func (h *ProductHandler) generatePreview(vendorID, productID, fileKey string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	keys, err := preview.Generate(ctx, h.storage, vendorID, productID, fileKey)
	if keys == nil {
		keys = []string{}
	}
	status := "ready"
	switch {
	case err != nil:
		log.Printf("preview: échec génération pour produit %s: %v", productID, err)
		status = "failed"
	case len(keys) == 0:
		status = "unsupported"
	}
	if err := h.productRepo.SetPreview(context.Background(), productID, keys, status); err != nil {
		log.Printf("preview: échec enregistrement pour produit %s: %v", productID, err)
	}
}

// Update — vendeur propriétaire
func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")

	product, err := h.productRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if product.VendorID != userID {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	var input model.UpdateProductInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	// Si le statut d'affiliation est modifié, vérifier la cohérence du plafond.
	if input.AffiliateEnabled != nil {
		capVal := product.MaxCloserCommissionPct
		if input.MaxCloserCommissionPct != nil {
			capVal = *input.MaxCloserCommissionPct
		}
		if !validAffiliateConfig(*input.AffiliateEnabled, capVal) {
			http.Error(w, `{"error":"invalid_affiliate_config"}`, http.StatusBadRequest)
			return
		}
	}

	if input.PriceMode != nil && *input.PriceMode == "flexible" {
		minVal := product.MinPriceCFA
		if input.MinPriceCFA != nil {
			minVal = input.MinPriceCFA
		}
		if minVal == nil || *minVal <= 0 {
			http.Error(w, `{"error":"min_price_required_for_flexible_pricing"}`, http.StatusBadRequest)
			return
		}
	}

	updated, err := h.productRepo.Update(r.Context(), id, input)
	if err != nil {
		http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
		return
	}

	// Le fichier livré a changé : les aperçus filigranés précédents ne
	// correspondent plus, on les régénère en tâche de fond (même mécanisme
	// qu'à la création).
	if input.FileKey != nil && h.storage != nil {
		go h.generatePreview(userID, id, *input.FileKey)
	}

	// Un produit déjà approuvé repasse en attente après modification : le
	// contenu a changé, il doit être revalidé avant de redevenir visible.
	if product.ModerationStatus == "approved" {
		if err := h.productRepo.UpdateModerationStatus(r.Context(), id, "pending", nil); err != nil {
			http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
			return
		}
		updated.ModerationStatus = "pending"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"product": updated})
}

// Delete — vendeur propriétaire ou admin. Un vendeur ne supprime jamais
// directement : sa demande est marquée deletion_requested et attend la
// confirmation d'un admin (voir AdminHandler.ConfirmDeletion/CancelDeletion).
func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.GetIsAdmin(r.Context())
	id := chi.URLParam(r, "id")

	product, err := h.productRepo.FindByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	if product.VendorID != userID && !isAdmin {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	if isAdmin {
		if err := h.productRepo.Delete(r.Context(), id); err != nil {
			http.Error(w, `{"error":"delete_failed"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
		return
	}

	if err := h.productRepo.RequestDeletion(r.Context(), id); err != nil {
		http.Error(w, `{"error":"delete_failed"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deletion_requested"})
}

// validAffiliateConfig vérifie la cohérence des paramètres d'affiliation :
// si le produit est ouvert à l'affiliation, le plafond doit être strictement
// positif et ne pas dépasser la part vendeur (100% - 15% plateforme).
func validAffiliateConfig(enabled bool, maxPct float64) bool {
	if !enabled {
		return true
	}
	return maxPct > 0 && maxPct <= (100-float64(service.DefaultPlatformFeePct))
}
