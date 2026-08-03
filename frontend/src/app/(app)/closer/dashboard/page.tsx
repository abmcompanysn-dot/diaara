'use client';

import { useEffect, useState } from 'react';
import { api, apiOrigin } from '@/lib/api';
import { RequireRole } from '@/lib/guards';
import { LinkIcon, MegaphoneIcon, ZapIcon } from '@/components/icons';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card';

interface ReferralLink {
  id: string;
  product_id: string;
  product_title?: string;
  slug: string;
  commission_pct: number;
  clicks: number;
  sales: number;
  revenue_cfa: number;
  commissions_cfa: number;
  created_at: string;
}

interface Product {
  id: string;
  title: string;
  price_cfa: number;
  affiliate_enabled: boolean;
  max_closer_commission_pct: number;
  moderation_status: string;
}

function CloserDashboard() {
  const [links, setLinks] = useState<ReferralLink[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [selected, setSelected] = useState('');
  const [commission, setCommission] = useState('10');
  const [copied, setCopied] = useState<string | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  const load = async () => {
    setLoading(true);
    try {
      const [linkRes, productRes] = await Promise.all([api.getCloserLinks(), api.getProducts()]);
      setLinks(linkRes.links);
      const affiliateProducts = productRes.products.filter(
        (p: Product) =>
          p.affiliate_enabled && p.moderation_status === 'approved' && p.price_cfa > 0
      );
      setProducts(affiliateProducts);
      if (!selected && affiliateProducts.length > 0) {
        setSelected(affiliateProducts[0].id);
      }
    } catch (err: any) {
      setError(err.message || 'Impossible de charger vos liens');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    try {
      await api.createReferralLink({
        product_id: selected,
        commission_pct: parseFloat(commission),
      });
      setSelected('');
      setCommission('10');
      load();
    } catch (err: any) {
      setError(err.message || 'Échec de la création du lien');
    }
  };

  const shareUrl = (slug: string) => `${apiOrigin}/r/${slug}`;

  const copy = async (slug: string) => {
    try {
      await navigator.clipboard.writeText(shareUrl(slug));
      setCopied(slug);
      setTimeout(() => setCopied(null), 2000);
    } catch {
      alert(shareUrl(slug));
    }
  };

  const selectedProduct = products.find((p) => p.id === selected);

  return (
    <main className="min-h-screen bg-paper">
      <section className="gradient-green text-white relative overflow-hidden">
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="relative max-w-6xl mx-auto px-4 py-16">
          <p className="font-mono text-sm text-green-300 mb-3 uppercase tracking-widest">
            // espace affilié
          </p>
          <h1 className="font-display text-4xl sm:text-5xl font-bold tracking-tight">
            Vos liens d&apos;affiliation
          </h1>
          <p className="mt-3 text-white/75 max-w-lg">
            Partagez un lien, gagnez une commission sur chaque vente. Suivez vos clics et vos revenus ici.
          </p>
        </div>
      </section>

      <section className="max-w-6xl mx-auto px-4 py-12">
        {error && <div className="mb-6 p-3 bg-destructive/10 text-destructive rounded">{error}</div>}

        <div className="grid gap-6 lg:grid-cols-3">
          <div className="lg:col-span-2">
            <div className="flex items-center gap-2 mb-4">
              <LinkIcon size={20} className="text-green-600" />
              <h2 className="font-display font-bold text-xl text-green-950">Mes liens</h2>
            </div>

            {loading ? (
              <p className="text-muted-foreground">Chargement...</p>
            ) : (
              <div className="grid gap-4 sm:grid-cols-2">
                {links.map((link) => (
                  <Card key={link.id} className="shadow-card border-green-900/5">
                    <CardContent className="p-5">
                      <p className="font-semibold text-foreground truncate">
                        {link.product_title || 'Produit'}
                      </p>
                      <Button
                        variant="ghost"
                        onClick={() => copy(link.slug)}
                        className="mt-2 block w-full text-left font-mono text-xs text-primary hover:text-primary/80 truncate"
                        title="Copier le lien"
                      >
                        {copied === link.slug ? 'Copié !' : shareUrl(link.slug)}
                      </Button>
                      <div className="mt-4 grid grid-cols-2 gap-3 text-sm">
                        <div>
                          <p className="text-muted-foreground">Commission</p>
                          <p className="font-semibold">{link.commission_pct}%</p>
                        </div>
                        <div>
                          <p className="text-muted-foreground">Clics</p>
                          <p className="font-semibold">{link.clicks}</p>
                        </div>
                        <div>
                          <p className="text-muted-foreground">Ventes</p>
                          <p className="font-semibold">{link.sales}</p>
                        </div>
                        <div>
                          <p className="text-muted-foreground">Gains</p>
                          <p className="font-semibold text-primary">
                            {link.commissions_cfa.toLocaleString('fr-FR')} FCFA
                          </p>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
                {links.length === 0 && !loading && (
                  <p className="text-muted-foreground">
                    Aucun lien pour le moment. Créez votre premier lien à droite.
                  </p>
                )}
              </div>
            )}
          </div>

          <div>
            <div className="flex items-center gap-2 mb-4">
              <MegaphoneIcon size={20} className="text-green-600" />
              <h2 className="font-display font-bold text-xl text-green-950">Nouveau lien</h2>
            </div>

            <Card className="shadow-card border-green-900/5">
              <CardHeader className="pb-3">
                <CardTitle className="text-base">Générer un lien</CardTitle>
                <CardDescription>
                  Choisissez un produit affiliable et votre commission.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <form onSubmit={handleCreate} className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="closer-product">Produit</Label>
                    <Select value={selected} onValueChange={(v) => setSelected(v || '')}>
                      <SelectTrigger id="closer-product" className="w-full bg-background">
                        <SelectValue placeholder="Choisir un produit..." />
                      </SelectTrigger>
                      <SelectContent>
                        {products.map((p) => (
                          <SelectItem key={p.id} value={p.id}>
                            {p.title} — {p.price_cfa.toLocaleString('fr-FR')} FCFA
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    {products.length === 0 && (
                      <p className="text-xs text-muted-foreground">
                        Aucun produit affiliable actuellement.
                      </p>
                    )}
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="closer-commission">
                      Commission (%){selectedProduct ? ` — max ${selectedProduct.max_closer_commission_pct}%` : ''}
                    </Label>
                    <Input
                      id="closer-commission"
                      type="number"
                      value={commission}
                      onChange={(e) => setCommission(e.target.value)}
                      min={1}
                      max={selectedProduct?.max_closer_commission_pct || 85}
                      required
                    />
                  </div>

                  <Button
                    type="submit"
                    disabled={!selected || products.length === 0}
                    className="w-full font-semibold"
                  >
                    Générer le lien
                  </Button>
                </form>
              </CardContent>
            </Card>

            <div className="mt-6 p-4 bg-green-950 text-white rounded-xl">
              <div className="flex items-center gap-2 mb-2">
                <ZapIcon size={18} className="text-lime" />
                <p className="font-semibold">Comment ça marche ?</p>
              </div>
              <ul className="text-sm text-white/70 space-y-2 list-disc list-inside">
                <li>Le vendeur doit activer l&apos;affiliation sur son produit.</li>
                <li>La commission est plafonnée par le vendeur.</li>
                <li>Chaque vente via votre lien vous rapporte votre commission.</li>
              </ul>
            </div>
          </div>
        </div>
      </section>
    </main>
  );
}

export default function CloserDashboardPage() {
  return (
    <RequireRole roles={['closer']}>
      <CloserDashboard />
    </RequireRole>
  );
}
