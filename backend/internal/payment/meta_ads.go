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
// de produits DIARRA sur Facebook + Instagram.
//
// Modèle (décision du propriétaire, 2026-10-01) : le VENDEUR fait la pub sur
// SA PROPRE page Facebook, avec SON PROPRE compte publicitaire, et c'est lui
// qui paie Meta (moyen de paiement de son compte publicitaire). DIARRA n'est
// qu'une interface simple : le vendeur connecte son compte Facebook (Facebook
// Login, voir meta_oauth.go), choisit sa page et son compte pub, puis DIARRA
// crée la campagne avec SON jeton. Aucun argent ne transite par DIARRA.
//
// MetaApp porte la configuration commune (app Meta DIARRA : id, secret,
// version Graph) ; MetaAdsClient est construit PAR VENDEUR (MetaApp.Client)
// avec son jeton, son compte pub, sa page et la devise de son compte. Le
// secret de l'app reste commun et signe chaque appel (appsecret_proof).
//
// Ordre de création, pensé pour qu'aucun échec partiel ne dépense d'argent :
// campagne PAUSED -> ensemble de pubs (budget total, dates, pays) -> visuel
// -> pub, puis activation de la campagne EN DERNIER. En cas d'échec à une
// étape, la campagne déjà créée est supprimée (best-effort) : sans campagne
// active, Meta ne dépense rien.
//
// Sécurité : en GET, le jeton part dans l'URL (format Graph). Les erreurs
// renvoyées ne contiennent JAMAIS l'URL ni le jeton (voir do), et ce fichier
// ne logue rien.

type MetaAppConfig struct {
	AppID        string
	AppSecret    string // secret de l'app : échange OAuth, appsecret_proof, signature du state
	RedirectURL  string // URL publique du callback OAuth (backend), déclarée dans l'app Meta
	GraphVersion string // ex "v23.0"
	BaseURL      string // tests uniquement ; défaut https://graph.facebook.com
	DialogURL    string // tests uniquement ; défaut https://www.facebook.com
}

type MetaApp struct {
	cfg      MetaAppConfig
	client   *http.Client
	stateKey []byte
}

func NewMetaApp(cfg MetaAppConfig) *MetaApp {
	if cfg.GraphVersion == "" {
		cfg.GraphVersion = "v23.0"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://graph.facebook.com"
	}
	if cfg.DialogURL == "" {
		cfg.DialogURL = "https://www.facebook.com"
	}
	cfg.BaseURL = strings.TrimSuffix(cfg.BaseURL, "/")
	cfg.DialogURL = strings.TrimSuffix(cfg.DialogURL, "/")
	// Clé dédiée à la signature du state OAuth, dérivée du secret de l'app :
	// jamais le secret lui-même pour deux usages différents.
	mac := hmac.New(sha256.New, []byte(cfg.AppSecret))
	mac.Write([]byte("diarra/meta-oauth-state/v1"))
	return &MetaApp{cfg: cfg, client: &http.Client{Timeout: 30 * time.Second}, stateKey: mac.Sum(nil)}
}

// MetaAccount — ce qui distingue un vendeur : son jeton (longue durée) et ses
// choix (compte pub "act_...", page, devise du compte pub). Les champs autres
// que AccessToken peuvent être vides pour les appels "utilisateur" (liste des
// pages / comptes pub).
type MetaAccount struct {
	AccessToken string
	AdAccountID string
	PageID      string
	Currency    string
}

type MetaAdsClient struct {
	app *MetaApp
	acc MetaAccount
}

// Client construit le client d'un vendeur.
func (a *MetaApp) Client(acc MetaAccount) *MetaAdsClient {
	if acc.AdAccountID != "" && !strings.HasPrefix(acc.AdAccountID, "act_") {
		acc.AdAccountID = "act_" + acc.AdAccountID
	}
	acc.Currency = strings.ToUpper(strings.TrimSpace(acc.Currency))
	return &MetaAdsClient{app: a, acc: acc}
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

// IsMetaTokenError — jeton expiré, révoqué ou invalidé (changement de mot de
// passe, app retirée par l'utilisateur...) : code OAuth 190 (ou 102, session
// invalide). Le vendeur doit reconnecter son compte Facebook.
func IsMetaTokenError(err error) bool {
	var e *MetaAPIError
	return errors.As(err, &e) && (e.Code == 190 || e.Code == 102)
}

// --- Devises -------------------------------------------------------------------

// XOF par EUR : parité fixe du franc CFA (exacte, pas un taux de marché).
const xofPerEUR = 655.957

// ErrMetaCurrencyUnsupported — devise du compte publicitaire que DIARRA ne
// sait pas convertir depuis le FCFA.
var ErrMetaCurrencyUnsupported = errors.New("devise du compte publicitaire non prise en charge")

// MetaSupportedCurrencies — devises de compte publicitaire acceptées : les
// francs CFA (sans conversion), l'euro (parité fixe) et le dollar (USDRates).
var MetaSupportedCurrencies = []string{"XOF", "XAF", "EUR", "USD"}

func MetaCurrencySupported(currency string) bool {
	c := strings.ToUpper(strings.TrimSpace(currency))
	for _, s := range MetaSupportedCurrencies {
		if s == c {
			return true
		}
	}
	return false
}

// XOFToMetaMinorUnits convertit un montant FCFA dans la plus petite unité de
// la devise du compte publicitaire, format attendu par Meta pour
// lifetime_budget : centimes pour EUR/USD, unités pour XOF/XAF (devises sans
// décimales chez Meta). Arrondi à l'inférieur : jamais plus que le budget
// saisi par le vendeur.
func XOFToMetaMinorUnits(amountXOF int, currency string) (int64, error) {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "XOF", "XAF":
		return int64(amountXOF), nil
	case "EUR":
		return int64(math.Floor(float64(amountXOF) / xofPerEUR * 100)), nil
	case "USD":
		return int64(math.Floor(float64(amountXOF) / USDRates["XOF"] * 100)), nil
	}
	return 0, ErrMetaCurrencyUnsupported
}

// MetaAmountToXOF convertit un montant Meta en devise du compte, format
// "12.34" (insights "spend"), vers des FCFA arrondis.
func MetaAmountToXOF(amount, currency string) int {
	v, err := strconv.ParseFloat(strings.TrimSpace(amount), 64)
	if err != nil {
		return 0
	}
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case "EUR":
		v *= xofPerEUR
	case "USD":
		v *= USDRates["XOF"]
	}
	return int(math.Round(v))
}

func (c *MetaAdsClient) ToAccountMinorUnits(amountXOF int) (int64, error) {
	return XOFToMetaMinorUnits(amountXOF, c.acc.Currency)
}

func (c *MetaAdsClient) FromAccountAmount(amount string) int {
	return MetaAmountToXOF(amount, c.acc.Currency)
}

// --- Compte de l'utilisateur : profil, pages, comptes publicitaires ----------

type MetaUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (c *MetaAdsClient) Me(ctx context.Context) (*MetaUser, error) {
	var u MetaUser
	if err := c.get(ctx, "me", url.Values{"fields": {"id,name"}}, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// GrantedPermissions — permissions effectivement accordées par l'utilisateur
// (il peut en décocher dans la fenêtre Facebook).
func (c *MetaAdsClient) GrantedPermissions(ctx context.Context) (map[string]bool, error) {
	var raw struct {
		Data []struct {
			Permission string `json:"permission"`
			Status     string `json:"status"`
		} `json:"data"`
	}
	if err := c.get(ctx, "me/permissions", nil, &raw); err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, p := range raw.Data {
		if p.Status == "granted" {
			out[p.Permission] = true
		}
	}
	return out, nil
}

type MetaPage struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListPages — pages Facebook que l'utilisateur gère (/me/accounts).
func (c *MetaAdsClient) ListPages(ctx context.Context) ([]MetaPage, error) {
	var raw struct {
		Data []MetaPage `json:"data"`
	}
	if err := c.get(ctx, "me/accounts", url.Values{"fields": {"id,name"}, "limit": {"200"}}, &raw); err != nil {
		return nil, err
	}
	if raw.Data == nil {
		raw.Data = []MetaPage{}
	}
	return raw.Data, nil
}

type MetaAdAccount struct {
	ID            string `json:"id"` // "act_123"
	Name          string `json:"name"`
	Currency      string `json:"currency"`
	AccountStatus int    `json:"account_status"` // 1 = actif ; 2 désactivé, 3 impayé, 7/8/9... voir doc Meta
}

// Usable — seul un compte actif (account_status = 1) peut diffuser.
func (a MetaAdAccount) Usable() bool { return a.AccountStatus == 1 }

// ListAdAccounts — comptes publicitaires accessibles (/me/adaccounts).
func (c *MetaAdsClient) ListAdAccounts(ctx context.Context) ([]MetaAdAccount, error) {
	var raw struct {
		Data []MetaAdAccount `json:"data"`
	}
	if err := c.get(ctx, "me/adaccounts", url.Values{"fields": {"name,currency,account_status"}, "limit": {"200"}}, &raw); err != nil {
		return nil, err
	}
	if raw.Data == nil {
		raw.Data = []MetaAdAccount{}
	}
	return raw.Data, nil
}

// RevokePermissions — retire l'autorisation donnée à l'app DIARRA
// (déconnexion). Best-effort côté appelant.
func (c *MetaAdsClient) RevokePermissions(ctx context.Context) error {
	return c.call(ctx, http.MethodDelete, "me/permissions", nil, nil)
}

// --- Campagnes -----------------------------------------------------------------

type SponsoredAdRequest struct {
	Name      string    // nom interne de la campagne (visible dans le gestionnaire de pubs du vendeur)
	LinkURL   string    // page produit DIARRA
	Headline  string    // titre du produit
	Message   string    // texte de la pub
	ImageURL  string    // couverture du produit (URL publique), optionnel
	Countries []string  // ISO2
	BudgetXOF int       // budget publicitaire total saisi par le vendeur (FCFA)
	StartTime time.Time // début de diffusion
	EndTime   time.Time // fin de diffusion
}

type SponsoredAdResult struct {
	CampaignID string
	AdSetID    string
	CreativeID string
	AdID       string
}

// CreateSponsoredAd crée campagne + ensemble de pubs + visuel + pub sur le
// compte publicitaire du vendeur, au nom de sa page, puis active la campagne
// (voir l'ordre expliqué en tête de fichier).
func (c *MetaAdsClient) CreateSponsoredAd(ctx context.Context, req SponsoredAdRequest) (res *SponsoredAdResult, err error) {
	if c.acc.AdAccountID == "" || c.acc.PageID == "" {
		return nil, errors.New("compte publicitaire ou page non choisi")
	}
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

	if err = c.post(ctx, c.acc.AdAccountID+"/campaigns", url.Values{
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
	if err = c.post(ctx, c.acc.AdAccountID+"/adsets", url.Values{
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
	// Pas d'instagram_user_id : Meta diffuse sur Instagram via le compte
	// Instagram relié à la page du vendeur s'il existe.
	spec, _ := json.Marshal(map[string]interface{}{"page_id": c.acc.PageID, "link_data": linkData})
	if err = c.post(ctx, c.acc.AdAccountID+"/adcreatives", url.Values{
		"name":              {req.Name},
		"object_story_spec": {string(spec)},
	}, &created); err != nil {
		return nil, fmt.Errorf("création visuel: %w", err)
	}
	out.CreativeID = created.ID

	creative, _ := json.Marshal(map[string]string{"creative_id": out.CreativeID})
	if err = c.post(ctx, c.acc.AdAccountID+"/ads", url.Values{
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

// PauseCampaign — arrêt immédiat de la diffusion (vendeur ou admin).
func (c *MetaAdsClient) PauseCampaign(ctx context.Context, campaignID string) error {
	return c.post(ctx, campaignID, url.Values{"status": {"PAUSED"}}, nil)
}

// --- Transport -----------------------------------------------------------------

func (c *MetaAdsClient) authParams(v url.Values) url.Values {
	if v == nil {
		v = url.Values{}
	}
	v.Set("access_token", c.acc.AccessToken)
	if c.app.cfg.AppSecret != "" {
		mac := hmac.New(sha256.New, []byte(c.app.cfg.AppSecret))
		mac.Write([]byte(c.acc.AccessToken))
		v.Set("appsecret_proof", hex.EncodeToString(mac.Sum(nil)))
	}
	return v
}

func (c *MetaAdsClient) post(ctx context.Context, path string, form url.Values, out interface{}) error {
	return c.call(ctx, http.MethodPost, path, form, out)
}

func (c *MetaAdsClient) get(ctx context.Context, path string, query url.Values, out interface{}) error {
	return c.call(ctx, http.MethodGet, path, query, out)
}

// call — POST : paramètres dans le corps ; GET/DELETE : dans l'URL.
func (c *MetaAdsClient) call(ctx context.Context, method, path string, params url.Values, out interface{}) error {
	return c.app.do(ctx, method, path, c.authParams(params), out)
}

func (a *MetaApp) endpoint(path string) string {
	return a.cfg.BaseURL + "/" + a.cfg.GraphVersion + "/" + strings.TrimPrefix(path, "/")
}

func (a *MetaApp) do(ctx context.Context, method, path string, params url.Values, out interface{}) error {
	var req *http.Request
	var err error
	if method == http.MethodPost {
		req, err = http.NewRequestWithContext(ctx, method, a.endpoint(path), strings.NewReader(params.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	} else {
		req, err = http.NewRequestWithContext(ctx, method, a.endpoint(path)+"?"+params.Encode(), nil)
	}
	if err != nil {
		// L'erreur de construction peut citer l'URL (donc le jeton).
		return &MetaAPIError{StatusCode: http.StatusBadGateway, Message: "requête meta invalide"}
	}
	resp, err := a.client.Do(req)
	if err != nil {
		// Ne jamais remonter l'URL (elle contient le jeton d'accès en GET).
		return &MetaAPIError{StatusCode: http.StatusBadGateway, Message: "meta injoignable"}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return &MetaAPIError{StatusCode: http.StatusBadGateway, Message: "réponse meta illisible"}
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
