package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// yes_webinar.go — webinaires YES Business (jusqu'à 1000 spectateurs), guide
// d'intégration reçu le 2026-09-30. DIARRA ne fait que créer/piloter le
// webinaire : diffusion vidéo (LiveKit), tchat, Q/R, sondages, inscription
// et replay restent entièrement gérés par l'interface YES — jamais intégrés
// dans le frontend DIARRA. Mêmes base URL, clé et signature HMAC que le
// reste de l'intégration (YesBusinessClient.do), aucune nouvelle clé.

// WebinarRegistrationField — champ supplémentaire du formulaire
// d'inscription public YES (ex {key:"company", label:"Entreprise"}).
type WebinarRegistrationField struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
}

type CreateWebinarRequest struct {
	Title                             string                     `json:"title"`
	Description                       string                     `json:"description"`
	ScheduledStartAt                  string                     `json:"scheduled_start_at"` // ISO 8601 / RFC3339
	EstimatedDurationMinutes          int                        `json:"estimated_duration_minutes"`
	AccessType                        string                     `json:"access_type"` // "public" | "private"
	ParticipantMicEnabledByDefault    bool                       `json:"participant_mic_enabled_by_default"`
	ParticipantCameraEnabledByDefault bool                       `json:"participant_camera_enabled_by_default"`
	ChatEnabled                       bool                       `json:"chat_enabled"`
	QAEnabled                         bool                       `json:"qa_enabled"`
	CustomRegistrationFields          []WebinarRegistrationField `json:"custom_registration_fields"`
}

// Webinar — objet webinaire renvoyé par YES. Seuls les champs documentés
// sont typés ; Raw conserve la réponse complète (liens d'inscription,
// statut...) pour la renvoyer telle quelle à l'admin sans en perdre.
type Webinar struct {
	ID                       string          `json:"id"`
	Title                    string          `json:"title"`
	Description              string          `json:"description"`
	ScheduledStartAt         string          `json:"scheduled_start_at"`
	EstimatedDurationMinutes int             `json:"estimated_duration_minutes"`
	AccessType               string          `json:"access_type"`
	Status                   string          `json:"status,omitempty"`
	RecordingURL             string          `json:"recording_url,omitempty"`
	Raw                      json.RawMessage `json:"-"`
}

func (w *Webinar) UnmarshalJSON(data []byte) error {
	type alias Webinar
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*w = Webinar(a)
	w.Raw = append(json.RawMessage(nil), data...)
	return nil
}

// MarshalJSON renvoie la réponse YES complète quand on l'a (tous les champs,
// y compris ceux non typés ici), sinon les champs typés.
func (w Webinar) MarshalJSON() ([]byte, error) {
	if len(w.Raw) > 0 {
		return w.Raw, nil
	}
	type alias Webinar
	return json.Marshal(alias(w))
}

type StartWebinarResponse struct {
	Webinar Webinar `json:"webinar"`
	// HostToken/WebsocketToken : réservés à l'interface d'hébergement YES,
	// jamais utilisés par le frontend DIARRA.
	HostToken      string `json:"host_token"`
	WebsocketToken string `json:"websocket_token"`
}

type WebinarStats struct {
	RegisteredCount     int     `json:"registered_count"`
	AttendedCount       int     `json:"attended_count"`
	AverageWatchMinutes float64 `json:"average_watch_minutes"`
}

// CreateWebinar — POST /api/v1/yes/webinar/create (201). L'id renvoyé sert
// aux appels suivants.
func (c *YesBusinessClient) CreateWebinar(ctx context.Context, req CreateWebinarRequest) (*Webinar, error) {
	if req.CustomRegistrationFields == nil {
		req.CustomRegistrationFields = []WebinarRegistrationField{}
	}
	var out Webinar
	if err := c.do(ctx, http.MethodPost, "/api/v1/yes/webinar/create", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListWebinars — GET /api/v1/yes/webinar/list.
func (c *YesBusinessClient) ListWebinars(ctx context.Context) ([]Webinar, error) {
	out := []Webinar{}
	if err := c.do(ctx, http.MethodGet, "/api/v1/yes/webinar/list", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// StartWebinar — POST /api/v1/yes/webinar/{id}/start, corps optionnel
// {enable_recording}.
func (c *YesBusinessClient) StartWebinar(ctx context.Context, id string, enableRecording bool) (*StartWebinarResponse, error) {
	var out StartWebinarResponse
	body := map[string]bool{"enable_recording": enableRecording}
	if err := c.do(ctx, http.MethodPost, webinarPath(id, "start"), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// EndWebinar — POST /api/v1/yes/webinar/{id}/end. recording_url est rempli
// si l'enregistrement était actif.
func (c *YesBusinessClient) EndWebinar(ctx context.Context, id string) (*Webinar, error) {
	var out Webinar
	if err := c.do(ctx, http.MethodPost, webinarPath(id, "end"), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetWebinarStats — GET /api/v1/yes/webinar/{id}/stats.
func (c *YesBusinessClient) GetWebinarStats(ctx context.Context, id string) (*WebinarStats, error) {
	var out WebinarStats
	if err := c.do(ctx, http.MethodGet, webinarPath(id, "stats"), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// webinarPath — l'id vient de YES (ou d'un admin) : échappé pour qu'il ne
// puisse pas modifier le chemin signé (ex "../session/x").
func webinarPath(id, action string) string {
	return fmt.Sprintf("/api/v1/yes/webinar/%s/%s", url.PathEscape(id), action)
}
