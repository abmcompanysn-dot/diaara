'use client';

// Espace vendeur → Mes pubs : sponsoriser un produit sur Facebook et
// Instagram en quelques clics. Le vendeur connecte SON compte Facebook,
// choisit SA page et SON compte publicitaire, et c'est Meta qui lui facture
// le budget (moyen de paiement de son compte pub) : DIARRA ne prélève rien
// (voir backend handler/ad_handler.go et handler/ad_meta_connect.go).
//
// 1. Connecter mon compte Facebook (Facebook Login, retour ?meta=connected
//    ou ?meta=error&reason=...) ; 2. choisir la page et le compte pub ;
// 3. créer la pub. Puis suivi des campagnes (stats, arrêt).

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
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
  AD_ACCOUNT_STATUS_LABELS,
  AD_STATUS_BADGE,
  AD_STATUS_LABELS,
  META_CONNECT_ERRORS,
  ctr,
  formatInAccountCurrency,
  minBudgetFor,
  roundUpThousand,
  type AdCampaign,
  type MetaAdAccount,
  type MetaPage,
  type MetaStatus,
} from '@/lib/ads';

interface AdsConfig {
  enabled: boolean;
  min_daily_cfa: number;
  max_duration_days: number;
  max_budget_cfa: number;
  supported_currencies: string[];
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

// Détail renvoyé par l'API en plus du code d'erreur (raison Meta, minimum,
// devise non prise en charge).
function errorText(err: unknown): string {
  let text = friendlyError(err);
  if (err instanceof ApiError) {
    const { details, min_budget_cfa, currency } = err.body || {};
    if (typeof details === 'string' && details) text += ` (${details})`;
    if (typeof min_budget_cfa === 'number') text += ` Minimum : ${formatPrice(min_budget_cfa)}.`;
    if (typeof currency === 'string' && currency) text += ` Devise actuelle : ${currency}.`;
  }
  return text;
}

export default function VendorAds() {
  const searchParams = useSearchParams();
  const router = useRouter();
  const [config, setConfig] = useState<AdsConfig | null>(null);
  const [meta, setMeta] = useState<MetaStatus | null>(null);
  const [pages, setPages] = useState<MetaPage[] | null>(null);
  const [accounts, setAccounts] = useState<MetaAdAccount[] | null>(null);
  const [products, setProducts] = useState<VendorProduct[]>([]);
  const [campaigns, setCampaigns] = useState<AdCampaign[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [msg, setMsg] = useState('');

  // Étape 1 / 2 : connexion et choix page + compte pub.
  const [connecting, setConnecting] = useState(false);
  const [disconnectOpen, setDisconnectOpen] = useState(false);
  const [editingSelection, setEditingSelection] = useState(false);
  const [assetsLoading, setAssetsLoading] = useState(false);
  const [pageId, setPageId] = useState('');
  const [accountId, setAccountId] = useState('');
  const [savingSelection, setSavingSelection] = useState(false);

  // Étape 3 : formulaire de pub.
  const [productId, setProductId] = useState('');
  const [days, setDays] = useState(7);
  const [budget, setBudget] = useState(0);
  const [countries, setCountries] = useState<string[]>(DEFAULT_COUNTRIES);
  const [message, setMessage] = useState('');
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  // Suivi : arrêt d'une pub.
  const [stopTarget, setStopTarget] = useState<AdCampaign | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  useEffect(() => {
    // Retour de Facebook Login.
    const status = searchParams.get('meta');
    if (status === 'connected') {
      setMsg('Compte Facebook connecté. Choisissez maintenant votre page et votre compte publicitaire.');
    } else if (status === 'error') {
      setError(META_CONNECT_ERRORS[searchParams.get('reason') || ''] || META_CONNECT_ERRORS.internal);
    }
    if (status) {
      const product = searchParams.get('product');
      router.replace(product ? `/vendor/ads?product=${encodeURIComponent(product)}` : '/vendor/ads');
    }
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function load() {
    setLoading(true);
    try {
      const [cfg, status, prods, ads] = await Promise.all([
        api.getAdsConfig(),
        api.getMetaStatus(),
        api.getVendorProducts(),
        api.getVendorAds(),
      ]);
      setConfig(cfg);
      setMeta(status);
      const approved = (prods.products || []).filter(
        (p: VendorProduct) => p.moderation_status === 'approved' && !p.deletion_requested
      );
      setProducts(approved);
      setCampaigns(ads.campaigns || []);
      if (!productId) {
        const wanted = searchParams.get('product');
        const initial = approved.find((p: VendorProduct) => p.id === wanted) || approved[0];
        if (initial) {
          setProductId(initial.id);
          setMessage(initial.title);
        }
        setBudget(roundUpThousand(minBudgetFor(7, cfg.min_daily_cfa) * 2));
      }
      if (status.connected && !status.connection?.needs_reconnect && !status.ready) {
        loadAssets(status);
      }
    } catch (err: unknown) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  }

  async function loadAssets(status: MetaStatus | null = meta) {
    setAssetsLoading(true);
    try {
      const r = await api.getMetaAssets();
      setPages(r.pages || []);
      setAccounts(r.ad_accounts || []);
      const conn = status?.connection;
      const firstUsable = (r.ad_accounts || []).find((a) => a.usable);
      setPageId(conn?.page_id || (r.pages || [])[0]?.id || '');
      setAccountId(conn?.ad_account_id || firstUsable?.id || '');
    } catch (err: unknown) {
      setError(errorText(err));
      // Jeton expiré détecté à l'instant : rafraîchit l'état (bouton Reconnecter).
      if (err instanceof ApiError && err.message === 'meta_reconnect_required') {
        setMeta(await api.getMetaStatus().catch(() => status));
      }
    } finally {
      setAssetsLoading(false);
    }
  }

  async function connect() {
    setConnecting(true);
    setError('');
    try {
      const { url } = await api.getMetaConnectUrl();
      window.location.href = url;
    } catch (err: unknown) {
      setError(friendlyError(err));
      setConnecting(false);
    }
  }

  async function disconnect() {
    setDisconnectOpen(false);
    setError('');
    setMsg('');
    try {
      await api.disconnectMeta();
      setPages(null);
      setAccounts(null);
      setEditingSelection(false);
      setMsg('Compte Facebook déconnecté de DIARRA.');
      setMeta(await api.getMetaStatus());
    } catch (err: unknown) {
      setError(friendlyError(err));
    }
  }

  async function saveSelection() {
    setSavingSelection(true);
    setError('');
    setMsg('');
    try {
      const status = await api.setMetaSelection({ page_id: pageId, ad_account_id: accountId });
      setMeta(status);
      setEditingSelection(false);
      setMsg('Page et compte publicitaire enregistrés. Vous pouvez créer votre pub.');
    } catch (err: unknown) {
      setError(errorText(err));
    } finally {
      setSavingSelection(false);
    }
  }

  function startEditSelection() {
    setEditingSelection(true);
    loadAssets();
  }

  const conn = meta?.connection || null;
  const needsReconnect = !!conn?.needs_reconnect;
  const showSelection = !!conn && !needsReconnect && (!meta?.ready || editingSelection);
  const ready = !!meta?.ready && !editingSelection;
  const currency = conn?.currency || '';
  const pageName = conn?.page_name || 'Votre page';

  const product = products.find((p) => p.id === productId);
  const minBudget = config ? minBudgetFor(days, config.min_daily_cfa) : 0;
  const presets = useMemo(() => {
    if (!config) return [];
    const base = roundUpThousand(minBudget);
    return Array.from(new Set([base, roundUpThousand(base * 2), roundUpThousand(base * 4)]));
  }, [config, minBudget]);
  const budgetInAccount = formatInAccountCurrency(budget || 0, currency);

  const problem = !config
    ? ''
    : !product
      ? 'Choisissez un produit.'
      : countries.length === 0
        ? 'Choisissez au moins un pays.'
        : budget < minBudget
          ? `Minimum ${formatPrice(minBudget)} pour ${days} jours.`
          : budget > config.max_budget_cfa
            ? `Maximum ${formatPrice(config.max_budget_cfa)}.`
            : '';

  function pickDays(d: number) {
    setDays(d);
    if (config) {
      const min = minBudgetFor(d, config.min_daily_cfa);
      if (budget < min) setBudget(roundUpThousand(min));
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
        budget_cfa: budget,
        duration_days: days,
        countries,
        message: message.trim(),
      });
      setMsg(
        'Votre pub est envoyée à Meta. Elle passe en vérification (en général quelques heures), puis la diffusion démarre automatiquement.'
      );
      await load();
    } catch (err: unknown) {
      setError(errorText(err));
      if (err instanceof ApiError && err.message === 'meta_reconnect_required') {
        setMeta(await api.getMetaStatus().catch(() => meta));
      }
    } finally {
      setSubmitting(false);
    }
  }

  async function stop(c: AdCampaign) {
    setStopTarget(null);
    setBusyId(c.id);
    setError('');
    setMsg('');
    try {
      await api.stopVendorAd(c.id);
      setMsg(`Pub « ${c.product_title || 'produit'} » arrêtée.`);
      await load();
    } catch (err: unknown) {
      setError(errorText(err));
    } finally {
      setBusyId(null);
    }
  }

  if (loading)
    return (
      <main>
        <PageHeader back="/vendor/products" eyebrow="// espace vendeur" title="Mes pubs" />
        <PageLoader />
      </main>
    );

  const unavailable = config && !config.enabled;

  return (
    <main>
      <PageHeader
        back="/vendor/products"
        eyebrow="// espace vendeur"
        title="Mes pubs"
        description="Sponsorisez vos produits sur Facebook et Instagram, depuis votre propre page"
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

        {unavailable ? (
          <div className="bg-white rounded-xl border border-green-900/10 shadow-card p-6 text-center space-y-2">
            <p className="text-lg font-semibold text-green-950">Bientôt disponible</p>
            <p className="text-sm text-green-900/60 max-w-md mx-auto">
              Vous pourrez bientôt faire la publicité de vos produits sur Facebook et Instagram directement depuis DIARRA,
              avec votre propre page et votre compte publicitaire.
            </p>
          </div>
        ) : (
          <>
            {/* Étape 1 : compte Facebook */}
            <div className="bg-white rounded-xl border border-green-900/10 shadow-card p-5 space-y-3">
              <StepTitle n={1} title="Votre compte Facebook" done={!!conn && !needsReconnect} />
              {!conn ? (
                <>
                  <p className="text-sm text-green-900/70">
                    Vos pubs sont publiées sur <strong>votre page Facebook</strong> avec <strong>votre compte publicitaire</strong>.
                    Le budget est facturé par Meta sur le moyen de paiement de votre compte publicitaire : DIARRA ne prélève rien.
                  </p>
                  <Button onClick={connect} disabled={connecting}>
                    {connecting ? 'Redirection vers Facebook…' : 'Connecter mon compte Facebook'}
                  </Button>
                </>
              ) : needsReconnect ? (
                <>
                  <p className="text-sm text-amber-700">
                    La connexion à votre compte Facebook a expiré ou a été retirée. Reconnectez-le pour lancer et suivre vos pubs.
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <Button onClick={connect} disabled={connecting}>
                      {connecting ? 'Redirection vers Facebook…' : 'Reconnecter'}
                    </Button>
                    <Button variant="outline" onClick={() => setDisconnectOpen(true)}>
                      Déconnecter
                    </Button>
                  </div>
                </>
              ) : (
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <p className="text-sm text-green-900/70">
                    Connecté en tant que <strong className="text-green-950">{conn.fb_user_name || 'compte Facebook'}</strong>
                  </p>
                  <Button variant="outline" size="sm" onClick={() => setDisconnectOpen(true)}>
                    Déconnecter
                  </Button>
                </div>
              )}
            </div>

            {/* Étape 2 : page + compte publicitaire */}
            {conn && !needsReconnect && (
              <div className="bg-white rounded-xl border border-green-900/10 shadow-card p-5 space-y-3">
                <StepTitle n={2} title="Page et compte publicitaire" done={ready} />
                {showSelection ? (
                  assetsLoading ? (
                    <p className="text-sm text-green-900/60">Chargement de vos pages et comptes…</p>
                  ) : pages === null || accounts === null ? (
                    <Button variant="outline" onClick={() => loadAssets()}>
                      Charger mes pages et comptes publicitaires
                    </Button>
                  ) : (
                    <div className="space-y-4">
                      <div className="space-y-1.5">
                        <Label htmlFor="meta-page">Page Facebook</Label>
                        {pages.length === 0 ? (
                          <p className="text-sm text-amber-700">
                            Aucune page trouvée. Créez une page Facebook, ou reconnectez-vous en cochant vos pages dans la fenêtre
                            Facebook.
                          </p>
                        ) : (
                          <select
                            id="meta-page"
                            value={pageId}
                            onChange={(e) => setPageId(e.target.value)}
                            className="h-10 w-full rounded-md border border-input bg-white px-3 text-sm"
                          >
                            {pages.map((p) => (
                              <option key={p.id} value={p.id}>
                                {p.name}
                              </option>
                            ))}
                          </select>
                        )}
                      </div>
                      <div className="space-y-1.5">
                        <Label htmlFor="meta-account">Compte publicitaire</Label>
                        {accounts.length === 0 ? (
                          <p className="text-sm text-amber-700">
                            Aucun compte publicitaire trouvé. Créez-en un dans le Gestionnaire de publicités Meta (avec un moyen
                            de paiement), puis reconnectez-vous.
                          </p>
                        ) : (
                          <select
                            id="meta-account"
                            value={accountId}
                            onChange={(e) => setAccountId(e.target.value)}
                            className="h-10 w-full rounded-md border border-input bg-white px-3 text-sm"
                          >
                            {accounts.map((a) => (
                              <option key={a.id} value={a.id} disabled={!a.usable}>
                                {a.name} · {a.currency}
                                {!a.active
                                  ? ` — ${AD_ACCOUNT_STATUS_LABELS[a.account_status] || 'inactif'}`
                                  : !a.currency_supported
                                    ? ' — devise non prise en charge'
                                    : ''}
                              </option>
                            ))}
                          </select>
                        )}
                        <p className="text-[11px] text-green-900/50">
                          Devises prises en charge : FCFA (XOF/XAF), euro (EUR), dollar (USD). Seuls les comptes actifs sont
                          utilisables.
                        </p>
                      </div>
                      <div className="flex flex-wrap gap-2">
                        <Button
                          onClick={saveSelection}
                          disabled={savingSelection || !pageId || !accountId || !accounts.find((a) => a.id === accountId)?.usable}
                        >
                          {savingSelection ? 'Enregistrement…' : 'Enregistrer'}
                        </Button>
                        {editingSelection && (
                          <Button variant="outline" onClick={() => setEditingSelection(false)}>
                            Annuler
                          </Button>
                        )}
                        <Button variant="ghost" onClick={connect} disabled={connecting}>
                          Reconnecter Facebook
                        </Button>
                      </div>
                    </div>
                  )
                ) : (
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <p className="text-sm text-green-900/70">
                      Page <strong className="text-green-950">{conn.page_name}</strong> · compte{' '}
                      <strong className="text-green-950">{conn.ad_account_name}</strong> ({conn.currency})
                    </p>
                    <Button variant="outline" size="sm" onClick={startEditSelection}>
                      Modifier
                    </Button>
                  </div>
                )}
              </div>
            )}

            {/* Étape 3 : la pub */}
            {ready &&
              config &&
              (products.length === 0 ? (
                <EmptyState
                  title="Aucun produit à sponsoriser"
                  description="Seuls les produits validés par DIARRA peuvent être sponsorisés."
                  action={<Button render={<Link href="/vendor/products" />}>Voir mes produits</Button>}
                />
              ) : (
                <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
                  {/* Formulaire */}
                  <div className="bg-white rounded-xl border border-green-900/10 shadow-card p-5 space-y-5">
                    <StepTitle n={3} title="Nouvelle pub Facebook & Instagram" done={false} />

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
                              days === d
                                ? 'border-green-800 bg-green-800 text-white'
                                : 'border-green-900/20 text-green-950 hover:bg-green-900/5'
                            }`}
                          >
                            {d} jours
                          </button>
                        ))}
                      </div>
                    </div>

                    <div className="space-y-1.5">
                      <Label htmlFor="ad-budget">Budget total</Label>
                      <div className="flex flex-wrap gap-2">
                        {presets.map((p) => (
                          <button
                            key={p}
                            type="button"
                            onClick={() => setBudget(p)}
                            className={`rounded-full border px-4 py-1.5 text-sm ${
                              budget === p
                                ? 'border-green-800 bg-green-800 text-white'
                                : 'border-green-900/20 text-green-950 hover:bg-green-900/5'
                            }`}
                          >
                            {formatPrice(p)}
                          </button>
                        ))}
                      </div>
                      <Input
                        id="ad-budget"
                        type="number"
                        min={0}
                        step={500}
                        value={budget || ''}
                        onChange={(e) => setBudget(parseInt(e.target.value, 10) || 0)}
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
                        <span className="text-green-900/70">Budget total</span>
                        <strong className="text-green-950">
                          {formatPrice(budget || 0)}
                          {budgetInAccount && <span className="font-normal text-green-900/60"> ≈ {budgetInAccount}</span>}
                        </strong>
                      </div>
                      <div className="flex justify-between text-green-900/60">
                        <span>Par jour (environ)</span>
                        <span>{formatPrice(Math.floor((budget || 0) / days))}</span>
                      </div>
                      <p className="text-xs text-green-900/60 pt-1">
                        Le budget est facturé par Meta sur le moyen de paiement de votre compte publicitaire
                        {conn?.ad_account_name ? ` (« ${conn.ad_account_name} »)` : ''}. DIARRA ne prélève rien.
                      </p>
                    </div>

                    {problem && <p className="text-sm text-amber-700">{problem}</p>}

                    <Button className="w-full" disabled={!!problem || submitting} onClick={() => setConfirmOpen(true)}>
                      {submitting ? 'Lancement…' : 'Lancer la pub'}
                    </Button>
                    <p className="text-[11px] text-green-900/50 text-center">
                      Meta vérifie chaque pub avant diffusion (quelques heures). Vous pouvez l’arrêter à tout moment.
                    </p>
                  </div>

                  {/* Aperçu */}
                  <div className="space-y-2">
                    <p className="text-xs font-medium uppercase tracking-wide text-green-900/50">Aperçu</p>
                    <div className="bg-white rounded-xl border border-green-900/10 shadow-card overflow-hidden">
                      <div className="flex items-center gap-2 p-3">
                        <div className="h-8 w-8 rounded-full bg-green-800 text-white flex items-center justify-center text-xs font-bold">
                          {pageName.trim().charAt(0).toUpperCase() || 'P'}
                        </div>
                        <div className="min-w-0">
                          <p className="text-sm font-semibold text-green-950 truncate">{pageName}</p>
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
              ))}
          </>
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
                      {formatPrice(c.budget_cfa)} · {c.duration_days} jours · {c.countries.join(', ')}
                      {c.starts_at && ` · du ${new Date(c.starts_at).toLocaleDateString('fr-FR')}`}
                      {c.ends_at && ` au ${new Date(c.ends_at).toLocaleDateString('fr-FR')}`}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge className={AD_STATUS_BADGE[c.status]}>{AD_STATUS_LABELS[c.status] || c.status}</Badge>
                    {(c.status === 'active' || c.status === 'in_review') && (
                      <Button size="sm" variant="outline" disabled={busyId === c.id} onClick={() => setStopTarget(c)}>
                        {busyId === c.id ? '…' : 'Arrêter'}
                      </Button>
                    )}
                  </div>
                </div>
                {c.failure_reason && (c.status === 'rejected' || c.status === 'failed' || c.status === 'stopped') && (
                  <p className="text-xs text-red-600">Raison : {c.failure_reason}</p>
                )}
                {(c.status === 'active' || c.status === 'completed' || c.status === 'stopped') && (
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 text-center">
                    <Stat label="Affichages" value={c.impressions.toLocaleString('fr-FR')} />
                    <Stat label="Personnes touchées" value={c.reach.toLocaleString('fr-FR')} />
                    <Stat
                      label="Clics vers le produit"
                      value={`${c.clicks.toLocaleString('fr-FR')} (${ctr(c.clicks, c.impressions)})`}
                    />
                    <Stat label="Dépensé (facturé par Meta)" value={`${formatPrice(c.spend_cfa)} / ${formatPrice(c.budget_cfa)}`} />
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
            ? `« ${product.title} » sera diffusé sur Facebook et Instagram depuis la page « ${pageName} » pendant ${days} jours, avec un budget total de ${formatPrice(budget)}${budgetInAccount ? ` (≈ ${budgetInAccount})` : ''}, facturé par Meta sur votre compte publicitaire.`
            : undefined
        }
        confirmLabel="Lancer la pub"
        cancelLabel="Annuler"
        onConfirm={submit}
        onCancel={() => setConfirmOpen(false)}
      />

      <ConfirmDialog
        open={!!stopTarget}
        title="Arrêter cette pub ?"
        description={
          stopTarget
            ? `La diffusion de « ${stopTarget.product_title || 'ce produit'} » s'arrête immédiatement chez Meta. Le budget déjà dépensé reste facturé par Meta.`
            : undefined
        }
        confirmLabel="Arrêter la pub"
        cancelLabel="Annuler"
        danger
        onConfirm={() => stopTarget && stop(stopTarget)}
        onCancel={() => setStopTarget(null)}
      />

      <ConfirmDialog
        open={disconnectOpen}
        title="Déconnecter votre compte Facebook ?"
        description="DIARRA n'aura plus accès à vos pages ni à vos comptes publicitaires. Les pubs déjà en cours continuent chez Meta : arrêtez-les avant si besoin, ou depuis votre Gestionnaire de publicités."
        confirmLabel="Déconnecter"
        cancelLabel="Annuler"
        danger
        onConfirm={disconnect}
        onCancel={() => setDisconnectOpen(false)}
      />
    </main>
  );
}

function StepTitle({ n, title, done }: { n: number; title: string; done: boolean }) {
  return (
    <div className="flex items-center gap-2">
      <span
        className={`flex h-6 w-6 items-center justify-center rounded-full text-xs font-bold ${
          done ? 'bg-green-800 text-white' : 'bg-green-900/10 text-green-950'
        }`}
      >
        {done ? '✓' : n}
      </span>
      <h2 className="text-base font-semibold text-green-950">{title}</h2>
    </div>
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
