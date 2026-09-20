// Commande ponctuelle : crée les 3 billets DIARRA Summit (26 octobre 2026)
// comme des produits catalogue normaux, sous un compte vendeur dédié
// "DIARRA Summit". Volontairement séparée de cmd/server — ce n'est pas du
// code qui tourne en continu, juste un script à exécuter une fois (et à
// relancer si besoin, il est idempotent : voir findOrCreateVendor et
// ErrProductSlugExists plus bas).
//
// Usage : DATABASE_URL=... S3_*... go run ./cmd/seed_summit_tickets
//
// Les 3 produits sont créés en moderation_status "pending" (comportement
// par défaut de ProductRepo.Create) : ils n'apparaissent PAS publiquement
// tant qu'un admin ne les approuve pas depuis /admin/products — voir
// ProductRepo.ListApproved (WHERE moderation_status = 'approved').
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/diarra/backend/internal/auth"
	"github.com/diarra/backend/internal/eventfile"
	"github.com/diarra/backend/internal/model"
	"github.com/diarra/backend/internal/repository"
	"github.com/diarra/backend/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	summitVendorEmail = "summit@diarra.app"
	summitVendorShop  = "DIARRA Summit"
)

type ticketTier struct {
	title       string
	priceCFA    int
	description string
	// summaryHTML : contenu du fichier livré à l'achat (voir buildTicketHTML).
	// Récapitulatif honnête de ce qui est inclus — pas de promesse
	// d'activation instantanée pour ce qu'un fichier ne peut pas faire
	// (carte MAHU, site vitrine) : l'acheteur est prévenu qu'une équipe le
	// recontacte pour la suite (voir la question posée à l'utilisateur).
	inclusions []string
}

var tiers = []ticketTier{
	{
		title:    "DIARRA Summit — Tier Essentiel",
		priceCFA: 5000,
		description: "Accès de base au DIARRA Summit (26 octobre 2026, à l'UCAD Dakar et en ligne) : identité numérique / carte connectée MAHU, " +
			"référencement au répertoire des partenaires ABMCY & DIARRA, support technique initial.",
		inclusions: []string{
			"Accès à l'événement DIARRA Summit du 26 octobre 2026 (UCAD, Dakar, et en ligne)",
			"Création et activation de votre identité numérique / carte connectée MAHU",
			"Référencement de base au sein du répertoire des partenaires (ABMCY & DIARRA)",
			"Support technique initial",
		},
	},
	{
		title:    "DIARRA Summit — Tier Business 2026",
		priceCFA: 15000,
		description: "Tout le Tier Essentiel, plus le Pack Business 2026 : carte intelligente et augmentée MAHU, kit de présentation " +
			"professionnel prêt à l'emploi, visibilité renforcée au sein du réseau d'affaires ABMCY.",
		inclusions: []string{
			"Tout le contenu du Tier Essentiel (5 000 FCFA)",
			"Carte intelligente et augmentée MAHU",
			"Contenus et kit de présentation professionnels prêts à l'emploi",
			"Visibilité renforcée au sein du réseau d'affaires ABMCY",
		},
	},
	{
		title:    "DIARRA Summit — Tier Premium / Site Vitrine",
		priceCFA: 25000,
		description: "Tout le Tier Business 2026, plus un site web vitrine personnel clé en main (sous-domaine utilisateur.abmcy.com " +
			"ou utilisateur.diarra.app), carte MAHU Premium interconnectée, accès à l'espace de coworking ABMCY, badge de certification " +
			"et accompagnement prioritaire ABMCY.",
		inclusions: []string{
			"Tout le contenu du Tier Business 2026 (15 000 FCFA)",
			"Création, design et déploiement de votre site web personnel",
			"Nom de sous-domaine personnalisé : utilisateur.abmcy.com ou utilisateur.diarra.app",
			"Intégration des boutons de contact et passerelles de paiement DIARRA",
			"Carte MAHU Premium interconnectée en temps réel avec votre nouveau site",
			"Accès à l'espace de coworking / innovation ABMCY pour votre organisation",
			"Badge de certification et accompagnement prioritaire ABMCY",
		},
	},
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

	s3, err := storage.NewS3Storage(storage.S3Config{
		Endpoint:        os.Getenv("S3_ENDPOINT"),
		PublicEndpoint:  os.Getenv("S3_PUBLIC_ENDPOINT"),
		AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
		Bucket:          os.Getenv("S3_BUCKET"),
		Region:          os.Getenv("S3_REGION"),
	})
	if err != nil {
		log.Fatalf("stockage objet requis pour livrer le fichier des billets: %v", err)
	}

	userRepo := repository.NewUserRepo(pool)
	productRepo := repository.NewProductRepo(pool)

	vendorID, err := findOrCreateSummitVendor(ctx, userRepo)
	if err != nil {
		log.Fatalf("compte vendeur DIARRA Summit: %v", err)
	}
	log.Printf("Compte vendeur DIARRA Summit : %s", vendorID)

	for _, tier := range tiers {
		fileKey := storage.NewFileKey(vendorID, fmt.Sprintf("billet-%d-fcfa.html", tier.priceCFA))
		confirmationHTML := eventfile.BuildHTML(eventfile.Confirmation{
			Title:      tier.title,
			PriceCFA:   tier.priceCFA,
			Inclusions: tier.inclusions,
			Note: "Une équipe DIARRA / ABMCY / MAHU vous contactera sous 48h pour la mise en place des éléments listés " +
				"ci-dessus (carte connectée, site vitrine, etc. selon votre palier). Ce document confirme votre achat ; " +
				"il ne constitue pas une activation immédiate de ces services.",
		})
		if err := s3.Upload(ctx, fileKey, []byte(confirmationHTML)); err != nil {
			log.Fatalf("upload fichier billet %q: %v", tier.title, err)
		}

		desc := tier.description
		product, err := productRepo.Create(ctx, model.CreateProductInput{
			Title:       tier.title,
			Description: &desc,
			PriceCFA:    tier.priceCFA,
			PriceMode:   "fixed",
			Category:    "event",
			FileKey:     fileKey,
		}, vendorID)
		if err != nil {
			log.Fatalf("création produit %q: %v", tier.title, err)
		}
		log.Printf("Créé (en attente de modération) : %s — %d FCFA — id=%s slug=%s",
			product.Title, product.PriceCFA, product.ID, product.Slug)
	}

	log.Println("Terminé. Les 3 billets sont en attente de modération (/admin/products) avant d'apparaître dans le catalogue public.")
}

// findOrCreateSummitVendor renvoie l'ID du compte vendeur "DIARRA Summit",
// le créant s'il n'existe pas encore — rend la commande idempotente (on
// peut la relancer sans dupliquer le compte).
func findOrCreateSummitVendor(ctx context.Context, userRepo *repository.UserRepo) (string, error) {
	existing, err := userRepo.FindByEmail(ctx, summitVendorEmail)
	if err == nil && existing != nil {
		return existing.ID, nil
	}

	// Mot de passe aléatoire : ce compte sert uniquement de propriétaire
	// catalogue pour les billets, personne ne doit s'y connecter — voir
	// JOURNAL-MODIFICATIONS.md pour changer ça si un jour un vrai accès est
	// nécessaire (réinitialisation de mot de passe classique).
	randomPassword, err := auth.HashPassword(fmt.Sprintf("summit-vendor-%s", storage.NewFileKey("seed", "pw")))
	if err != nil {
		return "", err
	}
	shopName := summitVendorShop
	user, err := userRepo.Create(ctx, model.RegisterInput{
		Email:    summitVendorEmail,
		ShopName: &shopName,
	}, randomPassword)
	if err != nil {
		return "", err
	}
	if err := userRepo.AddRole(ctx, user.ID, model.RoleVendeur); err != nil {
		return "", err
	}
	return user.ID, nil
}
