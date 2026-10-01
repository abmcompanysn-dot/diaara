package payment

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// meta_oauth.go — Facebook Login (OAuth) pour que le vendeur connecte SON
// compte Facebook à DIARRA (voir meta_ads.go pour le modèle).
//
// Flux :
//  1. GET /api/vendor/meta/connect (vendeur connecté) : DIARRA signe un state
//     (id vendeur + expiration courte + nonce, HMAC avec une clé dérivée du
//     secret de l'app) et renvoie l'URL de la fenêtre Facebook ;
//  2. Facebook rappelle GET /api/meta/oauth/callback?code=...&state=... ;
//  3. DIARRA vérifie le state, échange le code contre un jeton court
//     (ExchangeCode), puis contre un jeton longue durée ~60 jours
//     (ExchangeLongLived), et le stocke chiffré.

// MetaOAuthScopes — permissions demandées au vendeur.
var MetaOAuthScopes = []string{
	"ads_management",        // créer / mettre en pause ses campagnes
	"ads_read",              // statistiques
	"pages_show_list",       // lister ses pages
	"pages_read_engagement", // publier une pub au nom de sa page
	"business_management",   // comptes pub / pages détenus via un Business Manager
}

var (
	ErrMetaStateInvalid = errors.New("state OAuth invalide")
	ErrMetaStateExpired = errors.New("state OAuth expiré")
)

// MetaOAuthState — contenu vérifié d'un state.
type MetaOAuthState struct {
	VendorID string
	Nonce    string // à comparer au cookie posé sur le navigateur (anti-CSRF)
	Expires  time.Time
}

// SignState fabrique un state signé valable ttl : base64url(payload) + "." +
// base64url(HMAC-SHA256(payload)), payload = "vendorID|expUnix|nonce".
func (a *MetaApp) SignState(vendorID string, now time.Time, ttl time.Duration) (state, nonce string, err error) {
	if vendorID == "" || strings.Contains(vendorID, "|") {
		return "", "", ErrMetaStateInvalid
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	nonce = hex.EncodeToString(b)
	payload := vendorID + "|" + strconv.FormatInt(now.Add(ttl).Unix(), 10) + "|" + nonce
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." +
		base64.RawURLEncoding.EncodeToString(a.stateMAC(payload)), nonce, nil
}

// VerifyState vérifie la signature (temps constant) puis l'expiration.
func (a *MetaApp) VerifyState(state string, now time.Time) (*MetaOAuthState, error) {
	parts := strings.Split(state, ".")
	if len(parts) != 2 {
		return nil, ErrMetaStateInvalid
	}
	payload, err1 := base64.RawURLEncoding.DecodeString(parts[0])
	sig, err2 := base64.RawURLEncoding.DecodeString(parts[1])
	if err1 != nil || err2 != nil || !hmac.Equal(sig, a.stateMAC(string(payload))) {
		return nil, ErrMetaStateInvalid
	}
	fields := strings.Split(string(payload), "|")
	if len(fields) != 3 || fields[0] == "" || fields[2] == "" {
		return nil, ErrMetaStateInvalid
	}
	exp, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return nil, ErrMetaStateInvalid
	}
	out := &MetaOAuthState{VendorID: fields[0], Nonce: fields[2], Expires: time.Unix(exp, 0)}
	if !now.Before(out.Expires) {
		return nil, ErrMetaStateExpired
	}
	return out, nil
}

func (a *MetaApp) stateMAC(payload string) []byte {
	mac := hmac.New(sha256.New, a.stateKey)
	mac.Write([]byte(payload))
	return mac.Sum(nil)
}

// DialogURL — fenêtre « Continuer avec Facebook » vers laquelle le
// navigateur du vendeur est envoyé.
func (a *MetaApp) DialogURL(state string) string {
	q := url.Values{
		"client_id":     {a.cfg.AppID},
		"redirect_uri":  {a.cfg.RedirectURL},
		"state":         {state},
		"scope":         {strings.Join(MetaOAuthScopes, ",")},
		"response_type": {"code"},
	}
	return a.cfg.DialogURL + "/" + a.cfg.GraphVersion + "/dialog/oauth?" + q.Encode()
}

// MetaToken — jeton utilisateur. ExpiresAt nil = pas de date communiquée.
type MetaToken struct {
	AccessToken string
	ExpiresAt   *time.Time
}

type metaTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

func (r metaTokenResponse) token(now time.Time) (*MetaToken, error) {
	if r.AccessToken == "" {
		return nil, &MetaAPIError{StatusCode: http.StatusBadGateway, Message: "jeton absent de la réponse meta"}
	}
	t := &MetaToken{AccessToken: r.AccessToken}
	if r.ExpiresIn > 0 {
		exp := now.Add(time.Duration(r.ExpiresIn) * time.Second)
		t.ExpiresAt = &exp
	}
	return t, nil
}

// ExchangeCode échange le code reçu sur le callback contre un jeton court.
func (a *MetaApp) ExchangeCode(ctx context.Context, code string) (*MetaToken, error) {
	var r metaTokenResponse
	if err := a.do(ctx, http.MethodGet, "oauth/access_token", url.Values{
		"client_id":     {a.cfg.AppID},
		"client_secret": {a.cfg.AppSecret},
		"redirect_uri":  {a.cfg.RedirectURL},
		"code":          {code},
	}, &r); err != nil {
		return nil, err
	}
	return r.token(time.Now())
}

// ExchangeLongLived échange un jeton court contre un jeton longue durée
// (~60 jours), grant_type=fb_exchange_token.
func (a *MetaApp) ExchangeLongLived(ctx context.Context, shortToken string) (*MetaToken, error) {
	var r metaTokenResponse
	if err := a.do(ctx, http.MethodGet, "oauth/access_token", url.Values{
		"grant_type":        {"fb_exchange_token"},
		"client_id":         {a.cfg.AppID},
		"client_secret":     {a.cfg.AppSecret},
		"fb_exchange_token": {shortToken},
	}, &r); err != nil {
		return nil, err
	}
	return r.token(time.Now())
}
