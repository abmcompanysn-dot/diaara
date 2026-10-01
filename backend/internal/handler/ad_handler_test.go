package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/diarra/backend/internal/payment"
	"github.com/diarra/backend/internal/secretbox"
)

func TestNormalizeAdCountries(t *testing.T) {
	got := normalizeAdCountries([]string{" sn", "CI", "SN", "XX", ""})
	if strings.Join(got, ",") != "SN,CI" {
		t.Fatalf("pays: %v", got)
	}
	if len(normalizeAdCountries(nil)) != 0 {
		t.Fatal("liste vide attendue")
	}
}

func TestMinAdBudget(t *testing.T) {
	if minAdBudget(7) != 7000 || minAdBudget(1) != 1000 {
		t.Fatalf("minimum: %d / %d", minAdBudget(7), minAdBudget(1))
	}
}

func TestMetaErrorReason(t *testing.T) {
	if r := metaErrorReason(&payment.MetaAPIError{StatusCode: 400, UserMessage: "Texte refusé."}); r != "Texte refusé." {
		t.Fatal(r)
	}
	if r := metaErrorReason(&payment.MetaAPIError{StatusCode: 502}); !strings.Contains(r, "indisponible") {
		t.Fatal(r)
	}
	if r := metaErrorReason(payment.ErrMetaCurrencyUnsupported); !strings.Contains(r, "devise") {
		t.Fatal(r)
	}
	if r := metaErrorReason(errors.New("interne: secret")); strings.Contains(r, "secret") {
		t.Fatal("détail interne exposé")
	}
}

// testAdHandler — handler sans base de données : suffisant pour tous les
// chemins du callback OAuth qui s'arrêtent avant l'enregistrement.
func testAdHandler(t *testing.T, graphURL string) *AdHandler {
	box, err := secretbox.New(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	app := payment.NewMetaApp(payment.MetaAppConfig{
		AppID: "app1", AppSecret: "sec", RedirectURL: "https://api.test/api/meta/oauth/callback", BaseURL: graphURL,
	})
	return NewAdHandler(nil, nil, nil, nil, nil, app, box, "https://diarra.test/", "https://api.test", true)
}

func callback(h *AdHandler, query url.Values, cookie string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/meta/oauth/callback?"+query.Encode(), nil)
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: metaOAuthCookie, Value: cookie})
	}
	rec := httptest.NewRecorder()
	h.OAuthCallback(rec, req)
	return rec
}

func redirectReason(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	if rec.Code != http.StatusFound {
		t.Fatalf("redirection attendue, code %d", rec.Code)
	}
	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil || !strings.HasPrefix(loc.String(), "https://diarra.test/vendor/ads?") {
		t.Fatalf("redirection: %s", rec.Header().Get("Location"))
	}
	if loc.Query().Get("meta") == "connected" {
		return "connected"
	}
	return loc.Query().Get("reason")
}

func TestOAuthCallbackRejectsBadState(t *testing.T) {
	h := testAdHandler(t, "http://127.0.0.1:1")
	now := time.Now()
	state, nonce, _ := h.meta.SignState("vendor-1", now, time.Minute)

	// State absent / falsifié.
	if r := redirectReason(t, callback(h, url.Values{"code": {"c"}}, nonce)); r != "state" {
		t.Fatalf("sans state: %s", r)
	}
	if r := redirectReason(t, callback(h, url.Values{"code": {"c"}, "state": {state + "x"}}, nonce)); r != "state" {
		t.Fatalf("state falsifié: %s", r)
	}
	// State valide mais navigateur différent (pas de cookie / autre nonce) :
	// lien de connexion fabriqué par un tiers.
	if r := redirectReason(t, callback(h, url.Values{"code": {"c"}, "state": {state}}, "")); r != "state" {
		t.Fatalf("sans cookie: %s", r)
	}
	if r := redirectReason(t, callback(h, url.Values{"code": {"c"}, "state": {state}}, "autre-nonce")); r != "state" {
		t.Fatalf("mauvais cookie: %s", r)
	}
	// State expiré.
	old, oldNonce, _ := h.meta.SignState("vendor-1", now.Add(-time.Hour), time.Minute)
	if r := redirectReason(t, callback(h, url.Values{"code": {"c"}, "state": {old}}, oldNonce)); r != "expired" {
		t.Fatalf("state expiré: %s", r)
	}
	// Le cookie nonce est toujours effacé.
	rec := callback(h, url.Values{}, "")
	if c := rec.Result().Cookies(); len(c) != 1 || c[0].Name != metaOAuthCookie || c[0].MaxAge >= 0 {
		t.Fatalf("cookie non effacé: %+v", c)
	}
}

func TestOAuthCallbackUserDenied(t *testing.T) {
	h := testAdHandler(t, "http://127.0.0.1:1")
	state, nonce, _ := h.meta.SignState("vendor-1", time.Now(), time.Minute)
	q := url.Values{"state": {state}, "error": {"access_denied"}, "error_reason": {"user_denied"}}
	if r := redirectReason(t, callback(h, q, nonce)); r != "denied" {
		t.Fatalf("refus: %s", r)
	}
}

func TestOAuthCallbackPermissionMissing(t *testing.T) {
	var calls []string
	graph := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v23.0/")
		calls = append(calls, path)
		switch path {
		case "oauth/access_token":
			if r.URL.Query().Get("grant_type") == "fb_exchange_token" {
				io.WriteString(w, `{"access_token":"long","expires_in":5184000}`)
			} else {
				io.WriteString(w, `{"access_token":"short","expires_in":3600}`)
			}
		case "me/permissions":
			io.WriteString(w, `{"data":[{"permission":"ads_management","status":"declined"},{"permission":"pages_show_list","status":"granted"}]}`)
		default:
			t.Errorf("appel inattendu: %s", path)
			w.WriteHeader(500)
		}
	}))
	defer graph.Close()
	h := testAdHandler(t, graph.URL)
	state, nonce, _ := h.meta.SignState("vendor-1", time.Now(), time.Minute)
	if r := redirectReason(t, callback(h, url.Values{"state": {state}, "code": {"abc"}}, nonce)); r != "permissions" {
		t.Fatalf("permission refusée: %s", r)
	}
	if strings.Join(calls, ",") != "oauth/access_token,oauth/access_token,me/permissions" {
		t.Fatalf("appels: %v", calls)
	}

	// Échange du code refusé par Meta -> reason=exchange.
	reason, err := testAdHandler(t, "http://127.0.0.1:1").completeOAuth(context.Background(), "vendor-1", "abc")
	if err == nil || reason != "exchange" {
		t.Fatalf("échange en échec: %s %v", reason, err)
	}
}

func TestDialogURLCarriesSignedState(t *testing.T) {
	h := testAdHandler(t, "http://unused")
	state, nonce, _ := h.meta.SignState("vendor-1", time.Now(), metaOAuthStateTTL)
	u, _ := url.Parse(h.meta.DialogURL(state))
	st, err := h.meta.VerifyState(u.Query().Get("state"), time.Now())
	if err != nil || st.VendorID != "vendor-1" || st.Nonce != nonce {
		t.Fatalf("state de l'URL: %+v %v", st, err)
	}
}
