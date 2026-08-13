'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
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
import { ArrowLeftIcon, FileIcon } from '@/components/icons';
import { CATEGORY_LABELS } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';

export default function NewProductPage() {
  const router = useRouter();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [price, setPrice] = useState('');
  const [category, setCategory] = useState('ebook');
  const [affiliateEnabled, setAffiliateEnabled] = useState(false);
  const [maxCommission, setMaxCommission] = useState('10');
  const [file, setFile] = useState<File | null>(null);
  const [filePreview, setFilePreview] = useState('');
  const [coverFile, setCoverFile] = useState<File | null>(null);
  const [coverKey, setCoverKey] = useState('');
  const [coverPreview, setCoverPreview] = useState('');
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0] || null;
    setFile(f);
    if (filePreview) URL.revokeObjectURL(filePreview);
    setFilePreview(f ? URL.createObjectURL(f) : '');
  };

  const handleCoverSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0] || null;
    setCoverFile(f);
    if (!f) {
      setCoverKey('');
      setCoverPreview('');
      return;
    }
    setCoverPreview(URL.createObjectURL(f));
    try {
      const form = new FormData();
      form.append('file', f);
      form.append('type', 'cover');
      const res = await api.uploadFile(form);
      setCoverKey(res.file_key);
    } catch (err: any) {
      setError(friendlyError(err));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!file) {
      setError('Veuillez sélectionner un fichier');
      return;
    }
    const priceNum = parseInt(price, 10);
    if (isNaN(priceNum) || priceNum < 0) {
      setError('Prix invalide');
      return;
    }

    setUploading(true);
    try {
      // Étape 1 : upload du fichier
      const uploadForm = new FormData();
      uploadForm.append('file', file);
      const uploadResult = await api.uploadFile(uploadForm);

      // Étape 2 : création du produit
      await api.createProduct({
        title,
        description,
        price_cfa: priceNum,
        category,
        file_key: uploadResult.file_key,
        cover_image_key: coverKey || undefined,
        affiliate_enabled: affiliateEnabled,
        max_closer_commission_pct: affiliateEnabled ? parseInt(maxCommission, 10) || 0 : 0,
      });
      router.push('/vendor/products');
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setUploading(false);
    }
  };

  return (
    <main className="min-h-screen">
      <PageHeader
        eyebrow="// espace vendeur"
        title="Nouveau produit"
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

      <Card className="shadow-card border-green-900/5">
        <CardHeader>
          <CardTitle>Informations produit</CardTitle>
          <CardDescription>
            Le fichier est stocké localement et livré instantanément après paiement.
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
              <Label htmlFor="file">Fichier à vendre *</Label>
              <Input
                id="file"
                type="file"
                onChange={handleFileSelect}
                required
              />
              {file && (
                <div className="rounded-lg border border-border overflow-hidden">
                  {filePreview &&
                  (file.type.startsWith('image/') || file.type === 'application/pdf' || file.type.includes('video/')) ? (
                    file.type === 'application/pdf' ? (
                      <iframe
                        src={filePreview}
                        title="Aperçu du document"
                        className="w-full h-64 bg-white"
                      />
                    ) : file.type.startsWith('image/') ? (
                      <img
                        src={filePreview}
                        alt="Aperçu du fichier"
                        className="w-full h-40 object-contain bg-white"
                      />
                    ) : (
                      <video
                        src={filePreview}
                        controls
                        className="w-full h-40 bg-black"
                      />
                    )
                  ) : (
                    <div className="p-3 flex items-center gap-3 bg-secondary/60">
                      <FileIcon size={28} className="text-primary shrink-0" />
                      <div className="min-w-0">
                        <p className="text-sm font-medium truncate">{file.name}</p>
                        <p className="text-xs text-muted-foreground">
                          {(file.size / 1024 / 1024).toFixed(2)} Mo
                        </p>
                      </div>
                    </div>
                  )}
                </div>
              )}
              <p className="text-xs text-muted-foreground">Taille max : 50 Mo</p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="cover">Image de couverture (optionnel)</Label>
              {coverPreview && (
                <img
                  src={coverPreview}
                  alt="Aperçu de la couverture"
                  className="w-full h-40 object-cover rounded-lg border border-border"
                />
              )}
              <Input
                id="cover"
                type="file"
                accept="image/*"
                onChange={handleCoverSelect}
              />
              <p className="text-xs text-muted-foreground">
                PNG, JPG ou WebP. La carte du catalogue affichera cette image.
              </p>
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
                    Les affiliés (closers) pourront créer un lien de partage et gagner une commission.
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

            <Button type="submit" disabled={uploading} className="w-full font-semibold">
              {uploading ? 'Publication en cours...' : 'Publier le produit'}
            </Button>
          </form>
        </CardContent>
      </Card>
      </section>
    </main>
  );
}
