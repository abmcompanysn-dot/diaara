// Commande ponctuelle : crée un brouillon de message dans la boucle
// vendor-mail pour chaque vendeur actif (demandé le 2026-10-05, approuvé
// explicitement par l'utilisateur après clarification sur le principe de
// validation — voir JOURNAL-MODIFICATIONS.md). Ne crée que des BROUILLONS
// ('draft') : aucun email ne part ici, tout reste à valider dans
// /admin/vendor-mail ("Valider et envoyer tout", espacé de 3s par envoi).
//
// Usage : DATABASE_URL=... go run ./cmd/seed_vendor_mail_broadcast
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/diarra/backend/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

const subject = "Brunel (DIARRA) aimerait votre avis — 2 minutes suffisent"

func bodyFor(greetingName string) string {
	return fmt.Sprintf(`Bonjour%s,

Je m'appelle Brunel Atekossi, responsable technique et opérations chez DIARRA.

Je prends le temps d'écrire directement aux vendeurs de la plateforme pour une raison simple : nous voulons construire DIARRA avec vous, pas seulement pour vous. Les meilleures améliorations qu'on a pu apporter récemment viennent de remarques comme les vôtres — par exemple, un vendeur du Congo-Brazzaville nous a signalé un bug qui empêchait d'enregistrer correctement son numéro Mobile Money, et c'est corrigé depuis.

J'aimerais savoir, très concrètement :
- Qu'est-ce qui fonctionne bien pour vous sur DIARRA aujourd'hui ?
- Qu'est-ce qui vous bloque, vous ralentit, ou vous agace ?
- Y a-t-il une fonctionnalité qui vous manque et qui changerait vraiment les choses pour votre activité ?

Pas besoin d'un message long ou formel — même deux lignes nous aident à prioriser le bon travail.

Merci pour votre confiance et pour ce que vous construisez sur DIARRA.

Bien cordialement,
Brunel Atekossi
Responsable technique & opérations — DIARRA`, greetingName)
}

func main() {
	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	rows, err := pool.Query(ctx, `
		SELECT DISTINCT u.id, u.email, u.shop_name, u.display_name
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role = 'vendeur' AND u.email IS NOT NULL AND u.email != ''`)
	if err != nil {
		log.Fatalf("query vendors: %v", err)
	}
	type vendor struct {
		id, email             string
		shopName, displayName *string
	}
	var vendors []vendor
	for rows.Next() {
		var v vendor
		if err := rows.Scan(&v.id, &v.email, &v.shopName, &v.displayName); err != nil {
			log.Fatalf("scan: %v", err)
		}
		vendors = append(vendors, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		log.Fatalf("rows: %v", err)
	}
	log.Printf("%d vendeurs trouvés", len(vendors))

	mailRepo := repository.NewVendorMailRepo(pool)

	created, skipped := 0, 0
	for _, v := range vendors {
		name := ""
		if v.shopName != nil && strings.TrimSpace(*v.shopName) != "" {
			name = " " + strings.TrimSpace(*v.shopName)
		} else if v.displayName != nil && strings.TrimSpace(*v.displayName) != "" {
			name = " " + strings.TrimSpace(*v.displayName)
		}

		thread, err := mailRepo.FindOrCreateThread(ctx, v.id, subject)
		if err != nil {
			log.Printf("WARNING: fil introuvable pour %s (%s): %v", v.id, v.email, err)
			skipped++
			continue
		}
		if _, err := mailRepo.CreateDraft(ctx, thread.ID, subject, bodyFor(name), nil); err != nil {
			log.Printf("WARNING: brouillon échoué pour %s (%s): %v", v.id, v.email, err)
			skipped++
			continue
		}
		created++
	}

	log.Printf("Terminé : %d brouillons créés, %d ignorés (erreurs). À valider dans /admin/vendor-mail.", created, skipped)
}
