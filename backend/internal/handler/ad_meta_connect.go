package handler

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/diarra/backend/internal/middleware"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/repository"
)

// ad_meta_connect.go — connexion du compte Facebook du vendeur (Facebook
// Login) et choix de sa page / de son compte publicitaire. Voir
// payment/meta_oauth.go pour le flux OAuth.
//
// Anti-CSRF : le state signé contient l'id du vendeur ET un nonce, aussi posé
// dans un cookie httpOnly sur le navigateur qui a demandé la connexion. Au
// retour, les deux doivent correspondre : un lien de connexion fabriqué par
// quelqu'un d'autre (state valide mais navigateur différent) ne peut pas
// rattacher le compte Facebook d'une victime au compte DIARRA d'un tiers.

const (
	metaOAuthCookie     = "meta_oauth_nonce"
	metaOAuthCookiePath = "/api/meta/oauth"
	metaOAuthStateTTL   = 10 * time.Minute
)

// GetMeta — GET /api/vendor/meta : état de la connexion (jamais le jeton).
func (h *AdHandler) GetMeta(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"configured":           h.configured(),
		"enabled":              h.enabled(r.Context()),
		"connected":            false,
		"ready":                false,
		"connection":           nil,
		"supported_currencies": payment.MetaSupportedCurrencies,
	}
	conn, err := h.conns.Get(r.Context(), middleware.GetUserID(r.Context()))
	if err != nil && !errors.Is(err, repository.ErrMetaConnectionNotFound) {
		http.Error(w, `{"error":"meta_status_failed"}`, http.StatusInternalServerError)
		return
	}
	if conn != nil {
		resp["connected"] = true
		resp["ready"] = conn.Ready()
		resp["connection"] = conn
	}
	adJSON(w, http.StatusOK, resp)
}

// Connect — GET /api/vendor/meta/connect : URL de la fenêtre Facebook.
func (h *AdHandler) Connect(w http.ResponseWriter, r *http.Request) {
	if !h.enabled(r.Context()) {
		http.Error(w, `{"error":"ads_unavailable"}`, http.StatusServiceUnavailable)
		return
	}
	state, nonce, err := h.meta.SignState(middleware.GetUserID(r.Context()), time.Now(), metaOAuthStateTTL)
	if err != nil {
		http.Error(w, `{"error":"meta_connect_failed"}`, http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     metaOAuthCookie,
		Value:    nonce,
		Path:     metaOAuthCookiePath,
		HttpOnly: true,
		Secure:   h.secureCookie,
		// Lax : envoyé lors du retour depuis facebook.com (navigation GET de
		// premier niveau), jamais sur une requête cross-site en arrière-plan.
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(metaOAuthStateTTL.Seconds()),
	})
	adJSON(w, http.StatusOK, map[string]string{"url": h.meta.DialogURL(state)})
}

// OAuthCallback — GET /api/meta/oauth/callback (PUBLIC, appelé par le
// navigateur au retour de Facebook). Termine toujours par une redirection
// vers /vendor/ads?meta=connected ou ?meta=error&reason=...
func (h *AdHandler) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	// Le nonce ne sert qu'une fois.
	http.SetCookie(w, &http.Cookie{
		Name: metaOAuthCookie, Value: "", Path: metaOAuthCookiePath,
		HttpOnly: true, Secure: h.secureCookie, SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
	fail := func(reason string) {
		http.Redirect(w, r, h.frontendURL+"/vendor/ads?meta=error&reason="+url.QueryEscape(reason), http.StatusFound)
	}
	if !h.configured() {
		fail("unavailable")
		return
	}
	q := r.URL.Query()
	st, err := h.meta.VerifyState(q.Get("state"), time.Now())
	if errors.Is(err, payment.ErrMetaStateExpired) {
		fail("expired")
		return
	}
	if err != nil {
		fail("state")
		return
	}
	cookie, err := r.Cookie(metaOAuthCookie)
	if err != nil || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(st.Nonce)) != 1 {
		fail("state")
		return
	}
	// Le vendeur a cliqué « Annuler » / refusé (error=access_denied).
	if q.Get("error") != "" {
		fail("denied")
		return
	}
	code := q.Get("code")
	if code == "" {
		fail("denied")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	reason, err := h.completeOAuth(ctx, st.VendorID, code)
	if err != nil {
		log.Printf("ads: connexion Facebook vendeur=%s: %v", st.VendorID, err)
		fail(reason)
		return
	}
	http.Redirect(w, r, h.frontendURL+"/vendor/ads?meta=connected", http.StatusFound)
}

// completeOAuth : code -> jeton court -> jeton longue durée -> vérification
// des permissions -> profil -> stockage chiffré. reason = code court pour
// l'URL de retour en cas d'erreur.
func (h *AdHandler) completeOAuth(ctx context.Context, vendorID, code string) (reason string, err error) {
	short, err := h.meta.ExchangeCode(ctx, code)
	if err != nil {
		return "exchange", err
	}
	long, err := h.meta.ExchangeLongLived(ctx, short.AccessToken)
	if err != nil {
		return "exchange", err
	}
	client := h.meta.Client(payment.MetaAccount{AccessToken: long.AccessToken})
	perms, err := client.GrantedPermissions(ctx)
	if err != nil {
		return "exchange", err
	}
	if !perms["ads_management"] {
		return "permissions", errors.New("permission ads_management non accordée")
	}
	me, err := client.Me(ctx)
	if err != nil {
		return "exchange", err
	}
	enc, err := h.box.Encrypt(long.AccessToken)
	if err != nil {
		return "internal", err
	}
	if err := h.conns.Upsert(ctx, vendorID, me.ID, me.Name, enc, long.ExpiresAt); err != nil {
		return "internal", err
	}
	return "", nil
}

// loadConn — connexion utilisable du vendeur, ou réponse d'erreur écrite.
func (h *AdHandler) loadConn(w http.ResponseWriter, r *http.Request) (*model.VendorMetaConnection, bool) {
	if !h.configured() {
		http.Error(w, `{"error":"ads_unavailable"}`, http.StatusServiceUnavailable)
		return nil, false
	}
	conn, err := h.conns.Get(r.Context(), middleware.GetUserID(r.Context()))
	if errors.Is(err, repository.ErrMetaConnectionNotFound) {
		http.Error(w, `{"error":"meta_not_connected"}`, http.StatusConflict)
		return nil, false
	}
	if err != nil {
		http.Error(w, `{"error":"meta_status_failed"}`, http.StatusInternalServerError)
		return nil, false
	}
	if conn.NeedsReconnect {
		http.Error(w, `{"error":"meta_reconnect_required"}`, http.StatusConflict)
		return nil, false
	}
	return conn, true
}

type metaAssets struct {
	pages    []payment.MetaPage
	accounts []payment.MetaAdAccount
}

// fetchAssets — pages et comptes publicitaires accessibles avec le jeton du
// vendeur. Erreur de jeton : connexion marquée "à reconnecter".
func (h *AdHandler) fetchAssets(ctx context.Context, conn *model.VendorMetaConnection) (*metaAssets, error) {
	client, err := h.clientFor(ctx, conn, "", "", "")
	if err != nil {
		return nil, err
	}
	pages, err := client.ListPages(ctx)
	if err != nil {
		h.handleTokenError(ctx, conn.VendorID, err)
		return nil, err
	}
	accounts, err := client.ListAdAccounts(ctx)
	if err != nil {
		h.handleTokenError(ctx, conn.VendorID, err)
		return nil, err
	}
	return &metaAssets{pages: pages, accounts: accounts}, nil
}

func (h *AdHandler) writeAssetsError(w http.ResponseWriter, err error) {
	if payment.IsMetaTokenError(err) || errors.Is(err, errDecrypt) {
		http.Error(w, `{"error":"meta_reconnect_required"}`, http.StatusConflict)
		return
	}
	log.Printf("ads: lecture pages/comptes Meta: %v", err)
	adJSON(w, http.StatusBadGateway, map[string]interface{}{"error": "meta_assets_failed", "details": metaErrorReason(err)})
}

// Assets — GET /api/vendor/meta/assets : pages et comptes publicitaires du
// vendeur. Seuls les comptes actifs (account_status = 1) dans une devise
// prise en charge sont utilisables.
func (h *AdHandler) Assets(w http.ResponseWriter, r *http.Request) {
	conn, ok := h.loadConn(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	assets, err := h.fetchAssets(ctx, conn)
	if err != nil {
		h.writeAssetsError(w, err)
		return
	}
	accounts := make([]map[string]interface{}, 0, len(assets.accounts))
	for _, a := range assets.accounts {
		supported := payment.MetaCurrencySupported(a.Currency)
		accounts = append(accounts, map[string]interface{}{
			"id":                 a.ID,
			"name":               a.Name,
			"currency":           strings.ToUpper(a.Currency),
			"account_status":     a.AccountStatus,
			"active":             a.Usable(),
			"currency_supported": supported,
			"usable":             a.Usable() && supported,
		})
	}
	adJSON(w, http.StatusOK, map[string]interface{}{"pages": assets.pages, "ad_accounts": accounts})
}

// SetMeta — PUT /api/vendor/meta {page_id, ad_account_id} : choix de la page
// et du compte publicitaire, vérifiés auprès de Meta pour CE jeton (un
// vendeur ne peut pas choisir une page ou un compte qu'il ne gère pas).
func (h *AdHandler) SetMeta(w http.ResponseWriter, r *http.Request) {
	conn, ok := h.loadConn(w, r)
	if !ok {
		return
	}
	var input struct {
		PageID      string `json:"page_id"`
		AdAccountID string `json:"ad_account_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.PageID == "" || input.AdAccountID == "" {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(input.AdAccountID, "act_") {
		input.AdAccountID = "act_" + input.AdAccountID
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	assets, err := h.fetchAssets(ctx, conn)
	if err != nil {
		h.writeAssetsError(w, err)
		return
	}
	var page *payment.MetaPage
	for i := range assets.pages {
		if assets.pages[i].ID == input.PageID {
			page = &assets.pages[i]
		}
	}
	if page == nil {
		http.Error(w, `{"error":"meta_page_not_found"}`, http.StatusBadRequest)
		return
	}
	var account *payment.MetaAdAccount
	for i := range assets.accounts {
		if assets.accounts[i].ID == input.AdAccountID {
			account = &assets.accounts[i]
		}
	}
	if account == nil {
		http.Error(w, `{"error":"meta_ad_account_not_found"}`, http.StatusBadRequest)
		return
	}
	if !account.Usable() {
		http.Error(w, `{"error":"meta_ad_account_inactive"}`, http.StatusBadRequest)
		return
	}
	currency := strings.ToUpper(account.Currency)
	if !payment.MetaCurrencySupported(currency) {
		adJSON(w, http.StatusBadRequest, map[string]interface{}{"error": "meta_currency_unsupported", "currency": currency})
		return
	}
	if err := h.conns.SetSelection(r.Context(), conn.VendorID, page.ID, page.Name, account.ID, account.Name, currency); err != nil {
		http.Error(w, `{"error":"meta_status_failed"}`, http.StatusInternalServerError)
		return
	}
	h.GetMeta(w, r)
}

// DeleteMeta — DELETE /api/vendor/meta : déconnexion. L'autorisation donnée
// à l'app DIARRA est retirée chez Meta (best-effort) et le jeton effacé. Les
// pubs déjà en cours continuent chez Meta (c'est le compte du vendeur) :
// l'interface le signale avant confirmation.
func (h *AdHandler) DeleteMeta(w http.ResponseWriter, r *http.Request) {
	vendorID := middleware.GetUserID(r.Context())
	conn, err := h.conns.Get(r.Context(), vendorID)
	if errors.Is(err, repository.ErrMetaConnectionNotFound) {
		adJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	if err != nil {
		http.Error(w, `{"error":"meta_status_failed"}`, http.StatusInternalServerError)
		return
	}
	if h.configured() && !conn.NeedsReconnect {
		// Déchiffrement direct (pas clientFor) : un jeton illisible ne doit
		// pas déclencher de notification « reconnectez-vous » ici.
		if token, err := h.box.Decrypt(conn.AccessTokenEnc); err == nil {
			client := h.meta.Client(payment.MetaAccount{AccessToken: token})
			ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
			if err := client.RevokePermissions(ctx); err != nil {
				log.Printf("ads: révocation Meta vendeur=%s: %v", vendorID, err)
			}
			cancel()
		}
	}
	if err := h.conns.Delete(r.Context(), vendorID); err != nil {
		http.Error(w, `{"error":"meta_disconnect_failed"}`, http.StatusInternalServerError)
		return
	}
	adJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
