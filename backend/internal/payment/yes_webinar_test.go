package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// fakeYes vérifie la signature HMAC exactement comme la doc YES Business
// (METHOD\npath\ntimestamp\nbody) puis renvoie status/body.
func fakeYes(t *testing.T, secret string, status int, body string, gotPath *string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		msg := r.Method + "\n" + r.URL.EscapedPath() + "\n" + r.Header.Get("X-Timestamp") + "\n" + string(raw)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(msg))
		if hex.EncodeToString(mac.Sum(nil)) != r.Header.Get("X-Signature") {
			t.Errorf("signature HMAC invalide pour %s %s", r.Method, r.URL.EscapedPath())
		}
		if r.Header.Get("X-API-Key") != "bizkey_test" {
			t.Errorf("X-API-Key manquant")
		}
		if gotPath != nil {
			*gotPath = r.Method + " " + r.URL.EscapedPath()
		}
		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
}

func newTestClient(url string) *YesBusinessClient {
	return NewYesBusinessClient(YesBusinessConfig{BaseURL: url, APIKey: "bizkey_test", Secret: "s3cret"})
}

func TestWebinarCallsAreSigned(t *testing.T) {
	ctx := context.Background()
	var path string

	srv := fakeYes(t, "s3cret", 201, `{"id":"wb_1","title":"Lancement","registration_url":"https://yes/r/wb_1"}`, &path)
	w, err := newTestClient(srv.URL).CreateWebinar(ctx, CreateWebinarRequest{Title: "Lancement", ScheduledStartAt: "2026-10-01T10:00:00Z", AccessType: "public"})
	srv.Close()
	if err != nil || w.ID != "wb_1" || path != "POST /api/v1/yes/webinar/create" {
		t.Fatalf("create: w=%+v err=%v path=%s", w, err, path)
	}
	// Les champs YES non typés (ex registration_url) sont conservés.
	if out, _ := w.MarshalJSON(); string(out) != `{"id":"wb_1","title":"Lancement","registration_url":"https://yes/r/wb_1"}` {
		t.Fatalf("champs YES perdus: %s", out)
	}

	srv = fakeYes(t, "s3cret", 200, `[{"id":"a"},{"id":"b"}]`, &path)
	list, err := newTestClient(srv.URL).ListWebinars(ctx)
	srv.Close()
	if err != nil || len(list) != 2 || path != "GET /api/v1/yes/webinar/list" {
		t.Fatalf("list: %v %v %s", list, err, path)
	}

	srv = fakeYes(t, "s3cret", 200, `{"webinar":{"id":"a","status":"live"},"host_token":"h","websocket_token":"ws","host_join_link":"https://yes/w/a/host?t=x","registration_link":"https://yes/w/a"}`, &path)
	st, err := newTestClient(srv.URL).StartWebinar(ctx, "a", true)
	srv.Close()
	if err != nil || st.HostJoinLink != "https://yes/w/a/host?t=x" || st.RegistrationLink != "https://yes/w/a" || path != "POST /api/v1/yes/webinar/a/start" {
		t.Fatalf("start: %+v %v %s", st, err, path)
	}

	srv = fakeYes(t, "s3cret", 200, `{"id":"a","recording_url":"https://yes/rec"}`, &path)
	end, err := newTestClient(srv.URL).EndWebinar(ctx, "a")
	srv.Close()
	if err != nil || end.RecordingURL != "https://yes/rec" || path != "POST /api/v1/yes/webinar/a/end" {
		t.Fatalf("end: %+v %v %s", end, err, path)
	}

	srv = fakeYes(t, "s3cret", 200, `{"registered_count":120,"attended_count":80,"average_watch_minutes":42.5}`, &path)
	stats, err := newTestClient(srv.URL).GetWebinarStats(ctx, "a")
	srv.Close()
	if err != nil || stats.AttendedCount != 80 || path != "GET /api/v1/yes/webinar/a/stats" {
		t.Fatalf("stats: %+v %v %s", stats, err, path)
	}
}

func TestWebinarRegistrations(t *testing.T) {
	ctx := context.Background()
	var path string

	srv := fakeYes(t, "s3cret", 200, `[{"id":"r1","webinar_id":"a","email":"fatou@example.com","first_name":"Fatou","last_name":"Diop","phone":"","company":"","attended":true,"registered_at":"2026-09-28T10:15:00Z"}]`, &path)
	regs, err := newTestClient(srv.URL).ListWebinarRegistrations(ctx, "a")
	srv.Close()
	if err != nil || len(regs) != 1 || regs[0].FirstName != "Fatou" || !regs[0].Attended || path != "GET /api/v1/yes/webinar/a/registrations" {
		t.Fatalf("registrations: %+v %v %s", regs, err, path)
	}

	srv = fakeYes(t, "s3cret", 200, `{"status":"sent"}`, &path)
	err = newTestClient(srv.URL).ResendWebinarRegistration(ctx, "a", "r1")
	srv.Close()
	if err != nil || path != "POST /api/v1/yes/webinar/a/registrations/r1/resend" {
		t.Fatalf("resend: %v %s", err, path)
	}

	// registrationId inconnu -> 404 typé.
	srv = fakeYes(t, "s3cret", 404, `{"error":"registration not found"}`, nil)
	err = newTestClient(srv.URL).ResendWebinarRegistration(ctx, "a", "nope")
	srv.Close()
	var apiErr *YesAPIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 404 {
		t.Fatalf("resend 404: %v", err)
	}
}

// Un id malveillant ne doit pas pouvoir changer le chemin signé.
func TestWebinarIDIsEscaped(t *testing.T) {
	var path string
	srv := fakeYes(t, "s3cret", 200, `{}`, &path)
	defer srv.Close()
	_, _ = newTestClient(srv.URL).GetWebinarStats(context.Background(), "../session/x")
	if path != "GET /api/v1/yes/webinar/..%2Fsession%2Fx/stats" {
		t.Fatalf("id non échappé: %s", path)
	}
}

func TestWebinarErrors(t *testing.T) {
	for _, code := range []int{401, 422, 502} {
		srv := fakeYes(t, "s3cret", code, `{"error":"x"}`, nil)
		_, err := newTestClient(srv.URL).ListWebinars(context.Background())
		srv.Close()
		var apiErr *YesAPIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != code {
			t.Fatalf("code %d: err=%v", code, err)
		}
		// Compatibilité avec les appelants historiques.
		if !errors.Is(err, ErrPaymentFailed) {
			t.Fatalf("code %d: ne wrap plus ErrPaymentFailed", code)
		}
	}

	// YES injoignable -> 502.
	_, err := newTestClient("http://127.0.0.1:1").ListWebinars(context.Background())
	var apiErr *YesAPIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("injoignable: err=%v", err)
	}
}
