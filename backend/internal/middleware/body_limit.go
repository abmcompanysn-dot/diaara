package middleware

import (
	"net/http"
	"strings"
)

// maxJSONBodyBytes — plafond appliqué par défaut à toute requête qui n'est
// pas un upload de fichier (voir isMultipartUpload). 2 Mo est très large pour
// n'importe quel payload JSON légitime de l'API (le plus gros aujourd'hui est
// un broadcast admin en HTML libre, très en dessous de cette taille) — sert
// uniquement à empêcher un corps de requête arbitrairement gros (déni de
// service applicatif : mémoire consommée par json.Decode, temps CPU de
// parsing), pas à contraindre un usage normal.
const maxJSONBodyBytes = 2 << 20 // 2 Mo

// maxUploadBodyBytes — plafond pour les requêtes multipart/form-data
// (upload de fichier). Les handlers concernés (ProductHandler.Upload,
// AutoCreate, AttachFile, UpdateAutomation, UpdateAutomationCover) appellent
// déjà r.ParseMultipartForm(50 << 20) en interne ; http.MaxBytesReader rejette
// PLUS TÔT (dès la lecture du corps), pour la même limite déjà en vigueur.
const maxUploadBodyBytes = 50 << 20 // 50 Mo

// isMultipartUpload détecte un upload de fichier par Content-Type plutôt que
// par chemin de route : les routes d'upload de product_handler.go incluent
// des segments dynamiques ({id}/file, {id}/cover) qu'une liste de chemins
// exacts ne capturerait pas de façon fiable, et toute future route d'upload
// n'aurait pas besoin d'être ajoutée manuellement ici.
func isMultipartUpload(r *http.Request) bool {
	return strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data")
}

// MaxRequestBody plafonne la taille du corps de toute requête entrante —
// absent jusqu'ici (audit sécurité 2026-09-27) : un POST JSON pouvait être
// arbitrairement gros, borné seulement par le ReadTimeout du serveur (temps,
// pas octets). http.MaxBytesReader fait échouer la lecture dès que la limite
// est dépassée (le prochain r.Body.Read renvoie une erreur), avant que
// json.Decode ou ParseMultipartForm n'aient à traiter le surplus.
func MaxRequestBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := int64(maxJSONBodyBytes)
		if isMultipartUpload(r) {
			limit = maxUploadBodyBytes
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		next.ServeHTTP(w, r)
	})
}
