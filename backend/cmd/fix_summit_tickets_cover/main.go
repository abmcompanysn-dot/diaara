// Commande ponctuelle : les 3 billets DIARRA Summit créés par
// cmd/seed_summit_tickets avaient été rejetés en modération faute d'image de
// couverture (moderation_note "image de couverture inexistante"). Ce script
// upload une couverture générée pour chacun, met à jour la description (le
// Summit est passé 100% en ligne, retrait des mentions UCAD/Dakar) et repasse
// les 3 produits en "approved" — pas de code qui tourne en continu, à
// exécuter une fois.
//
// Usage : DATABASE_URL=... S3_*... go run ./cmd/fix_summit_tickets_cover
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ticketFix struct {
	id          string
	title       string
	coverFile   string
	description string
}

var fixes = []ticketFix{
	{
		id:        "ce3919cc-f2bc-423d-b552-3aa6195eae93",
		title:     "DIARRA Summit — Tier Essentiel",
		coverFile: "essentiel.png",
		description: "Accès de base au DIARRA Summit (26 octobre 2026, 100% en ligne) : identité numérique / carte connectée MAHU, " +
			"référencement au répertoire des partenaires ABMCY & DIARRA, support technique initial.",
	},
	{
		id:        "f3c4e983-2194-443e-ad07-fd4993d611ba",
		title:     "DIARRA Summit — Tier Business 2026",
		coverFile: "business.png",
		description: "Tout le Tier Essentiel, plus le Pack Business 2026 : carte intelligente et augmentée MAHU, kit de présentation " +
			"professionnel prêt à l'emploi, visibilité renforcée au sein du réseau d'affaires ABMCY.",
	},
	{
		id:        "8deb934d-7bf8-45d5-9a33-16f3f8be9790",
		title:     "DIARRA Summit — Tier Premium / Site Vitrine",
		coverFile: "premium.png",
		description: "Tout le Tier Business 2026, plus un site web vitrine personnel clé en main (sous-domaine utilisateur.abmcy.com " +
			"ou utilisateur.diarra.app), carte MAHU Premium interconnectée, accès à l'espace de coworking ABMCY, badge de certification " +
			"et accompagnement prioritaire ABMCY.",
	},
}

func main() {
	ctx := context.Background()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	coverDir := os.Getenv("COVER_DIR")
	if coverDir == "" {
		log.Fatal("COVER_DIR is required (dossier contenant essentiel.png / business.png / premium.png)")
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	s3, err := storage.NewS3Storage(storage.S3Config{
		Endpoint:        os.Getenv("S3_ENDPOINT"),
		PublicEndpoint:  os.Getenv("S3_PUBLIC_ENDPOINT"),
		AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
		Bucket:          os.Getenv("S3_BUCKET"),
		Region:          os.Getenv("S3_REGION"),
	})
	if err != nil {
		log.Fatalf("stockage objet requis: %v", err)
	}

	productRepo := repository.NewProductRepo(pool)

	for _, f := range fixes {
		data, err := os.ReadFile(coverDir + "/" + f.coverFile)
		if err != nil {
			log.Fatalf("lecture %s: %v", f.coverFile, err)
		}
		key := storage.NewFileKey("summit-covers", f.coverFile)
		if err := s3.Upload(ctx, key, data); err != nil {
			log.Fatalf("upload %s: %v", f.coverFile, err)
		}

		desc := f.description
		_, err = productRepo.Update(ctx, f.id, model.UpdateProductInput{
			Description:   &desc,
			CoverImageKey: &key,
		})
		if err != nil {
			log.Fatalf("update produit %q: %v", f.title, err)
		}

		if err := productRepo.UpdateModerationStatus(ctx, f.id, "approved", nil); err != nil {
			log.Fatalf("approbation %q: %v", f.title, err)
		}

		fmt.Printf("OK : %s (id=%s) — couverture + description mises à jour, approuvé\n", f.title, f.id)
	}

	log.Println("Terminé. Les 3 billets sont approuvés et visibles sur /summit et le catalogue.")
}
