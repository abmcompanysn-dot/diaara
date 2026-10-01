package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// meta_ads.go — client Meta Marketing API (Graph API) pour la sponsorisation
// de produits DIARRA sur Facebook + Instagram (demande du 2026-10-01).
//
// DIARRA crée les campagnes sur SON PROPRE compte publicitaire (Business
// Manager DIARRA), avec le jeton d'un "utilisateur système" ayant la
// permission ads_management ; le vendeur ne se connecte jamais à Meta. Les
// pubs sont publiées au nom de la page Facebook DIARRA et renvoient vers la
// page produit DIARRA.
//
// Ordre de création, pensé pour qu'aucun échec partiel ne dépense d'argent :
// campagne PAUSED -> ensemble de pubs (budget total, dates, pays) -> visuel
// -> pub, puis activation de la campagne EN DERNIER. En cas d'échec à une
// étape, la campagne déjà créée est supprimée (best-effort) : sans campagne
// active, Meta ne dépense rien.

type MetaAdsConfig struct {
	AccessToken string // jeton utilisateur système (ads_management, pages_read_engagement)
	AppSecret   string // optionnel : signe chaque appel (appsecret_proof), recommandé par Meta
	AdAccountID string // "act_123..." ou "123..."
	PageID      string // page Facebook au nom de laquelle les pubs sont publiées
	// InstagramUserID — optionnel : compte Instagram professionnel relié à la
	// page. Sans lui, Meta diffuse sur Instagram via le compte relié à la page
	// s'il existe.
	InstagramUserID string
	// Currency — devise du compte publicitaire : "EUR" (défaut), "USD" ou
	// "XOF". Le budget payé en FCFA est converti dans cette devise.
	Currency     string
	GraphVersion string // ex "v23.0"
	BaseURL      string // tests uniquement ; défaut https://graph.facebook.com
}

type MetaAdsClient struct {
	cfg    MetaAdsConfig
	client *http.Client
}

func NewMetaAdsClient(cfg MetaAdsConfig) *MetaAdsClient {
	if !strings.HasPrefix(cfg.AdAccountID, "act_") {
		cfg.AdAccountID = "act_" + cfg.AdAccountID
	}
	cfg.Currency = strings.ToUpper(strings.TrimSpace(cfg.Currency))
	if cfg.Currency == "" {
		cfg.Currency = "EUR"
	}
	if cfg.GraphVersion == "" {
		cfg.GraphVersion = "v23.0"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://graph.facebook.com"
	}
	cfg.BaseURL = strings.TrimSuffix(cfg.BaseURL, "/")
	return &MetaAdsClient{cfg: cfg, client: &http.Client{Timeout: 30 * time.Second}}
}

// MetaAPIError — erreur renvoyée par la Graph API. UserMessage (en français
// si Meta le fournit) est affichable tel quel au vendeur.
type MetaAPIError struct {
	StatusCode  int
	Code        int
	Subcode     int
	Message     string
	UserMessage string
}

func (e *MetaAPIError) Error() string {
	return fmt.Sprintf("meta api %d (code %d/%d): %s", e.StatusCode, e.Code, e.Subcode, e.Message)
}

// XOF par EUR : parité fixe du franc CFA (exacte, pas un taux de marché).
const xofPerEUR = 655.957

// ToAccountMinorUnits convertit un montant FCFA dans la plus petite unité de
// la devise du compte publicitaire, format attendu par Meta pour
// lifetime_budget : centimes pour EUR/USD, unités pour XOF (devise sans
// décimales chez Meta).
func (c *MetaAdsClient) ToAccountMinorUnits(amountXOF int) (int64, error) {
	switch c.cfg.Currency {
	case "XOF":
		return int64(amountXOF), nil
	case "EUR":
		return int64(math.Floor(float64(amountXOF) / xofPerEUR * 100)), nil
	case "USD":
		rate := USDRates["XOF"]
		return int64(math.Floor(float64(amountXOF) / rate * 100)), nil
	}
	return 0, fmt.Errorf("devise du compte publicitaire non supportée: %s", c.cfg.Currency)
}

// FromAccountAmount convertit un montant Meta en devise du compte, format
// "12.34" (insights "spend"), vers des FCFA arrondis.
func (c *MetaAdsClient) FromAccountAmount(amount string) int {
	v, err := strconv.ParseFloat(strings.TrimSpace(amount), 64)
	if err != nil {
		return 0
	}
	switch c.cfg.Currency {
	case "EUR":
		v *= xofPerEUR
	case "USD":
		v *= USDRates["XOF"]
	}
	return int(math.Round(v))
}

type SponsoredAdRequest struct {
	Name      string    // nom interne de la campagne (visible dans le gestionnaire de pubs DIARRA)
	LinkURL   string    // page produit DIARRA
	Headline  string    // titre du produit
	Message   string    // texte de la pub
	ImageURL  string    // couverture du produit (URL publique), optionnel
	Countries []string  // ISO2
	BudgetXOF int       // budget publicitaire total (après commission DIARRA)
	StartTime time.Time // début de diffusion
	EndTime   time.Time // fin de diffusion
}

type SponsoredAdResult struct {
	CampaignID string
	AdSetID    string
	CreativeID string
	AdID       string
}

// CreateSponsoredAd crée campagne + ensemble de pubs + visuel + pub, puis
// active la campagne (voir l'ordre expliqué en tête de fichier).
func (c *MetaAdsClient) CreateSponsoredAd(ctx context.Context, req SponsoredAdRequest) (res *SponsoredAdResult, err error) {
	budget, err := c.ToAccountMinorUnits(req.BudgetXOF)
	if err != nil {
		return nil, err
	}
	if budget <= 0 {
		return nil, errors.New("budget publicitaire trop faible après conversion")
	}

	out := &SponsoredAdResult{}
	defer func() {
		// Échec après la création de la campagne : on la supprime (elle est
		// encore en pause, donc rien n'a pu être dépensé).
		if err != nil && out.CampaignID != "" {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_ = c.post(cleanupCtx, out.CampaignID, url.Values{"status": {"DELETED"}}, nil)
		}
	}()

	var created struct {
		ID string `json:"id"`
	}

	if err = c.post(ctx, c.cfg.AdAccountID+"/campaigns", url.Values{
		"name":                            {req.Name},
		"objective":                       {"OUTCOME_TRAFFIC"},
		"status":                          {"PAUSED"},
		"special_ad_categories":           {"[]"},
		"is_adset_budget_sharing_enabled": {"false"},
	}, &created); err != nil {
		return nil, fmt.Errorf("création campagne: %w", err)
	}
	out.CampaignID = created.ID

	targeting, _ := json.Marshal(map[string]interface{}{
		"geo_locations":        map[string]interface{}{"countries": req.Countries},
		"age_min":              18,
		"publisher_platforms":  []string{"facebook", "instagram"},
		"targeting_automation": map[string]interface{}{"advantage_audience": 1},
	})
	if err = c.post(ctx, c.cfg.AdAccountID+"/adsets", url.Values{
		"name":              {req.Name},
		"campaign_id":       {out.CampaignID},
		"lifetime_budget":   {strconv.FormatInt(budget, 10)},
		"start_time":        {req.StartTime.UTC().Format(time.RFC3339)},
		"end_time":          {req.EndTime.UTC().Format(time.RFC3339)},
		"billing_event":     {"IMPRESSIONS"},
		"optimization_goal": {"LINK_CLICKS"},
		"bid_strategy":      {"LOWEST_COST_WITHOUT_CAP"},
		"destination_type":  {"WEBSITE"},
		"targeting":         {string(targeting)},
		"status":            {"ACTIVE"},
	}, &created); err != nil {
		return nil, fmt.Errorf("création ensemble de pubs: %w", err)
	}
	out.AdSetID = created.ID

	linkData := map[string]interface{}{
		"link":    req.LinkURL,
		"message": req.Message,
		"name":    req.Headline,
		"call_to_action": map[string]interface{}{
			"type":  "SHOP_NOW",
			"value": map[string]string{"link": req.LinkURL},
		},
	}
	if req.ImageURL != "" {
		linkData["picture"] = req.ImageURL
	}
	storySpec := map[string]interface{}{"page_id": c.cfg.PageID, "link_data": linkData}
	if c.cfg.InstagramUserID != "" {
		storySpec["instagram_user_id"] = c.cfg.InstagramUserID
	}
	spec, _ := json.Marshal(storySpec)
	if err = c.post(ctx, c.cfg.AdAccountID+"/adcreatives", url.Values{
		"name":              {req.Name},
		"object_story_spec": {string(spec)},
	}, &created); err != nil {
		return nil, fmt.Errorf("création visuel: %w", err)
	}
	out.CreativeID = created.ID

	creative, _ := json.Marshal(map[string]string{"creative_id": out.CreativeID})
	if err = c.post(ctx, c.cfg.AdAccountID+"/ads", url.Values{
		"name":     {req.Name},
		"adset_id": {out.AdSetID},
		"creative": {string(creative)},
		"status":   {"ACTIVE"},
	}, &created); err != nil {
		return nil, fmt.Errorf("création pub: %w", err)
	}
	out.AdID = created.ID

	if err = c.post(ctx, out.CampaignID, url.Values{"status": {"ACTIVE"}}, nil); err != nil {
		return nil, fmt.Errorf("activation campagne: %w", err)
	}
	return out, nil
}

// AdStatus — statut réel de la pub chez Meta. EffectiveStatus : ACTIVE,
// PAUSED, PENDING_REVIEW, IN_PROCESS, PREAPPROVED, DISAPPROVED, WITH_ISSUES,
// CAMPAIGN_PAUSED, ADSET_PAUSED, ARCHIVED, DELETED...
type AdStatus struct {
	EffectiveStatus string
	ReviewFeedback  string // raison du refus (DISAPPROVED), en clair
}

func (c *MetaAdsClient) GetAdStatus(ctx context.Context, adID string) (*AdStatus, error) {
	var raw struct {
		EffectiveStatus  string                     `json:"effective_status"`
		AdReviewFeedback map[string]json.RawMessage `json:"ad_review_feedback"`
		IssuesInfo       []struct {
			ErrorMessage string `json:"error_message"`
		} `json:"issues_info"`
	}
	if err := c.get(ctx, adID, url.Values{"fields": {"effective_status,ad_review_feedback,issues_info"}}, &raw); err != nil {
		return nil, err
	}
	out := &AdStatus{EffectiveStatus: raw.EffectiveStatus}
	// ad_review_feedback : {"global": {"Règle": "explication", ...}, ...}
	var reasons []string
	for _, v := range raw.AdReviewFeedback {
		var m map[string]string
		if json.Unmarshal(v, &m) == nil {
			for k, msg := range m {
				if msg != "" {
					reasons = append(reasons, k+" : "+msg)
				} else {
					reasons = append(reasons, k)
				}
			}
		}
	}
	for _, i := range raw.IssuesInfo {
		if i.ErrorMessage != "" {
			reasons = append(reasons, i.ErrorMessage)
		}
	}
	out.ReviewFeedback = strings.Join(reasons, " · ")
	return out, nil
}

type AdInsights struct {
	Impressions int64
	Reach       int64
	Clicks      int64 // clics sur le lien (vers la page produit)
	SpendXOF    int
}

// GetCampaignInsights — statistiques cumulées depuis le début de la campagne.
func (c *MetaAdsClient) GetCampaignInsights(ctx context.Context, campaignID string) (*AdInsights, error) {
	var raw struct {
		Data []struct {
			Impressions      string `json:"impressions"`
			Reach            string `json:"reach"`
			InlineLinkClicks string `json:"inline_link_clicks"`
			Spend            string `json:"spend"`
		} `json:"data"`
	}
	if err := c.get(ctx, campaignID+"/insights", url.Values{
		"fields":      {"impressions,reach,inline_link_clicks,spend"},
		"date_preset": {"maximum"},
	}, &raw); err != nil {
		return nil, err
	}
	out := &AdInsights{}
	if len(raw.Data) > 0 {
		d := raw.Data[0]
		out.Impressions, _ = strconv.ParseInt(d.Impressions, 10, 64)
		out.Reach, _ = strconv.ParseInt(d.Reach, 10, 64)
		out.Clicks, _ = strconv.ParseInt(d.InlineLinkClicks, 10, 64)
		out.SpendXOF = c.FromAccountAmount(d.Spend)
	}
	return out, nil
}

// PauseCampaign — arrêt immédiat de la diffusion (admin).
func (c *MetaAdsClient) PauseCampaign(ctx context.Context, campaignID string) error {
	return c.post(ctx, campaignID, url.Values{"status": {"PAUSED"}}, nil)
}

func (c *MetaAdsClient) authParams(v url.Values) url.Values {
	if v == nil {
		v = url.Values{}
	}
	v.Set("access_token", c.cfg.AccessToken)
	if c.cfg.AppSecret != "" {
		mac := hmac.New(sha256.New, []byte(c.cfg.AppSecret))
		mac.Write([]byte(c.cfg.AccessToken))
		v.Set("appsecret_proof", hex.EncodeToString(mac.Sum(nil)))
	}
	return v
}

func (c *MetaAdsClient) endpoint(path string) string {
	return c.cfg.BaseURL + "/" + c.cfg.GraphVersion + "/" + strings.TrimPrefix(path, "/")
}

func (c *MetaAdsClient) post(ctx context.Context, path string, form url.Values, out interface{}) error {
	body := c.authParams(form).Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(path), strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.do(req, out)
}

func (c *MetaAdsClient) get(ctx context.Context, path string, query url.Values, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint(path)+"?"+c.authParams(query).Encode(), nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *MetaAdsClient) do(req *http.Request, out interface{}) error {
	resp, err := c.client.Do(req)
	if err != nil {
		// Ne jamais remonter l'URL (elle contient le jeton d'accès en GET).
		return &MetaAPIError{StatusCode: http.StatusBadGateway, Message: "meta injoignable"}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var e struct {
			Error struct {
				Message   string `json:"message"`
				Code      int    `json:"code"`
				Subcode   int    `json:"error_subcode"`
				UserTitle string `json:"error_user_title"`
				UserMsg   string `json:"error_user_msg"`
			} `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		apiErr := &MetaAPIError{StatusCode: resp.StatusCode, Code: e.Error.Code, Subcode: e.Error.Subcode, Message: e.Error.Message, UserMessage: e.Error.UserMsg}
		if apiErr.Message == "" {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return apiErr
	}
	if out != nil && len(raw) > 0 {
		return json.Unmarshal(raw, out)
	}
	return nil
}
