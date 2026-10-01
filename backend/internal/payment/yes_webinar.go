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
	// HostToken/WebsocketToken : jetons LiveKit bruts, réservés à une
	// intégration vidéo avancée — jamais transmis au frontend DIARRA (voir
	// WebinarHandler.Start), qui utilise HostJoinLink.
	HostToken      string `json:"host_token"`
	WebsocketToken string `json:"websocket_token"`
	// HostJoinLink — lien « Rejoindre en tant qu'hôte », valable 10 minutes :
	// à afficher tel quel, jamais stocké (doc YES Business §6.1/6.3).
	HostJoinLink string `json:"host_join_link"`
	// RegistrationLink — lien PUBLIC d'inscription des participants.
	RegistrationLink string `json:"registration_link"`
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

// WebinarRegistration — un inscrit (doc YES Business §6.6). Attended passe
// à true seulement une fois la personne réellement entrée dans le live.
type WebinarRegistration struct {
	ID           string `json:"id"`
	WebinarID    string `json:"webinar_id"`
	Email        string `json:"email"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Phone        string `json:"phone"`
	Company      string `json:"company"`
	Attended     bool   `json:"attended"`
	RegisteredAt string `json:"registered_at"`
}

// ListWebinarRegistrations — GET /api/v1/yes/webinar/{id}/registrations,
// plus ancien inscrit en premier.
func (c *YesBusinessClient) ListWebinarRegistrations(ctx context.Context, id string) ([]WebinarRegistration, error) {
	out := []WebinarRegistration{}
	if err := c.do(ctx, http.MethodGet, webinarPath(id, "registrations"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ResendWebinarRegistration — POST
// /api/v1/yes/webinar/{id}/registrations/{registrationId}/resend, corps vide.
// YES choisit l'email selon le statut du webinaire : "ça commence bientôt"
// (scheduled) ou "c'est en direct maintenant" (live), avec le lien d'accès.
func (c *YesBusinessClient) ResendWebinarRegistration(ctx context.Context, id, registrationID string) error {
	path := webinarPath(id, "registrations/"+url.PathEscape(registrationID)+"/resend")
	return c.do(ctx, http.MethodPost, path, nil, nil)
}

// webinarPath — l'id vient de YES (ou d'un admin) : échappé pour qu'il ne
// puisse pas modifier le chemin signé (ex "../session/x").
func webinarPath(id, action string) string {
	return fmt.Sprintf("/api/v1/yes/webinar/%s/%s", url.PathEscape(id), action)
}
