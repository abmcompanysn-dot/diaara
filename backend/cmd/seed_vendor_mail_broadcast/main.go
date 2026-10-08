// Commande ponctuelle : crée un brouillon de message dans la boucle
// vendor-mail pour chaque vendeur actif (campagne Octobre Rose, demandée le
// 2026-10-08). Ne crée que des BROUILLONS ('draft') : aucun email ne part
// ici, tout reste à valider dans /admin/vendor-mail ("Valider et envoyer
// tout", espacé de 3s par envoi). Pas de personnalisation par nom (demande
// explicite) — bodyFor n'a donc pas de paramètre, contrairement aux
// campagnes précédentes utilisant ce même script.
//
// Usage : DATABASE_URL=... go run ./cmd/seed_vendor_mail_broadcast
package main

import (
	"context"
	"log"
	"os"

	"github.com/diarra/backend/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

const subject = "Octobre Rose — un mot de l'équipe DIARRA"
const bannerURL = "https://diarra.app/brand/octobre-rose-banner.png"

const body = `Bonjour,

Octobre, c'est le mois de la sensibilisation au cancer du sein — Octobre Rose. Je voulais prendre un instant pour en parler avec vous, au-delà du business qui nous réunit d'habitude.

Le dépistage précoce change tout : détecté tôt, ce cancer se soigne dans l'immense majorité des cas. Si vous avez une mère, une sœur, une épouse, une amie — ou vous-même — n'attendez pas un symptôme pour consulter. Un simple examen peut sauver une vie. Prenez ce mois pour vous renseigner, en parler autour de vous, ou simplement prendre rendez-vous.

Si vous le souhaitez, vous pouvez aussi publier gratuitement un ebook ou un guide sur DIARRA ce mois-ci (santé, bien-être, ou tout autre sujet qui vous tient à cœur) — un geste simple pour la communauté, sans frais ni commission sur un produit à 0 FCFA.

Chez DIARRA, vous faites partie d'une communauté de plus de 150 vendeurs à travers plusieurs pays d'Afrique. Cette communauté, c'est aussi des personnes, pas seulement des transactions. Prenez soin de vous et de vos proches.

Bien à vous,
Brunel Atekossi Mahuzonsou
Équipe DIARRA`

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
		SELECT DISTINCT u.id, u.email
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role = 'vendeur' AND u.email IS NOT NULL AND u.email != ''`)
	if err != nil {
		log.Fatalf("query vendors: %v", err)
	}
	type vendor struct {
		id, email string
	}
	var vendors []vendor
	for rows.Next() {
		var v vendor
		if err := rows.Scan(&v.id, &v.email); err != nil {
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
		thread, err := mailRepo.FindOrCreateThread(ctx, v.id, subject)
		if err != nil {
			log.Printf("WARNING: fil introuvable pour %s (%s): %v", v.id, v.email, err)
			skipped++
			continue
		}
		if _, err := mailRepo.CreateDraftWithBanner(ctx, thread.ID, subject, body, bannerURL); err != nil {
			log.Printf("WARNING: brouillon échoué pour %s (%s): %v", v.id, v.email, err)
			skipped++
			continue
		}
		created++
	}

	log.Printf("Terminé : %d brouillons créés, %d ignorés (erreurs). À valider dans /admin/vendor-mail.", created, skipped)
}
