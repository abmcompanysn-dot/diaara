# Scénarios d'utilisation par rôle — DIARRA

Plateforme de vente de produits numériques (abonnements, comptes, ebooks, PDF).
Rôles : **Client**, **Vendeur**, **Affilié (closer)**, **Administrateur**.

---

## 1. Client

### Scénario A — Acheter un produit
1. Le client ouvre `/catalog`, consulte les produits.
2. Il ouvre un produit sur `/product` (description, prix FCFA, catégorie).
3. Il passe commande sur `/order` : paiement, puis téléchargement instantané du fichier.
4. Il retrouve tous ses achats sur `/orders` (téléchargement à nouveau si besoin).

### Scénario B — Suivre une commande
1. Sur `/orders`, le client voit le statut de chaque commande (pending / paid / delivered / failed).
2. En cas de problème, il ouvre un ticket via `/support` puis `/support/ticket`.

### Scénario C — Compte
1. Inscription sur `/auth/register` avec rôle **client**.
2. Connexion sur `/auth/login`, récupération de mot de passe (`/auth/forgot-password`, `/auth/reset-password`).

---

## 2. Vendeur

### Scénario D — Publier un produit avec affiliation
1. Le vendeur se connecte (rôle `vendeur`).
2. Sur `/vendor/products/new` : titre, description, prix FCFA, catégorie, fichier à vendre.
3. Il coche « Autoriser l'affiliation » et fixe une commission max pour les affiliés (plafonné à 85 %).
4. Publication → le produit apparaît dans le catalogue `/catalog`.

### Scénario E — Suivre ses gains
1. `/dashboard` : tableau de bord filtré (statistiques vendeur).
2. `/vendor/earnings` : détails des gains, ventes et commissions versées.

---

## 3. Affilié (closer)

### Scénario F — Créer et partager un lien d'affiliation
1. L'affilié se connecte (rôle `closer`).
2. Sur `/closer/dashboard`, il choisit un produit affiliable (commission ≤ max définie par le vendeur).
3. Il crée un lien : URL `/r/{slug}` (préfixée par l'origine de l'API).
4. Il copie l'URL et la partage (réseaux, groupes, WhatsApp, etc.).

### Scénario G — Suivre ses performances
1. `/closer/dashboard` : statistiques de ses liens — clics, ventes, commissions gagnées (FCFA).
2. Chaque vente conclue via son lien lui rapporte la commission configurée.

---

## 4. Administrateur

### Scénario H — Superviser la plateforme
1. `/admin` : vue d'ensemble.
2. `/admin/products` : gestion des produits.
3. `/admin/sales` : toutes les ventes — montant, frais plateforme (15 %), commission affilié, part vendeur, statut.

### Scénario I — Gérer les utilisateurs
1. `/admin/users` : liste des comptes avec rôles et statut.
2. Il accorde/retire les rôles `vendeur` / `closer` (cumulables).
3. Il suspend un compte 30 jours en cas d'abus (`locked_until`).

---

## Règles de partage des revenus (rappel)
- Plateforme : 15 % du prix de vente.
- Vendeur : prix − frais plateforme − commission affilié.
- Affilié : commission ≤ max fixée par le vendeur (0 % si pas d'affiliation).
