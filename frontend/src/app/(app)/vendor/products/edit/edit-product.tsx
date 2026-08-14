'use client';

import { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { Badge } from '@/components/ui/badge';
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
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { ArrowLeftIcon, FileIcon } from '@/components/icons';
import { CATEGORY_LABELS, PRODUCT_STATUS_BADGE, PRODUCT_STATUS_LABELS } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';

export default function EditProduct() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const id = searchParams.get('id') || '';

  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [status, setStatus] = useState('');

  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [price, setPrice] = useState('');
  const [flexiblePrice, setFlexiblePrice] = useState(false);
  const [minPrice, setMinPrice] = useState('');
  const [category, setCategory] = useState('ebook');
  const [affiliateEnabled, setAffiliateEnabled] = useState(false);
  const [maxCommission, setMaxCommission] = useState('10');

  const [newFile, setNewFile] = useState<File | null>(null);
  const [uploadingFile, setUploadingFile] = useState(false);

  useEffect(() => {
    if (!id) return;
    api
      .getProduct(id)
      .then(({ product }) => {
        setTitle(product.title);
        setDescription(product.description || '');
        setPrice(String(product.price_cfa));
        setFlexiblePrice(product.price_mode === 'flexible');
        setMinPrice(product.min_price_cfa ? String(product.min_price_cfa) : '');
        setCategory(product.category);
        setAffiliateEnabled(product.affiliate_enabled);
        setMaxCommission(String(product.max_closer_commission_pct || 10));
        setStatus(product.moderation_status);
      })
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  }, [id]);

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    setNewFile(e.target.files?.[0] || null);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    const priceNum = parseInt(price, 10);
    if (isNaN(priceNum) || priceNum < 0) {
      setError('Prix invalide');
      return;
    }
    const minPriceNum = parseInt(minPrice, 10);
    if (flexiblePrice && (isNaN(minPriceNum) || minPriceNum <= 0)) {
      setError('Indiquez un prix minimum pour le prix libre');
      return;
    }

    setSaving(true);
    try {
      let fileKey: string | undefined;
      if (newFile) {
        setUploadingFile(true);
        const form = new FormData();
        form.append('file', newFile);
        const uploadResult = await api.uploadFile(form);
        fileKey = uploadResult.file_key;
        setUploadingFile(false);
      }

      await api.updateProduct(id, {
        title,
        description,
        price_cfa: priceNum,
        price_mode: flexiblePrice ? 'flexible' : 'fixed',
        min_price_cfa: flexiblePrice ? minPriceNum : undefined,
        category,
        ...(fileKey ? { file_key: fileKey } : {}),
        affiliate_enabled: affiliateEnabled,
        max_closer_commission_pct: affiliateEnabled ? parseInt(maxCommission, 10) || 0 : 0,
      });
      router.push('/vendor/products');
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSaving(false);
      setUploadingFile(false);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// espace vendeur" title="Modifier le produit" />
        <PageLoader />
      </main>
    );

  return (
    <main className="min-h-screen">
      <PageHeader
        eyebrow="// espace vendeur"
        title="Modifier le produit"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/vendor/products" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Mes produits
          </Button>
        }
      />

      <section className="max-w-2xl mx-auto px-4 sm:px-6 py-10">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {status === 'approved' && (
          <div className="mb-4 p-3 bg-amber-50 border border-amber-200 text-amber-800 rounded-lg text-sm">
            <span className="font-semibold">Attention —</span> ce produit est actuellement{' '}
            <Badge className={PRODUCT_STATUS_BADGE.approved}>
              {PRODUCT_STATUS_LABELS.approved}
            </Badge>{' '}
            et visible dans le catalogue. Enregistrer une modification le repassera en{' '}
            <Badge className={PRODUCT_STATUS_BADGE.pending}>{PRODUCT_STATUS_LABELS.pending}</Badge>{' '}
            le temps qu&apos;un administrateur le revalide.
          </div>
        )}

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Informations produit</CardTitle>
            <CardDescription>
              Modifiez les informations du produit, ou remplacez le fichier livré ci-dessous.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="title">Titre *</Label>
                <Input
                  id="title"
                  type="text"
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="min-h-32"
                />
              </div>

              <div className="grid sm:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="price">Prix (FCFA) *</Label>
                  <Input
                    id="price"
                    type="number"
                    value={price}
                    onChange={(e) => setPrice(e.target.value)}
                    required
                    min={0}
                  />
                  <label className="flex items-center gap-2 cursor-pointer mt-1">
                    <Checkbox checked={flexiblePrice} onCheckedChange={(checked) => setFlexiblePrice(checked === true)} />
                    <span className="text-xs text-muted-foreground">Prix libre (le client choisit son montant)</span>
                  </label>
                  {flexiblePrice && (
                    <div className="space-y-1">
                      <Label htmlFor="min-price" className="text-xs">Prix minimum (FCFA)</Label>
                      <Input
                        id="min-price"
                        type="number"
                        value={minPrice}
                        onChange={(e) => setMinPrice(e.target.value)}
                        min={1}
                      />
                    </div>
                  )}
                </div>

                <div className="space-y-2">
                  <Label htmlFor="category">Catégorie *</Label>
                  <Select value={category} onValueChange={(v) => setCategory(v || '')}>
                    <SelectTrigger id="category" className="w-full bg-background">
                      <SelectValue placeholder="Choisir une catégorie" />
                    </SelectTrigger>
                    <SelectContent>
                      {Object.entries(CATEGORY_LABELS).map(([value, label]) => (
                        <SelectItem key={value} value={value}>
                          {label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="new-file">Remplacer le fichier livré (optionnel)</Label>
                <Input id="new-file" type="file" onChange={handleFileSelect} />
                {newFile ? (
                  <div className="p-3 flex items-center gap-3 bg-secondary/60 rounded-lg border border-border">
                    <FileIcon size={24} className="text-primary shrink-0" />
                    <div className="min-w-0">
                      <p className="text-sm font-medium truncate">{newFile.name}</p>
                      <p className="text-xs text-muted-foreground">
                        {(newFile.size / 1024 / 1024).toFixed(2)} Mo — remplacera le fichier actuel
                      </p>
                    </div>
                  </div>
                ) : (
                  <p className="text-xs text-muted-foreground">
                    Laissez vide pour conserver le fichier actuel. Un nouveau fichier régénère les aperçus.
                  </p>
                )}
              </div>

              <div className="p-4 border border-border rounded-lg bg-secondary/60 space-y-3">
                <label className="flex items-start gap-3 cursor-pointer">
                  <Checkbox
                    checked={affiliateEnabled}
                    onCheckedChange={(checked) => setAffiliateEnabled(checked === true)}
                    className="mt-1"
                  />
                  <span>
                    <span className="block text-sm font-medium">Autoriser l&apos;affiliation</span>
                    <span className="block text-xs text-muted-foreground">
                      Les affiliés (closers) pourront créer un lien de partage et gagner une
                      commission.
                    </span>
                  </span>
                </label>

                {affiliateEnabled && (
                  <div className="ml-7 space-y-2">
                    <Label htmlFor="max-commission">Commission closeur max (%)</Label>
                    <Input
                      id="max-commission"
                      type="number"
                      value={maxCommission}
                      onChange={(e) => setMaxCommission(e.target.value)}
                      min={1}
                      max={85}
                    />
                    <p className="text-xs text-muted-foreground">
                      Plafonné à 85 %. La plateforme garde 15 % du prix de vente.
                    </p>
                  </div>
                )}
              </div>

              <Button type="submit" disabled={saving} className="w-full font-semibold">
                {uploadingFile ? 'Envoi du fichier...' : saving ? 'Enregistrement...' : 'Enregistrer les modifications'}
              </Button>
            </form>
          </CardContent>
        </Card>
      </section>
    </main>
  );
}
