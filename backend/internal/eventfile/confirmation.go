// Package eventfile génère le fichier livré automatiquement à l'achat d'une
// offre payante d'événement (billet). Ce n'est pas un vrai billet
// d'activation (voir Confirmation.Note) : les prestations qu'un fichier ne
// peut pas exécuter lui-même (carte connectée, site web...) restent un
// suivi humain déclaré honnêtement dans le document, jamais présentées
// comme déjà actives. Utilisé par cmd/seed_summit_tickets et
// EventHandler.Create (toute offre payante d'événement vendeur).
package eventfile

import "fmt"

// Confirmation décrit le contenu du fichier de confirmation généré pour une
// offre payante d'événement.
type Confirmation struct {
	Title      string
	PriceCFA   int
	Inclusions []string
	// Note : bandeau affiché sous les inclusions — typiquement pour préciser
	// qu'une équipe recontacte l'acheteur pour la mise en place de ce qu'un
	// fichier ne peut pas activer lui-même. Optionnel.
	Note string
}

// BuildHTML rend le fichier HTML livré comme FileKey du Product associé à
// l'offre. Format volontairement simple (pas de PDF, voir la décision prise
// pour les billets DIARRA Summit — pas de nouvelle dépendance juste pour ça).
func BuildHTML(c Confirmation) string {
	var items string
	for _, inc := range c.Inclusions {
		items += fmt.Sprintf("<li>%s</li>\n", inc)
	}
	note := ""
	if c.Note != "" {
		note = fmt.Sprintf(`<div class="note">%s</div>`, c.Note)
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="fr">
<head>
<meta charset="utf-8">
<title>%s — Confirmation</title>
<style>
body { font-family: Arial, Helvetica, sans-serif; background: #f2f7f4; margin: 0; padding: 32px 16px; color: #0a3225; }
.card { max-width: 600px; margin: 0 auto; background: #fff; border-radius: 16px; padding: 32px; }
h1 { color: #052018; font-size: 22px; }
.price { color: #0f7a50; font-weight: 700; }
ul { padding-left: 20px; line-height: 1.7; }
.note { margin-top: 24px; padding: 16px; background: #f2f7f4; border-radius: 8px; font-size: 14px; }
</style>
</head>
<body>
<div class="card">
<h1>%s</h1>
<p>Merci pour votre inscription — <span class="price">%d FCFA</span></p>
<p><strong>Ce qui est inclus :</strong></p>
<ul>
%s</ul>
%s
</div>
</body>
</html>`, c.Title, c.Title, c.PriceCFA, items, note)
}
