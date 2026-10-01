package payment

import (
	"context"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestMetaStateValid(t *testing.T) {
	app := newTestApp("http://unused")
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	state, nonce, err := app.SignState("vendor-uuid-1", now, 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	got, err := app.VerifyState(state, now.Add(9*time.Minute))
	if err != nil || got.VendorID != "vendor-uuid-1" || got.Nonce != nonce || nonce == "" {
		t.Fatalf("state: %+v %v", got, err)
	}
	// Deux states pour le même vendeur sont différents (nonce).
	state2, _, _ := app.SignState("vendor-uuid-1", now, 10*time.Minute)
	if state == state2 {
		t.Fatal("state réutilisé")
	}
}

func TestMetaStateExpired(t *testing.T) {
	app := newTestApp("http://unused")
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	state, _, _ := app.SignState("v1", now, 10*time.Minute)
	if _, err := app.VerifyState(state, now.Add(10*time.Minute)); !errors.Is(err, ErrMetaStateExpired) {
		t.Fatalf("state expiré accepté: %v", err)
	}
}

func TestMetaStateTampered(t *testing.T) {
	app := newTestApp("http://unused")
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	state, _, _ := app.SignState("victime", now, 10*time.Minute)
	parts := strings.Split(state, ".")
	payload, _ := base64.RawURLEncoding.DecodeString(parts[0])
	forged := strings.Replace(string(payload), "victime", "attaquant", 1)
	cases := []string{
		base64.RawURLEncoding.EncodeToString([]byte(forged)) + "." + parts[1], // payload modifié
		parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte("fausse-signature")),
		parts[0],
		"",
		"a.b.c",
	}
	for _, s := range cases {
		if _, err := app.VerifyState(s, now); !errors.Is(err, ErrMetaStateInvalid) {
			t.Fatalf("state falsifié accepté (%q): %v", s, err)
		}
	}
	// Signé avec un autre secret d'app : refusé.
	other := NewMetaApp(MetaAppConfig{AppSecret: "autre"})
	s, _, _ := other.SignState("victime", now, time.Minute)
	if _, err := app.VerifyState(s, now); !errors.Is(err, ErrMetaStateInvalid) {
		t.Fatal("state d'une autre clé accepté")
	}
	if _, _, err := app.SignState("a|b", now, time.Minute); err == nil {
		t.Fatal("id vendeur avec séparateur accepté")
	}
}

func TestMetaDialogURL(t *testing.T) {
	app := newTestApp("http://unused")
	u, err := url.Parse(app.DialogURL("st"))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Host != "fb.test" || u.Path != "/v23.0/dialog/oauth" || q.Get("client_id") != "app1" || q.Get("state") != "st" ||
		q.Get("redirect_uri") != "https://api.test/api/meta/oauth/callback" || q.Get("response_type") != "code" {
		t.Fatalf("url: %s", u)
	}
	for _, p := range []string{"ads_management", "ads_read", "pages_show_list", "pages_read_engagement", "business_management"} {
		if !strings.Contains(q.Get("scope"), p) {
			t.Fatalf("permission %s absente: %s", p, q.Get("scope"))
		}
	}
	if strings.Contains(u.String(), "sec") {
		t.Fatal("le secret de l'app ne doit jamais apparaître dans l'URL du navigateur")
	}
}

func TestMetaTokenExchange(t *testing.T) {
	srv, calls := fakeMeta(t, "")
	defer srv.Close()
	app := newTestApp(srv.URL)
	short, err := app.ExchangeCode(context.Background(), "good-code")
	if err != nil || short.AccessToken != "short-tok" || short.ExpiresAt == nil {
		t.Fatalf("échange du code: %+v %v", short, err)
	}
	long, err := app.ExchangeLongLived(context.Background(), short.AccessToken)
	if err != nil || long.AccessToken != "long-tok" || long.ExpiresAt == nil || time.Until(*long.ExpiresAt) < 59*24*time.Hour {
		t.Fatalf("jeton longue durée: %+v %v", long, err)
	}
	if len(*calls) != 2 || (*calls)[1].Form.Get("grant_type") != "fb_exchange_token" {
		t.Fatalf("appels: %+v", *calls)
	}

	_, err = app.ExchangeCode(context.Background(), "mauvais-code")
	var metaErr *MetaAPIError
	if !errors.As(err, &metaErr) || strings.Contains(err.Error(), "client_secret") {
		t.Fatalf("code invalide: %v", err)
	}
}
