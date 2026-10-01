'use client';

// Admin → Pubs vendeurs : toutes les sponsorisations Facebook/Instagram
// lancées par les vendeurs depuis DIARRA, sur LEURS pages et LEURS comptes
// publicitaires (ils paient Meta directement : aucun argent DIARRA engagé).
// L'admin suit les campagnes, peut en arrêter une, et coupe la création de
// nouvelles pubs avec l'interrupteur ads_enabled.

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { ArrowLeftIcon } from '@/components/icons';
import { formatPrice } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';
import { AD_STATUS_BADGE, AD_STATUS_LABELS, ctr, type AdCampaign } from '@/lib/ads';

export default function AdminAdsPage() {
  const [campaigns, setCampaigns] = useState<AdCampaign[]>([]);
  const [metaConfigured, setMetaConfigured] = useState(false);
  const [enabled, setEnabled] = useState(true);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [msg, setMsg] = useState('');
  const [stopTarget, setStopTarget] = useState<AdCampaign | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  useEffect(() => {
    load();
  }, []);

  async function load() {
    setLoading(true);
    try {
      const r = await api.adminListAds();
      setCampaigns(r.campaigns || []);
      setMetaConfigured(r.meta_configured);
      setEnabled(r.enabled);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  }

  async function toggleEnabled() {
    const next = !enabled;
    setSaving(true);
    setError('');
    setMsg('');
    try {
      await api.updateAdminSettings({ ads_enabled: next ? 'true' : 'false' });
      setEnabled(next);
      setMsg(next ? 'Les vendeurs peuvent créer des pubs.' : 'Création de nouvelles pubs désactivée.');
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSaving(false);
    }
  }

  async function stop(c: AdCampaign) {
    setStopTarget(null);
    setBusyId(c.id);
    setError('');
    setMsg('');
    try {
      await api.adminStopAd(c.id);
      setMsg(`Pub « ${c.product_title} » arrêtée chez Meta.`);
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
    }
  }

  const totals = useMemo(
    () => ({
      count: campaigns.length,
      budget: campaigns
        .filter((c) => c.status !== 'failed' && c.status !== 'rejected')
        .reduce((s, c) => s + c.budget_cfa, 0),
      spend: campaigns.reduce((s, c) => s + c.spend_cfa, 0),
      running: campaigns.filter((c) => c.status === 'active' || c.status === 'in_review').length,
    }),
    [campaigns]
  );

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// administration" title="Pubs vendeurs" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Pubs vendeurs"
        description="Pubs Facebook & Instagram lancées par les vendeurs sur leurs propres comptes publicitaires"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/admin" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Dashboard
          </Button>
        }
      />

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-8 space-y-6">
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

        {!metaConfigured && (
          <div className="p-4 rounded-xl border border-amber-300 bg-amber-50 text-sm text-amber-900">
            <p className="font-semibold">App Meta non configurée</p>
            <p className="mt-1">
              Les vendeurs voient « Bientôt disponible ». Pour activer : renseigner sur le serveur META_APP_ID,
              META_APP_SECRET, META_OAUTH_REDIRECT_URL et META_TOKEN_ENCRYPTION_KEY, puis redéployer.
            </p>
          </div>
        )}

        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <Kpi label="Campagnes" value={String(totals.count)} />
          <Kpi label="En cours" value={String(totals.running)} />
          <Kpi label="Budgets déclarés" value={formatPrice(totals.budget)} />
          <Kpi label="Dépensé (selon Meta)" value={formatPrice(totals.spend)} />
        </div>

        <div className="bg-white rounded-xl border border-green-900/10 shadow-card p-4 flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-semibold text-green-950">Création de pubs par les vendeurs</h2>
            <p className="text-xs text-green-900/60">
              {enabled ? 'Activée' : 'Désactivée'} · les vendeurs paient Meta directement, DIARRA ne prélève rien.
            </p>
          </div>
          <Button size="sm" variant={enabled ? 'outline' : 'default'} disabled={saving} onClick={toggleEnabled}>
            {enabled ? 'Désactiver les nouvelles pubs' : 'Activer les pubs'}
          </Button>
        </div>

        {campaigns.length === 0 ? (
          <EmptyState title="Aucune pub" description="Les pubs lancées par les vendeurs apparaîtront ici." />
        ) : (
          <div className="rounded-xl border border-green-900/10 bg-white shadow-card overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Date</TableHead>
                  <TableHead>Produit / vendeur</TableHead>
                  <TableHead>Budget</TableHead>
                  <TableHead>Diffusion</TableHead>
                  <TableHead>Résultats</TableHead>
                  <TableHead>Statut</TableHead>
                  <TableHead></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {campaigns.map((c) => (
                  <TableRow key={c.id}>
                    <TableCell className="text-sm whitespace-nowrap">{new Date(c.created_at).toLocaleString('fr-FR')}</TableCell>
                    <TableCell className="max-w-[220px]">
                      <span className="block truncate text-green-950">{c.product_title}</span>
                      <span className="block truncate text-xs text-green-900/60">{c.vendor_email}</span>
                    </TableCell>
                    <TableCell className="whitespace-nowrap text-sm">
                      <span className="block font-mono">{formatPrice(c.budget_cfa)}</span>
                      <span className="block text-xs text-green-900/60">compte en {c.currency}</span>
                    </TableCell>
                    <TableCell className="text-sm">
                      {c.duration_days} j · {c.countries.join(', ')}
                    </TableCell>
                    <TableCell className="text-xs whitespace-nowrap text-green-900/70">
                      {c.impressions.toLocaleString('fr-FR')} aff. · {c.clicks.toLocaleString('fr-FR')} clics (
                      {ctr(c.clicks, c.impressions)})
                      <span className="block">
                        dépensé {formatPrice(c.spend_cfa)} / {formatPrice(c.budget_cfa)}
                      </span>
                    </TableCell>
                    <TableCell className="max-w-[220px]">
                      <Badge className={AD_STATUS_BADGE[c.status]}>{AD_STATUS_LABELS[c.status] || c.status}</Badge>
                      {c.failure_reason && <span className="block text-[11px] text-red-600 mt-1">{c.failure_reason}</span>}
                    </TableCell>
                    <TableCell>
                      {(c.status === 'active' || c.status === 'in_review') && (
                        <Button size="sm" variant="outline" disabled={busyId === c.id} onClick={() => setStopTarget(c)}>
                          {busyId === c.id ? '…' : 'Arrêter'}
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </section>

      <ConfirmDialog
        open={!!stopTarget}
        title="Arrêter cette pub ?"
        description={
          stopTarget
            ? `La diffusion de « ${stopTarget.product_title} » s'arrête immédiatement chez Meta (compte publicitaire du vendeur). Le vendeur est prévenu par une notification.`
            : undefined
        }
        confirmLabel="Arrêter la pub"
        cancelLabel="Annuler"
        danger
        onConfirm={() => stopTarget && stop(stopTarget)}
        onCancel={() => setStopTarget(null)}
      />
    </main>
  );
}

function Kpi({ label, value }: { label: string; value: string }) {
  return (
    <div className="bg-white rounded-xl border border-green-900/10 shadow-card p-3">
      <p className="text-lg font-semibold text-green-950">{value}</p>
      <p className="text-xs text-green-900/60">{label}</p>
    </div>
  );
}
