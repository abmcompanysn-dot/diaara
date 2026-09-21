'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { ArrowLeftIcon, TrashIcon, EditIcon } from '@/components/icons';
import { formatPrice } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';

interface Tier {
  id: string;
  name: string;
  price_cfa: number;
  perks: string[];
  highlight: boolean;
  sort_order: number;
}

interface Sponsor {
  id: string;
  tier_id?: string | null;
  name: string;
  logo_key?: string | null;
  website_url?: string | null;
  published: boolean;
  sort_order: number;
}

// Formulaire d'édition d'un palier — modale simplifiée en ligne plutôt qu'un
// dialog séparé, le nombre de champs reste petit (nom, prix, avantages).
function TierForm({
  tier,
  onSave,
  onCancel,
}: {
  tier: Partial<Tier>;
  onSave: (data: { name: string; price_cfa: number; perks: string[]; highlight: boolean }) => Promise<void>;
  onCancel: () => void;
}) {
  const [name, setName] = useState(tier.name || '');
  const [price, setPrice] = useState(String(tier.price_cfa ?? ''));
  const [perksText, setPerksText] = useState((tier.perks || []).join('\n'));
  const [highlight, setHighlight] = useState(tier.highlight || false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim() || !price) {
      setError('Nom et prix requis.');
      return;
    }
    setSaving(true);
    setError('');
    try {
      await onSave({
        name: name.trim(),
        price_cfa: parseInt(price, 10),
        perks: perksText.split('\n').map((p) => p.trim()).filter(Boolean),
        highlight,
      });
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3 pt-3 border-t border-border">
      <div className="space-y-2">
        <Label>Nom du palier</Label>
        <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Ex: Bronze" />
      </div>
      <div className="space-y-2">
        <Label>Prix (FCFA)</Label>
        <Input type="number" value={price} onChange={(e) => setPrice(e.target.value)} min={0} />
      </div>
      <div className="space-y-2">
        <Label>Avantages (un par ligne)</Label>
        <Textarea value={perksText} onChange={(e) => setPerksText(e.target.value)} className="min-h-28" />
      </div>
      <label className="flex items-center gap-2 cursor-pointer">
        <Checkbox checked={highlight} onCheckedChange={(c) => setHighlight(c === true)} />
        <span className="text-sm">Mettre en avant (badge « Le plus choisi »)</span>
      </label>
      {error && <p className="text-xs text-red-600">{error}</p>}
      <div className="flex gap-2">
        <Button type="submit" size="sm" disabled={saving} className="font-semibold">
          {saving ? 'Enregistrement...' : 'Enregistrer'}
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={onCancel}>
          Annuler
        </Button>
      </div>
    </form>
  );
}

function SponsorForm({
  sponsor,
  tiers,
  onSave,
  onCancel,
}: {
  sponsor: Partial<Sponsor>;
  tiers: Tier[];
  onSave: (data: { name: string; tier_id?: string; website_url?: string; published: boolean; logo_key?: string }) => Promise<void>;
  onCancel: () => void;
}) {
  const [name, setName] = useState(sponsor.name || '');
  const [tierId, setTierId] = useState(sponsor.tier_id || '');
  const [websiteUrl, setWebsiteUrl] = useState(sponsor.website_url || '');
  const [published, setPublished] = useState(sponsor.published ?? true);
  const [logoKey, setLogoKey] = useState(sponsor.logo_key || '');
  const [logoPreview, setLogoPreview] = useState('');
  const [uploading, setUploading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const handleLogoSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0] || null;
    if (!f) return;
    setLogoPreview(URL.createObjectURL(f));
    setUploading(true);
    setError('');
    try {
      const form = new FormData();
      form.append('file', f);
      form.append('type', 'cover');
      const res = await api.adminUploadSponsorLogo(form);
      setLogoKey(res.file_key);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setUploading(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) {
      setError("Le nom de l'entreprise est requis.");
      return;
    }
    setSaving(true);
    setError('');
    try {
      await onSave({
        name: name.trim(),
        tier_id: tierId || undefined,
        website_url: websiteUrl.trim() || undefined,
        published,
        logo_key: logoKey || undefined,
      });
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3 pt-3 border-t border-border">
      <div className="space-y-2">
        <Label>Nom de l&rsquo;entreprise</Label>
        <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Ex: ABMCY" />
      </div>
      <div className="space-y-2">
        <Label>Palier</Label>
        <select
          value={tierId}
          onChange={(e) => setTierId(e.target.value)}
          className="w-full h-9 px-3 rounded-md border border-input bg-background text-sm"
        >
          <option value="">Aucun</option>
          {tiers.map((t) => (
            <option key={t.id} value={t.id}>
              {t.name} — {formatPrice(t.price_cfa)}
            </option>
          ))}
        </select>
      </div>
      <div className="space-y-2">
        <Label>Site web (optionnel)</Label>
        <Input type="url" value={websiteUrl} onChange={(e) => setWebsiteUrl(e.target.value)} placeholder="https://..." />
      </div>
      <div className="space-y-2">
        <Label>Logo</Label>
        {(logoPreview || logoKey) && (
          <img
            src={logoPreview || undefined}
            alt="Aperçu du logo"
            className="w-16 h-16 object-cover rounded-lg border border-border"
          />
        )}
        <Input type="file" accept="image/*" onChange={handleLogoSelect} />
        {uploading && <p className="text-xs text-muted-foreground">Envoi en cours...</p>}
      </div>
      <label className="flex items-center gap-2 cursor-pointer">
        <Checkbox checked={published} onCheckedChange={(c) => setPublished(c === true)} />
        <span className="text-sm">Publié (visible sur /summit/sponsors)</span>
      </label>
      {error && <p className="text-xs text-red-600">{error}</p>}
      <div className="flex gap-2">
        <Button type="submit" size="sm" disabled={saving || uploading} className="font-semibold">
          {saving ? 'Enregistrement...' : 'Enregistrer'}
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={onCancel}>
          Annuler
        </Button>
      </div>
    </form>
  );
}

export default function AdminSummitSponsorsPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tiers, setTiers] = useState<Tier[]>([]);
  const [sponsors, setSponsors] = useState<Sponsor[]>([]);

  const [addingTier, setAddingTier] = useState(false);
  const [editingTierId, setEditingTierId] = useState<string | null>(null);
  const [addingSponsor, setAddingSponsor] = useState(false);
  const [editingSponsorId, setEditingSponsorId] = useState<string | null>(null);
  const [confirmDeleteTier, setConfirmDeleteTier] = useState<Tier | null>(null);
  const [confirmDeleteSponsor, setConfirmDeleteSponsor] = useState<Sponsor | null>(null);

  const load = () => {
    Promise.all([api.adminListSponsorTiers(), api.adminListSponsors()])
      .then(([t, s]) => {
        setTiers(t.tiers || []);
        setSponsors(s.sponsors || []);
      })
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const tierById = (id?: string | null) => tiers.find((t) => t.id === id);

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// administration" title="Sponsors DIARRA Summit" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Sponsors DIARRA Summit"
        description="Paliers de sponsoring et entreprises sponsors affichés sur /summit/sponsors"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/admin" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Tableau de bord
          </Button>
        }
      />

      <section className="max-w-2xl mx-auto px-4 sm:px-6 py-10 space-y-6">
        {error && (
          <div className="p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Paliers</CardTitle>
            <CardDescription>Prix et avantages affichés dans les cartes de /summit/sponsors.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {tiers.map((tier) =>
              editingTierId === tier.id ? (
                <TierForm
                  key={tier.id}
                  tier={tier}
                  onCancel={() => setEditingTierId(null)}
                  onSave={async (data) => {
                    await api.adminUpdateSponsorTier(tier.id, data);
                    setEditingTierId(null);
                    load();
                  }}
                />
              ) : (
                <div key={tier.id} className="p-3 rounded-lg border border-border">
                  <div className="flex items-center justify-between gap-3">
                    <div className="min-w-0">
                      <p className="text-sm font-semibold truncate">
                        {tier.name} {tier.highlight && <Badge className="ml-1 bg-lime/20 text-green-800">Mis en avant</Badge>}
                      </p>
                      <p className="text-xs text-muted-foreground font-mono">{formatPrice(tier.price_cfa)}</p>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <button
                        type="button"
                        onClick={() => setEditingTierId(tier.id)}
                        className="text-muted-foreground hover:text-foreground"
                        aria-label={`Modifier ${tier.name}`}
                      >
                        <EditIcon size={16} />
                      </button>
                      <button
                        type="button"
                        onClick={() => setConfirmDeleteTier(tier)}
                        className="text-red-600 hover:text-red-700"
                        aria-label={`Supprimer ${tier.name}`}
                      >
                        <TrashIcon size={16} />
                      </button>
                    </div>
                  </div>
                  <ul className="mt-2 text-xs text-muted-foreground list-disc pl-4 space-y-0.5">
                    {tier.perks.map((p) => (
                      <li key={p}>{p}</li>
                    ))}
                  </ul>
                </div>
              )
            )}

            {addingTier ? (
              <TierForm
                tier={{}}
                onCancel={() => setAddingTier(false)}
                onSave={async (data) => {
                  await api.adminCreateSponsorTier({ ...data, sort_order: tiers.length });
                  setAddingTier(false);
                  load();
                }}
              />
            ) : (
              <Button variant="outline" size="sm" onClick={() => setAddingTier(true)}>
                + Ajouter un palier
              </Button>
            )}
          </CardContent>
        </Card>

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Sponsors</CardTitle>
            <CardDescription>
              Entreprises affichées dans la section « écosystème » de /summit/sponsors. Seuls les sponsors publiés sont
              visibles publiquement.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {sponsors.map((sponsor) =>
              editingSponsorId === sponsor.id ? (
                <SponsorForm
                  key={sponsor.id}
                  sponsor={sponsor}
                  tiers={tiers}
                  onCancel={() => setEditingSponsorId(null)}
                  onSave={async (data) => {
                    await api.adminUpdateSponsor(sponsor.id, data);
                    setEditingSponsorId(null);
                    load();
                  }}
                />
              ) : (
                <div key={sponsor.id} className="flex items-center justify-between gap-3 p-3 rounded-lg border border-border">
                  <div className="flex items-center gap-3 min-w-0">
                    {sponsor.logo_key ? (
                      <img
                        src={`/api/summit/sponsors/${sponsor.id}/logo`}
                        alt={sponsor.name}
                        className="w-10 h-10 rounded-lg object-cover border border-border shrink-0"
                      />
                    ) : (
                      <div className="w-10 h-10 rounded-lg bg-secondary shrink-0" />
                    )}
                    <div className="min-w-0">
                      <p className="text-sm font-semibold truncate">{sponsor.name}</p>
                      <p className="text-xs text-muted-foreground truncate">
                        {tierById(sponsor.tier_id)?.name || 'Sans palier'}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    {sponsor.published ? (
                      <Badge className="bg-green-100 text-green-700">Publié</Badge>
                    ) : (
                      <Badge className="bg-amber-100 text-amber-800">Brouillon</Badge>
                    )}
                    <button
                      type="button"
                      onClick={() => setEditingSponsorId(sponsor.id)}
                      className="text-muted-foreground hover:text-foreground"
                      aria-label={`Modifier ${sponsor.name}`}
                    >
                      <EditIcon size={16} />
                    </button>
                    <button
                      type="button"
                      onClick={() => setConfirmDeleteSponsor(sponsor)}
                      className="text-red-600 hover:text-red-700"
                      aria-label={`Supprimer ${sponsor.name}`}
                    >
                      <TrashIcon size={16} />
                    </button>
                  </div>
                </div>
              )
            )}

            {addingSponsor ? (
              <SponsorForm
                sponsor={{ published: true }}
                tiers={tiers}
                onCancel={() => setAddingSponsor(false)}
                onSave={async (data) => {
                  await api.adminCreateSponsor({ ...data, sort_order: sponsors.length });
                  setAddingSponsor(false);
                  load();
                }}
              />
            ) : (
              <Button variant="outline" size="sm" onClick={() => setAddingSponsor(true)}>
                + Ajouter un sponsor
              </Button>
            )}
          </CardContent>
        </Card>
      </section>

      <ConfirmDialog
        open={!!confirmDeleteTier}
        title="Supprimer ce palier ?"
        description={`« ${confirmDeleteTier?.name} » ne sera plus proposé sur /summit/sponsors.`}
        confirmLabel="Supprimer"
        cancelLabel="Annuler"
        danger
        onConfirm={async () => {
          if (!confirmDeleteTier) return;
          await api.adminDeleteSponsorTier(confirmDeleteTier.id);
          setConfirmDeleteTier(null);
          load();
        }}
        onCancel={() => setConfirmDeleteTier(null)}
      />

      <ConfirmDialog
        open={!!confirmDeleteSponsor}
        title="Supprimer ce sponsor ?"
        description={`« ${confirmDeleteSponsor?.name} » disparaîtra de /summit/sponsors.`}
        confirmLabel="Supprimer"
        cancelLabel="Annuler"
        danger
        onConfirm={async () => {
          if (!confirmDeleteSponsor) return;
          await api.adminDeleteSponsor(confirmDeleteSponsor.id);
          setConfirmDeleteSponsor(null);
          load();
        }}
        onCancel={() => setConfirmDeleteSponsor(null)}
      />
    </main>
  );
}
