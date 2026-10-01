'use client';

// Espace vendeur → Mes pubs : sponsoriser un produit sur Facebook et
// Instagram en quelques clics, payé avec le solde de gains. DIARRA lance la
// pub sur son propre compte publicitaire Meta ; refus Meta = remboursement
// automatique (voir backend handler/ad_handler.go).

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { api, ApiError } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { ProductImage } from '@/components/product-image';
import { formatPrice } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';
import {
  AD_STATUS_BADGE,
  AD_STATUS_LABELS,
  ctr,
  minAmountFor,
  roundUpThousand,
  splitAmount,
  type AdCampaign,
} from '@/lib/ads';

interface AdsConfig {
  enabled: boolean;
  commission_pct: number;
  min_daily_cfa: number;
  max_duration_days: number;
  max_amount_cfa: number;
  available_cfa: number;
  countries: { code: string; name: string }[];
}

interface VendorProduct {
  id: string;
  title: string;
  description?: string;
  category: string;
  cover_image_key?: string;
  moderation_status: string;
  deletion_requested?: boolean;
}

const DURATIONS = [3, 7, 14, 30];

// Pays de diffusion présélectionnés : ceux où DIARRA vend le plus.
const DEFAULT_COUNTRIES = ['SN', 'CI'];

export default function VendorAds() {
  const searchParams = useSearchParams();
  const [config, setConfig] = useState<AdsConfig | null>(null);
  const [products, setProducts] = useState<VendorProduct[]>([]);
  const [campaigns, setCampaigns] = useState<AdCampaign[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [msg, setMsg] = useState('');

  const [productId, setProductId] = useState('');
  const [days, setDays] = useState(7);
  const [amount, setAmount] = useState(0);
  const [countries, setCountries] = useState<string[]>(DEFAULT_COUNTRIES);
  const [message, setMessage] = useState('');
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function load() {
    setLoading(true);
    try {
      const [cfg, prods, ads] = await Promise.all([api.getAdsConfig(), api.getVendorProducts(), api.getVendorAds()]);
      setConfig(cfg);
      const approved = (prods.products || []).filter(
        (p: VendorProduct) => p.moderation_status === 'approved' && !p.deletion_requested
      );
      setProducts(approved);
      setCampaigns(ads.campaigns || []);
      const wanted = searchParams.get('product');
      const initial = approved.find((p: VendorProduct) => p.id === wanted) || approved[0];
      if (initial) {
        setProductId(initial.id);
        setMessage(initial.title);
      }
      setAmount(roundUpThousand(minAmountFor(7, cfg.min_daily_cfa, cfg.commission_pct)));
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  }

  const product = products.find((p) => p.id === productId);
  const minAmount = config ? minAmountFor(days, config.min_daily_cfa, config.commission_pct) : 0;
  const { commission, adBudget } = splitAmount(amount || 0, config?.commission_pct || 0);
  const presets = useMemo(() => {
    if (!config) return [];
    const base = roundUpThousand(minAmount);
    return Array.from(new Set([base, roundUpThousand(base * 2), roundUpThousand(base * 4)]));
  }, [config, minAmount]);

  const problem = !config
    ? ''
    : !product
      ? 'Choisissez un produit.'
      : countries.length === 0
        ? 'Choisissez au moins un pays.'
        : amount < minAmount
          ? `Minimum ${formatPrice(roundUpThousand(minAmount))} pour ${days} jours.`
          : amount > config.max_amount_cfa
            ? `Maximum ${formatPrice(config.max_amount_cfa)}.`
            : amount > config.available_cfa
              ? `Solde insuffisant : vous avez ${formatPrice(config.available_cfa)} disponibles.`
              : '';

  function pickDays(d: number) {
    setDays(d);
    if (config) {
      const min = roundUpThousand(minAmountFor(d, config.min_daily_cfa, config.commission_pct));
      if (amount < min) setAmount(min);
    }
  }

  function toggleCountry(code: string) {
    setCountries((prev) => (prev.includes(code) ? prev.filter((c) => c !== code) : [...prev, code]));
  }

  async function submit() {
    setConfirmOpen(false);
    setSubmitting(true);
    setError('');
    setMsg('');
    try {
      await api.createVendorAd({
        product_id: productId,
        amount_cfa: amount,
        duration_days: days,
        countries,
        message: message.trim(),
      });
      setMsg(
        'Votre pub est envoyée à Meta. Elle passe en vérification (en général quelques heures), puis la diffusion démarre automatiquement. En cas de refus, vous êtes remboursé.'
      );
      await load();
    } catch (err: any) {
      const details = err instanceof ApiError ? err.body?.details : undefined;
      const min = err instanceof ApiError ? err.body?.min_amount_cfa : undefined;
      let text = friendlyError(err);
      if (typeof details === 'string' && details) text += ` (${details})`;
      if (typeof min === 'number') text += ` Minimum : ${formatPrice(min)}.`;
      setError(text);
    } finally {
      setSubmitting(false);
    }
  }

  if (loading)
    return (
      <main>
        <PageHeader back="/vendor/products" eyebrow="// espace vendeur" title="Mes pubs" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        back="/vendor/products"
        eyebrow="// espace vendeur"
        title="Mes pubs"
        description="Sponsorisez vos produits sur Facebook et Instagram, payé avec votre solde"
      />

      <section className="max-w-5xl mx-auto px-4 sm:px-6 py-6 space-y-6">
        {error && (
          <div className="p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}
        {msg && (
          <div className="p-3 bg-green-900/5 text-green-900 rounded text-sm" role="status">
            {msg}
          </div>
        )}

        {config && !config.enabled ? (
          <div className="bg-white rounded-xl border border-green-900/10 shadow-card p-6 text-center space-y-2">
            <p className="text-lg font-semibold text-green-950">Bientôt disponible</p>
            <p className="text-sm text-green-900/60 max-w-md mx-auto">
              Vous pourrez bientôt faire la publicité de vos produits sur Facebook et Instagram directement depuis DIARRA,
              payée avec votre solde de gains.
            </p>
          </div>
        ) : products.length === 0 ? (
          <EmptyState
            title="Aucun produit à sponsoriser"
            description="Seuls les produits validés par DIARRA peuvent être sponsorisés."
            action={<Button render={<Link href="/vendor/products" />}>Voir mes produits</Button>}
          />
        ) : (
          config && (
            <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
              {/* Formulaire */}
              <div className="bg-white rounded-xl border border-green-900/10 shadow-card p-5 space-y-5">
                <div>
                  <h2 className="text-base font-semibold text-green-950">Nouvelle pub Facebook & Instagram</h2>
                  <p className="text-xs text-green-900/60 mt-0.5">
                    Solde disponible : <strong>{formatPrice(config.available_cfa)}</strong>
                  </p>
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="ad-product">Produit</Label>
                  <select
                    id="ad-product"
                    value={productId}
                    onChange={(e) => {
                      setProductId(e.target.value);
                      const p = products.find((x) => x.id === e.target.value);
                      if (p) setMessage(p.title);
                    }}
                    className="h-10 w-full rounded-md border border-input bg-white px-3 text-sm"
                  >
                    {products.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.title}
                      </option>
                    ))}
                  </select>
                </div>

                <div className="space-y-1.5">
                  <Label>Durée</Label>
                  <div className="flex flex-wrap gap-2">
                    {DURATIONS.filter((d) => d <= config.max_duration_days).map((d) => (
                      <button
                        key={d}
                        type="button"
                        onClick={() => pickDays(d)}
                        className={`rounded-full border px-4 py-1.5 text-sm ${
                          days === d ? 'border-green-800 bg-green-800 text-white' : 'border-green-900/20 text-green-950 hover:bg-green-900/5'
                        }`}
                      >
                        {d} jours
                      </button>
                    ))}
                  </div>
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="ad-amount">Budget total</Label>
                  <div className="flex flex-wrap gap-2">
                    {presets.map((p) => (
                      <button
                        key={p}
                        type="button"
                        onClick={() => setAmount(p)}
                        className={`rounded-full border px-4 py-1.5 text-sm ${
                          amount === p ? 'border-green-800 bg-green-800 text-white' : 'border-green-900/20 text-green-950 hover:bg-green-900/5'
                        }`}
                      >
                        {formatPrice(p)}
                      </button>
                    ))}
                  </div>
                  <Input
                    id="ad-amount"
                    type="number"
                    min={0}
                    step={500}
                    value={amount || ''}
                    onChange={(e) => setAmount(parseInt(e.target.value, 10) || 0)}
                    placeholder="Autre montant (FCFA)"
                    className="bg-white mt-2"
                  />
                </div>

                <div className="space-y-1.5">
                  <Label>Pays de diffusion</Label>
                  <div className="flex flex-wrap gap-1.5">
                    {config.countries.map((c) => (
                      <button
                        key={c.code}
                        type="button"
                        onClick={() => toggleCountry(c.code)}
                        className={`rounded-full border px-3 py-1 text-xs ${
                          countries.includes(c.code)
                            ? 'border-green-800 bg-green-800 text-white'
                            : 'border-green-900/20 text-green-950 hover:bg-green-900/5'
                        }`}
                      >
                        {c.name}
                      </button>
                    ))}
                  </div>
                </div>

                <div className="space-y-1.5">
                  <Label htmlFor="ad-message">Texte de la pub</Label>
                  <Textarea
                    id="ad-message"
                    value={message}
                    onChange={(e) => setMessage(e.target.value.slice(0, 500))}
                    rows={3}
                    className="bg-white"
                    placeholder="Ex : Apprenez à lancer votre business en 30 jours. Guide complet, téléchargement immédiat."
                  />
                  <p className="text-[11px] text-green-900/40 text-right">{message.length}/500</p>
                </div>

                <div className="rounded-lg bg-green-900/5 p-3 text-sm space-y-1">
                  <div className="flex justify-between">
                    <span className="text-green-900/70">Vous payez (débité de votre solde)</span>
                    <strong className="text-green-950">{formatPrice(amount || 0)}</strong>
                  </div>
                  <div className="flex justify-between text-green-900/60">
                    <span>Frais de service DIARRA ({config.commission_pct} %)</span>
                    <span>{formatPrice(commission)}</span>
                  </div>
                  <div className="flex justify-between text-green-900/60">
                    <span>Budget publicitaire Meta</span>
                    <span>
                      {formatPrice(adBudget)} · ≈ {formatPrice(Math.floor(adBudget / days))}/jour
                    </span>
                  </div>
                </div>

                {problem && <p className="text-sm text-amber-700">{problem}</p>}

                <Button className="w-full" disabled={!!problem || submitting} onClick={() => setConfirmOpen(true)}>
                  {submitting ? 'Lancement…' : `Lancer la pub · ${formatPrice(amount || 0)}`}
                </Button>
                <p className="text-[11px] text-green-900/50 text-center">
                  Meta vérifie chaque pub avant diffusion (quelques heures). Pub refusée = montant remboursé sur votre solde.
                </p>
              </div>

              {/* Aperçu */}
              <div className="space-y-2">
                <p className="text-xs font-medium uppercase tracking-wide text-green-900/50">Aperçu</p>
                <div className="bg-white rounded-xl border border-green-900/10 shadow-card overflow-hidden">
                  <div className="flex items-center gap-2 p-3">
                    <div className="h-8 w-8 rounded-full bg-green-800 text-white flex items-center justify-center text-xs font-bold">D</div>
                    <div>
                      <p className="text-sm font-semibold text-green-950">DIARRA</p>
                      <p className="text-[11px] text-green-900/50">Sponsorisé</p>
                    </div>
                  </div>
                  {message && <p className="px-3 pb-2 text-sm text-green-950 whitespace-pre-line">{message}</p>}
                  {product && <ProductImage product={product} className="h-56" />}
                  <div className="flex items-center justify-between gap-2 bg-green-900/5 p-3">
                    <p className="text-sm font-semibold text-green-950 line-clamp-2">{product?.title}</p>
                    <span className="shrink-0 rounded-md bg-green-900/10 px-3 py-1.5 text-xs font-semibold text-green-950">
                      Acheter
                    </span>
                  </div>
                </div>
              </div>
            </div>
          )
        )}

        {/* Campagnes */}
        <div className="space-y-3">
          <h2 className="text-base font-semibold text-green-950">Mes campagnes</h2>
          {campaigns.length === 0 ? (
            <p className="text-sm text-green-900/50">Aucune pub pour le moment.</p>
          ) : (
            campaigns.map((c) => (
              <div key={c.id} className="bg-white rounded-xl border border-green-900/10 shadow-card p-4 space-y-3">
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="font-medium text-green-950">{c.product_title || 'Produit'}</p>
                    <p className="text-xs text-green-900/60">
                      {formatPrice(c.amount_cfa)} · {c.duration_days} jours · {c.countries.join(', ')}
                      {c.starts_at && ` · du ${new Date(c.starts_at).toLocaleDateString('fr-FR')}`}
                      {c.ends_at && ` au ${new Date(c.ends_at).toLocaleDateString('fr-FR')}`}
                    </p>
                  </div>
                  <Badge className={AD_STATUS_BADGE[c.status]}>{AD_STATUS_LABELS[c.status] || c.status}</Badge>
                </div>
                {c.failure_reason && (c.status === 'rejected' || c.status === 'failed' || c.status === 'stopped') && (
                  <p className="text-xs text-red-600">Raison : {c.failure_reason}</p>
                )}
                {c.refunded && <p className="text-xs text-green-800">{formatPrice(c.amount_cfa)} remboursés sur votre solde.</p>}
                {(c.status === 'active' || c.status === 'completed' || c.status === 'stopped') && (
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 text-center">
                    <Stat label="Affichages" value={c.impressions.toLocaleString('fr-FR')} />
                    <Stat label="Personnes touchées" value={c.reach.toLocaleString('fr-FR')} />
                    <Stat label="Clics vers le produit" value={`${c.clicks.toLocaleString('fr-FR')} (${ctr(c.clicks, c.impressions)})`} />
                    <Stat label="Budget dépensé" value={`${formatPrice(c.spend_cfa)} / ${formatPrice(c.ad_budget_cfa)}`} />
                  </div>
                )}
                {c.stats_updated_at && (
                  <p className="text-[11px] text-green-900/40">
                    Statistiques mises à jour le {new Date(c.stats_updated_at).toLocaleString('fr-FR')}
                  </p>
                )}
              </div>
            ))
          )}
        </div>
      </section>

      <ConfirmDialog
        open={confirmOpen}
        title="Lancer cette pub ?"
        description={
          product
            ? `« ${product.title} » sera diffusé sur Facebook et Instagram pendant ${days} jours. ${formatPrice(amount)} seront prélevés sur votre solde (remboursés si Meta refuse la pub).`
            : undefined
        }
        confirmLabel="Lancer la pub"
        cancelLabel="Annuler"
        onConfirm={submit}
        onCancel={() => setConfirmOpen(false)}
      />
    </main>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg bg-green-900/5 p-2">
      <p className="text-sm font-semibold text-green-950">{value}</p>
      <p className="text-[11px] text-green-900/60">{label}</p>
    </div>
  );
}
