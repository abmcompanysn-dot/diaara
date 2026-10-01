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

// fakeMeta enregistre les appels Graph et répond selon failPath (erreur 400
// sur ce chemin) ; tous les objets créés reçoivent un id dérivé du chemin.
func fakeMeta(t *testing.T, failPath string) (*httptest.Server, *[]metaCall) {
	var mu sync.Mutex
	calls := []metaCall{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		form, _ := url.ParseQuery(string(raw))
		if r.Method == http.MethodGet {
			form = r.URL.Query()
		}
		if form.Get("access_token") != "tok" {
			t.Errorf("access_token manquant sur %s", r.URL.Path)
		}
		if form.Get("appsecret_proof") == "" {
			t.Errorf("appsecret_proof manquant sur %s", r.URL.Path)
		}
		path := strings.TrimPrefix(r.URL.Path, "/v23.0/")
		mu.Lock()
		calls = append(calls, metaCall{r.Method, path, form})
		mu.Unlock()
		if path == failPath {
			w.WriteHeader(400)
			io.WriteString(w, `{"error":{"message":"Invalid parameter","code":100,"error_subcode":1885183,"error_user_msg":"Le visuel ne respecte pas les règles."}}`)
			return
		}
		switch {
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

func newTestMeta(url string) *MetaAdsClient {
	return NewMetaAdsClient(MetaAdsConfig{
		AccessToken: "tok", AppSecret: "sec", AdAccountID: "123", PageID: "page1",
		Currency: "EUR", BaseURL: url,
	})
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
	if res.AdID != "act_123_ads_id" || res.CampaignID != "act_123_campaigns_id" {
		t.Fatalf("ids: %+v", res)
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

func TestMetaCurrencyConversion(t *testing.T) {
	for _, tc := range []struct {
		cur  string
		xof  int
		want int64
	}{{"EUR", 655957, 100000}, {"XOF", 5000, 5000}, {"EUR", 656, 100}} {
		got, err := NewMetaAdsClient(MetaAdsConfig{Currency: tc.cur}).ToAccountMinorUnits(tc.xof)
		if err != nil || got != tc.want {
			t.Fatalf("%s %d -> %d (%v), attendu %d", tc.cur, tc.xof, got, err, tc.want)
		}
	}
	if _, err := NewMetaAdsClient(MetaAdsConfig{Currency: "GBP"}).ToAccountMinorUnits(1000); err == nil {
		t.Fatal("devise non supportée acceptée")
	}
}
