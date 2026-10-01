package payment

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

type metaCall struct {
	Method string
	Path   string
	Form   url.Values
}

// fakeMeta — faux serveur Graph : enregistre les appels et répond selon
// failPath (erreur 400 sur ce chemin) ; tous les objets créés reçoivent un id
// dérivé du chemin. Les appels "utilisateur" (me, me/accounts...) et OAuth
// ont des réponses fixes.
func fakeMeta(t *testing.T, failPath string) (*httptest.Server, *[]metaCall) {
	var mu sync.Mutex
	calls := []metaCall{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(raw))
		if r.Method != http.MethodPost {
			form = r.URL.Query()
		}
		path := strings.TrimPrefix(r.URL.Path, "/v23.0/")
		mu.Lock()
		calls = append(calls, metaCall{r.Method, path, form})
		mu.Unlock()

		if path == "oauth/access_token" {
			if form.Get("client_secret") != "sec" || form.Get("client_id") != "app1" {
				w.WriteHeader(400)
				io.WriteString(w, `{"error":{"message":"bad client","code":101}}`)
				return
			}
			switch {
			case form.Get("code") == "good-code" && form.Get("redirect_uri") == "https://api.test/api/meta/oauth/callback":
				io.WriteString(w, `{"access_token":"short-tok","token_type":"bearer","expires_in":3600}`)
			case form.Get("grant_type") == "fb_exchange_token" && form.Get("fb_exchange_token") == "short-tok":
				io.WriteString(w, `{"access_token":"long-tok","token_type":"bearer","expires_in":5184000}`)
			default:
				w.WriteHeader(400)
				io.WriteString(w, `{"error":{"message":"Invalid verification code format.","code":100}}`)
			}
			return
		}

		if form.Get("access_token") != "tok" {
			t.Errorf("access_token manquant sur %s", r.URL.Path)
		}
		if form.Get("appsecret_proof") == "" {
			t.Errorf("appsecret_proof manquant sur %s", r.URL.Path)
		}
		if path == failPath {
			w.WriteHeader(400)
			io.WriteString(w, `{"error":{"message":"Invalid parameter","code":100,"error_subcode":1885183,"error_user_msg":"Le visuel ne respecte pas les règles."}}`)
			return
		}
		switch {
		case path == "me":
			io.WriteString(w, `{"id":"fb42","name":"Awa Diop"}`)
		case path == "me/accounts":
			io.WriteString(w, `{"data":[{"id":"page1","name":"Boutique Awa"}]}`)
		case path == "me/adaccounts":
			io.WriteString(w, `{"data":[{"id":"act_1","name":"Pub Awa","currency":"XOF","account_status":1},{"id":"act_2","name":"Ancien","currency":"EUR","account_status":2}]}`)
		case path == "me/permissions" && r.Method == http.MethodGet:
			io.WriteString(w, `{"data":[{"permission":"ads_management","status":"granted"},{"permission":"ads_read","status":"declined"}]}`)
		case strings.HasSuffix(path, "/insights"):
			io.WriteString(w, `{"data":[{"impressions":"1200","reach":"900","inline_link_clicks":"45","spend":"3.05"}]}`)
		case r.Method == http.MethodGet:
			io.WriteString(w, `{"effective_status":"DISAPPROVED","ad_review_feedback":{"global":{"Contenu trompeur":"L'image promet un résultat irréaliste."}}}`)
		default:
			io.WriteString(w, `{"id":"`+strings.ReplaceAll(path, "/", "_")+`_id","success":true}`)
		}
	}))
	return srv, &calls
}

func newTestApp(url string) *MetaApp {
	return NewMetaApp(MetaAppConfig{
		AppID: "app1", AppSecret: "sec", RedirectURL: "https://api.test/api/meta/oauth/callback",
		BaseURL: url, DialogURL: "https://fb.test",
	})
}

func newTestMeta(url string) *MetaAdsClient {
	return newTestApp(url).Client(MetaAccount{AccessToken: "tok", AdAccountID: "123", PageID: "page1", Currency: "EUR"})
}

func sampleReq() SponsoredAdRequest {
	start := time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)
	return SponsoredAdRequest{
		Name: "DIARRA test", LinkURL: "https://diarra.app/product?id=p1", Headline: "Ebook Wax",
		Message: "Découvrez", ImageURL: "https://api.diarra.app/api/products/p1/cover",
		Countries: []string{"SN", "CI"}, BudgetXOF: 6560, StartTime: start, EndTime: start.Add(7 * 24 * time.Hour),
	}
}

func TestCreateSponsoredAdOrderAndActivationLast(t *testing.T) {
	srv, calls := fakeMeta(t, "")
	defer srv.Close()
	res, err := newTestMeta(srv.URL).CreateSponsoredAd(context.Background(), sampleReq())
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"act_123/campaigns", "act_123/adsets", "act_123/adcreatives", "act_123/ads", "act_123_campaigns_id"}
	if len(*calls) != len(want) {
		t.Fatalf("appels: %+v", *calls)
	}
	for i, w := range want {
		if (*calls)[i].Path != w {
			t.Fatalf("appel %d = %s, attendu %s", i, (*calls)[i].Path, w)
		}
	}
	c := *calls
	if c[0].Form.Get("status") != "PAUSED" || c[4].Form.Get("status") != "ACTIVE" {
		t.Fatal("la campagne doit être créée en pause et activée en dernier")
	}
	// 6560 FCFA = 10,00 EUR -> 1000 centimes.
	if c[1].Form.Get("lifetime_budget") != "1000" {
		t.Fatalf("budget = %s", c[1].Form.Get("lifetime_budget"))
	}
	var targeting map[string]interface{}
	json.Unmarshal([]byte(c[1].Form.Get("targeting")), &targeting)
	if geo := targeting["geo_locations"].(map[string]interface{}); len(geo["countries"].([]interface{})) != 2 {
		t.Fatalf("ciblage pays: %v", targeting)
	}
	var spec map[string]interface{}
	json.Unmarshal([]byte(c[2].Form.Get("object_story_spec")), &spec)
	if spec["page_id"] != "page1" || spec["link_data"].(map[string]interface{})["picture"] == nil {
		t.Fatalf("visuel: %v", spec)
	}
	if _, ok := spec["instagram_user_id"]; ok {
		t.Fatal("pas d'instagram_user_id par défaut")
	}
	if res.AdID != "act_123_ads_id" || res.CampaignID != "act_123_campaigns_id" {
		t.Fatalf("ids: %+v", res)
	}
}

func TestCreateSponsoredAdXOFAccountNoConversion(t *testing.T) {
	srv, calls := fakeMeta(t, "")
	defer srv.Close()
	c := newTestApp(srv.URL).Client(MetaAccount{AccessToken: "tok", AdAccountID: "act_9", PageID: "p", Currency: "xof"})
	if _, err := c.CreateSponsoredAd(context.Background(), sampleReq()); err != nil {
		t.Fatal(err)
	}
	if got := (*calls)[1].Form.Get("lifetime_budget"); got != "6560" || (*calls)[0].Path != "act_9/campaigns" {
		t.Fatalf("budget XOF = %s, chemin %s", got, (*calls)[0].Path)
	}
}

func TestCreateSponsoredAdCleansUpOnFailure(t *testing.T) {
	srv, calls := fakeMeta(t, "act_123/adcreatives")
	defer srv.Close()
	_, err := newTestMeta(srv.URL).CreateSponsoredAd(context.Background(), sampleReq())
	var metaErr *MetaAPIError
	if !errors.As(err, &metaErr) || metaErr.UserMessage != "Le visuel ne respecte pas les règles." {
		t.Fatalf("erreur: %v", err)
	}
	last := (*calls)[len(*calls)-1]
	if last.Path != "act_123_campaigns_id" || last.Form.Get("status") != "DELETED" {
		t.Fatalf("la campagne en pause doit être supprimée après un échec, dernier appel: %+v", last)
	}
	for _, c := range *calls {
		if c.Form.Get("status") == "ACTIVE" && c.Path == "act_123_campaigns_id" {
			t.Fatal("campagne activée malgré l'échec")
		}
	}
}

func TestCreateSponsoredAdUnsupportedCurrencyMakesNoCall(t *testing.T) {
	srv, calls := fakeMeta(t, "")
	defer srv.Close()
	c := newTestApp(srv.URL).Client(MetaAccount{AccessToken: "tok", AdAccountID: "1", PageID: "p", Currency: "GBP"})
	if _, err := c.CreateSponsoredAd(context.Background(), sampleReq()); !errors.Is(err, ErrMetaCurrencyUnsupported) {
		t.Fatalf("erreur: %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("aucun appel Meta attendu: %+v", *calls)
	}
}

func TestMetaStatusAndInsights(t *testing.T) {
	srv, _ := fakeMeta(t, "")
	defer srv.Close()
	m := newTestMeta(srv.URL)
	st, err := m.GetAdStatus(context.Background(), "ad1")
	if err != nil || st.EffectiveStatus != "DISAPPROVED" || !strings.Contains(st.ReviewFeedback, "Contenu trompeur") {
		t.Fatalf("statut: %+v %v", st, err)
	}
	ins, err := m.GetCampaignInsights(context.Background(), "camp1")
	// 3,05 EUR = 2000,67 FCFA -> 2001.
	if err != nil || ins.Clicks != 45 || ins.Impressions != 1200 || ins.SpendXOF != 2001 {
		t.Fatalf("insights: %+v %v", ins, err)
	}
}

func TestMetaUserAssets(t *testing.T) {
	srv, _ := fakeMeta(t, "")
	defer srv.Close()
	c := newTestApp(srv.URL).Client(MetaAccount{AccessToken: "tok"})
	me, err := c.Me(context.Background())
	if err != nil || me.ID != "fb42" || me.Name != "Awa Diop" {
		t.Fatalf("me: %+v %v", me, err)
	}
	pages, err := c.ListPages(context.Background())
	if err != nil || len(pages) != 1 || pages[0].Name != "Boutique Awa" {
		t.Fatalf("pages: %+v %v", pages, err)
	}
	accounts, err := c.ListAdAccounts(context.Background())
	if err != nil || len(accounts) != 2 || !accounts[0].Usable() || accounts[1].Usable() || accounts[0].Currency != "XOF" {
		t.Fatalf("comptes: %+v %v", accounts, err)
	}
	perms, err := c.GrantedPermissions(context.Background())
	if err != nil || !perms["ads_management"] || perms["ads_read"] {
		t.Fatalf("permissions: %v %v", perms, err)
	}
}

func TestMetaTokenErrorAndNoURLLeak(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		io.WriteString(w, `{"error":{"message":"Error validating access token: Session has expired","type":"OAuthException","code":190,"error_subcode":463}}`)
	}))
	defer srv.Close()
	c := newTestApp(srv.URL).Client(MetaAccount{AccessToken: "SECRET-TOKEN"})
	_, err := c.Me(context.Background())
	if !IsMetaTokenError(err) {
		t.Fatalf("code 190 non reconnu: %v", err)
	}
	if strings.Contains(err.Error(), "SECRET-TOKEN") {
		t.Fatal("le jeton apparaît dans l'erreur")
	}
	if IsMetaTokenError(&MetaAPIError{Code: 100}) || IsMetaTokenError(errors.New("x")) {
		t.Fatal("faux positif")
	}

	// Serveur injoignable : l'erreur ne doit pas citer l'URL (qui porte le jeton).
	srv.Close()
	_, err = c.Me(context.Background())
	if err == nil || strings.Contains(err.Error(), "SECRET-TOKEN") || strings.Contains(err.Error(), "access_token") {
		t.Fatalf("fuite dans l'erreur réseau: %v", err)
	}
}

func TestMetaCurrencyConversion(t *testing.T) {
	for _, tc := range []struct {
		cur  string
		xof  int
		want int64
	}{{"EUR", 655957, 100000}, {"XOF", 5000, 5000}, {"XAF", 5000, 5000}, {"EUR", 656, 100}, {"usd", 56876, 10000}} {
		got, err := XOFToMetaMinorUnits(tc.xof, tc.cur)
		if err != nil || got != tc.want {
			t.Fatalf("%s %d -> %d (%v), attendu %d", tc.cur, tc.xof, got, err, tc.want)
		}
	}
	if _, err := XOFToMetaMinorUnits(1000, "GBP"); !errors.Is(err, ErrMetaCurrencyUnsupported) {
		t.Fatal("devise non supportée acceptée")
	}
	if !MetaCurrencySupported("eur") || MetaCurrencySupported("NGN") {
		t.Fatal("MetaCurrencySupported")
	}
	if MetaAmountToXOF("10", "XOF") != 10 || MetaAmountToXOF("1", "EUR") != 656 || MetaAmountToXOF("x", "EUR") != 0 {
		t.Fatal("MetaAmountToXOF")
	}
}
