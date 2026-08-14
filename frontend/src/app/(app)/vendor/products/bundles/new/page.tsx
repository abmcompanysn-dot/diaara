'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/page-header';
import { ArrowLeftIcon } from '@/components/icons';
import { formatPrice } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';

interface Product {
  id: string;
  title: string;
  price_cfa: number;
}

export default function NewBundlePage() {
  const router = useRouter();
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [price, setPrice] = useState('');
  const [selected, setSelected] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    api
      .getVendorProducts()
      .then((r) => setProducts(r.products))
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  }, []);

  const toggle = (id: string) => {
    setSelected((prev) => (prev.includes(id) ? prev.filter((p) => p !== id) : [...prev, id]));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    const priceNum = parseInt(price, 10);
    if (isNaN(priceNum) || priceNum <= 0) {
      setError('Prix invalide');
      return;
    }
    if (selected.length < 2) {
      setError('Sélectionnez au moins 2 produits pour créer un pack');
      return;
    }

    setSaving(true);
    try {
      await api.createBundle({ title, description, price_cfa: priceNum, product_ids: selected });
      router.push('/vendor/products/bundles');
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <main className="min-h-screen">
      <PageHeader
        eyebrow="// espace vendeur"
        title="Créer un pack"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/vendor/products/bundles" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Mes packs
          </Button>
        }
      />

      <section className="max-w-2xl mx-auto px-4 sm:px-6 py-10">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Nouveau pack</CardTitle>
            <CardDescription>Regroupez plusieurs de vos produits déjà publiés en un seul article.</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="bundle-title">Titre du pack *</Label>
                <Input id="bundle-title" value={title} onChange={(e) => setTitle(e.target.value)} required />
              </div>

              <div className="space-y-2">
                <Label htmlFor="bundle-description">Description</Label>
                <Textarea
                  id="bundle-description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="min-h-24"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="bundle-price">Prix du pack (FCFA) *</Label>
                <Input
                  id="bundle-price"
                  type="number"
                  value={price}
                  onChange={(e) => setPrice(e.target.value)}
                  required
                  min={0}
                />
              </div>

              <div className="space-y-2">
                <Label>Produits inclus * (minimum 2)</Label>
                {loading ? (
                  <p className="text-sm text-muted-foreground">Chargement…</p>
                ) : products.length === 0 ? (
                  <p className="text-sm text-muted-foreground">Publiez d&apos;abord des produits pour pouvoir les regrouper en pack.</p>
                ) : (
                  <div className="space-y-2 max-h-72 overflow-y-auto border border-border rounded-lg p-2">
                    {products.map((p) => (
                      <label key={p.id} className="flex items-center gap-3 p-2 rounded hover:bg-secondary/60 cursor-pointer">
                        <Checkbox checked={selected.includes(p.id)} onCheckedChange={() => toggle(p.id)} />
                        <span className="flex-1 text-sm">{p.title}</span>
                        <span className="text-xs font-mono text-muted-foreground">{formatPrice(p.price_cfa)}</span>
                      </label>
                    ))}
                  </div>
                )}
              </div>

              <Button type="submit" disabled={saving} className="w-full font-semibold">
                {saving ? 'Création…' : 'Créer le pack'}
              </Button>
            </form>
          </CardContent>
        </Card>
      </section>
    </main>
  );
}
