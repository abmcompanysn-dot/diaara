package handler

import (
	"context"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
)

// FeedHandler expose les flux produits pour les plateformes publicitaires
// (Google Merchant Center, catalogue Meta/Facebook Ads). Ce sont des biens
// numériques (pas d'expédition ni de stock physique) : availability et
// condition sont donc fixes plutôt que dérivées d'un état d'inventaire.
type FeedHandler struct {
	productRepo *repository.ProductRepo
	frontendURL string
}

func NewFeedHandler(productRepo *repository.ProductRepo, frontendURL string) *FeedHandler {
	return &FeedHandler{productRepo: productRepo, frontendURL: frontendURL}
}

// allApprovedProducts récupère la totalité des produits approuvés en
// paginant en interne — contrairement à l'API publique /api/products, un
// flux doit lister le catalogue complet, pas une seule page.
func (h *FeedHandler) allApprovedProducts(ctx context.Context) ([]*model.Product, error) {
	var all []*model.Product
	const limit = 200
	offset := 0
	for {
		batch, err := h.productRepo.ListApproved(ctx, "", "", limit, offset)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < limit {
			break
		}
		offset += limit
	}
	return all, nil
}

func requestOrigin(r *http.Request) string {
	scheme := "https"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS == nil {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

// productLink construit l'URL publique d'un produit, en préférant le slug
// lisible à l'UUID technique (retombe sur l'ID si le slug n'est pas encore
// renseigné — fenêtre de migration).
func productLink(frontendURL string, p *model.Product) string {
	ref := p.Slug
	if ref == "" {
		ref = p.ID
	}
	return fmt.Sprintf("%s/product?id=%s", strings.TrimSuffix(frontendURL, "/"), ref)
}

func productDescription(p *model.Product) string {
	if p.Description != nil && *p.Description != "" {
		return *p.Description
	}
	return p.Title
}

// --- Google Merchant Center (flux RSS 2.0 + espace de noms g:) ---

type merchantFeed struct {
	XMLName xml.Name        `xml:"rss"`
	Version string          `xml:"version,attr"`
	XmlnsG  string          `xml:"xmlns:g,attr"`
	Channel merchantChannel `xml:"channel"`
}

type merchantChannel struct {
	Title       string         `xml:"title"`
	Link        string         `xml:"link"`
	Description string         `xml:"description"`
	Items       []merchantItem `xml:"item"`
}

type merchantItem struct {
	ID               string `xml:"g:id"`
	Title            string `xml:"title"`
	Description      string `xml:"description"`
	Link             string `xml:"link"`
	ImageLink        string `xml:"g:image_link,omitempty"`
	Availability     string `xml:"g:availability"`
	Price            string `xml:"g:price"`
	Condition        string `xml:"g:condition"`
	Brand            string `xml:"g:brand"`
	ProductType      string `xml:"g:product_type,omitempty"`
	IdentifierExists string `xml:"g:identifier_exists"`
}

// GoogleMerchant — GET /feed/google-merchant.xml (public). Flux produits au
// format attendu par Google Merchant Center (Content API / import par URL).
func (h *FeedHandler) GoogleMerchant(w http.ResponseWriter, r *http.Request) {
	products, err := h.allApprovedProducts(r.Context())
	if err != nil {
		http.Error(w, `{"error":"feed_failed"}`, http.StatusInternalServerError)
		return
	}

	origin := requestOrigin(r)
	items := make([]merchantItem, 0, len(products))
	for _, p := range products {
		var imageLink string
		if p.CoverImageKey != nil && *p.CoverImageKey != "" {
			imageLink = fmt.Sprintf("%s/api/products/%s/cover", origin, p.ID)
		}
		items = append(items, merchantItem{
			ID:               p.ID,
			Title:            p.Title,
			Description:      productDescription(p),
			Link:             productLink(h.frontendURL, p),
			ImageLink:        imageLink,
			Availability:     "in stock",
			Price:            fmt.Sprintf("%d XOF", p.PriceCFA),
			Condition:        "new",
			Brand:            "DIARRA",
			ProductType:      p.Category,
			IdentifierExists: "no",
		})
	}

	feed := merchantFeed{
		Version: "2.0",
		XmlnsG:  "http://base.google.com/ns/1.0",
		Channel: merchantChannel{
			Title:       "DIARRA — Catalogue produits",
			Link:        strings.TrimSuffix(h.frontendURL, "/"),
			Description: "Flux produits DIARRA pour Google Merchant Center",
			Items:       items,
		},
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=1800")
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(feed); err != nil {
		log.Printf("feed: échec encodage flux Google Merchant: %v", err)
	}
}

// --- sitemap.xml (produits + boutiques, avec extension image) ---

type sitemapURLSet struct {
	XMLName    xml.Name     `xml:"urlset"`
	Xmlns      string       `xml:"xmlns,attr"`
	XmlnsImage string       `xml:"xmlns:image,attr"`
	URLs       []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc        string         `xml:"loc"`
	LastMod    string         `xml:"lastmod,omitempty"`
	ChangeFreq string         `xml:"changefreq,omitempty"`
	Priority   string         `xml:"priority,omitempty"`
	Images     []sitemapImage `xml:"image:image"`
}

type sitemapImage struct {
	Loc string `xml:"image:loc"`
}

// Sitemap — GET /sitemap.xml (public). Générée à la demande côté backend
// plutôt qu'au build du frontend statique : depuis le conteneur de build,
// un fetch vers le propre domaine public du VPS échoue silencieusement
// (hairpin NAT — même souci que la connexion SMTP sortante, voir main.go),
// ce qui produisait un sitemap sans aucun produit/boutique.
func (h *FeedHandler) Sitemap(w http.ResponseWriter, r *http.Request) {
	products, err := h.allApprovedProducts(r.Context())
	if err != nil {
		http.Error(w, `{"error":"sitemap_failed"}`, http.StatusInternalServerError)
		return
	}

	base := strings.TrimSuffix(h.frontendURL, "/")
	origin := requestOrigin(r)

	urls := []sitemapURL{
		{Loc: base + "/", ChangeFreq: "daily", Priority: "1.0"},
		{Loc: base + "/catalog", ChangeFreq: "daily", Priority: "0.9"},
		{Loc: base + "/how-it-works", ChangeFreq: "monthly", Priority: "0.5"},
		{Loc: base + "/sell", ChangeFreq: "monthly", Priority: "0.6"},
		{Loc: base + "/faq", ChangeFreq: "monthly", Priority: "0.5"},
		{Loc: base + "/mentions-legales", ChangeFreq: "yearly", Priority: "0.1"},
		{Loc: base + "/confidentialite", ChangeFreq: "yearly", Priority: "0.1"},
		{Loc: base + "/cgu", ChangeFreq: "yearly", Priority: "0.1"},
	}

	seenVendor := map[string]bool{}
	vendorOrder := []string{}
	vendorImage := map[string]string{}

	for _, p := range products {
		var pImages []sitemapImage
		var pImageURL string
		if p.CoverImageKey != nil && *p.CoverImageKey != "" {
			pImageURL = fmt.Sprintf("%s/api/products/%s/cover", origin, p.ID)
			pImages = []sitemapImage{{Loc: pImageURL}}
		}
		urls = append(urls, sitemapURL{
			Loc:        productLink(h.frontendURL, p),
			LastMod:    p.UpdatedAt.UTC().Format(time.RFC3339),
			ChangeFreq: "weekly",
			Priority:   "0.8",
			Images:     pImages,
		})

		if !seenVendor[p.VendorID] {
			seenVendor[p.VendorID] = true
			vendorOrder = append(vendorOrder, p.VendorID)
		}
		if pImageURL != "" {
			if _, ok := vendorImage[p.VendorID]; !ok {
				vendorImage[p.VendorID] = pImageURL
			}
		}
	}

	// Une entrée par boutique (vendeur), avec la couverture d'un de ses
	// produits comme image représentative (pas de photo de profil dédiée
	// côté vendeur).
	for _, vid := range vendorOrder {
		var images []sitemapImage
		if img, ok := vendorImage[vid]; ok {
			images = []sitemapImage{{Loc: img}}
		}
		urls = append(urls, sitemapURL{
			Loc:        fmt.Sprintf("%s/boutique?id=%s", base, vid),
			ChangeFreq: "weekly",
			Priority:   "0.6",
			Images:     images,
		})
	}

	feed := sitemapURLSet{
		Xmlns:      "http://www.sitemaps.org/schemas/sitemap/0.9",
		XmlnsImage: "http://www.google.com/schemas/sitemap-image/1.1",
		URLs:       urls,
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=1800")
	w.Write([]byte(xml.Header))
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(feed); err != nil {
		log.Printf("feed: échec encodage sitemap: %v", err)
	}
}

// Facebook — GET /feed/facebook.csv (public). Flux au format CSV attendu
// par le catalogue Meta/Facebook Ads (import par URL dans Commerce Manager).
func (h *FeedHandler) Facebook(w http.ResponseWriter, r *http.Request) {
	products, err := h.allApprovedProducts(r.Context())
	if err != nil {
		http.Error(w, `{"error":"feed_failed"}`, http.StatusInternalServerError)
		return
	}

	origin := requestOrigin(r)

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=1800")
	w.Header().Set("Content-Disposition", `inline; filename="diarra-facebook-catalog.csv"`)

	writer := csv.NewWriter(w)
	defer writer.Flush()

	writer.Write([]string{"id", "title", "description", "availability", "condition", "price", "link", "image_link", "brand"})
	for _, p := range products {
		var imageLink string
		if p.CoverImageKey != nil && *p.CoverImageKey != "" {
			imageLink = fmt.Sprintf("%s/api/products/%s/cover", origin, p.ID)
		}
		writer.Write([]string{
			p.ID,
			p.Title,
			productDescription(p),
			"in stock",
			"new",
			fmt.Sprintf("%d XOF", p.PriceCFA),
			productLink(h.frontendURL, p),
			imageLink,
			"DIARRA",
		})
	}
}
