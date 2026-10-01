'use client';

// Admin → Pubs vendeurs : toutes les sponsorisations Facebook/Instagram
// (argent réel engagé sur le compte publicitaire Meta DIARRA), arrêt d'une
// campagne, et réglages (activation, commission, budget journalier minimum).

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
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
  const [commission, setCommission] = useState('');
  const [minDaily, setMinDaily] = useState('');
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
      setCommission(String(r.commission_pct));
      setMinDaily(String(r.min_daily_cfa));
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  }

  async function saveSettings(nextEnabled = enabled) {
    setSaving(true);
    setError('');
    setMsg('');
    try {
      await api.updateAdminSettings({
        ads_enabled: nextEnabled ? 'true' : 'false',
        ads_commission_pct: commission.trim(),
        ads_min_daily_cfa: minDaily.trim(),
      });
      setEnabled(nextEnabled);
      setMsg('Réglages des pubs enregistrés.');
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

  const totals = useMemo(() => {
    const paid = campaigns.filter((c) => !c.refunded);
    return {
      revenue: paid.reduce((s, c) => s + c.amount_cfa, 0),
      commission: paid.reduce((s, c) => s + c.commission_cfa, 0),
      spend: campaigns.reduce((s, c) => s + c.spend_cfa, 0),
      running: campaigns.filter((c) => c.status === 'active' || c.status === 'in_review').length,
    };
  }, [campaigns]);

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
        description="Sponsorisations Facebook & Instagram payées par les vendeurs"
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
            <p className="font-semibold">Compte publicitaire Meta non configuré</p>
            <p className="mt-1">
              Les vendeurs voient « Bientôt disponible ». Pour activer : renseigner sur le serveur
              META_ADS_ACCESS_TOKEN, META_AD_ACCOUNT_ID, META_PAGE_ID (et idéalement META_APP_SECRET,
              META_AD_ACCOUNT_CURRENCY, META_INSTAGRAM_USER_ID), puis redéployer.
            </p>
          </div>
        )}

        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <Kpi label="Payé par les vendeurs" value={formatPrice(totals.revenue)} />
          <Kpi label="Commission DIARRA" value={formatPrice(totals.commission)} />
          <Kpi label="Dépensé chez Meta" value={formatPrice(totals.spend)} />
          <Kpi label="En cours" value={String(totals.running)} />
        </div>

        <div className="bg-white rounded-xl border border-green-900/10 shadow-card p-4 space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <h2 className="text-base font-semibold text-green-950">Réglages</h2>
            <Button size="sm" variant={enabled ? 'outline' : 'default'} disabled={saving} onClick={() => saveSettings(!enabled)}>
              {enabled ? 'Désactiver les nouvelles pubs' : 'Activer les pubs'}
            </Button>
          </div>
          <div className="grid gap-3 sm:grid-cols-3 items-end">
            <div className="space-y-1.5">
              <Label htmlFor="ads-commission">Commission DIARRA (%)</Label>
              <Input id="ads-commission" type="number" min={0} max={90} value={commission} onChange={(e) => setCommission(e.target.value)} className="bg-white" />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="ads-min-daily">Budget pub minimum / jour (FCFA)</Label>
              <Input id="ads-min-daily" type="number" min={100} step={100} value={minDaily} onChange={(e) => setMinDaily(e.target.value)} className="bg-white" />
            </div>
            <Button disabled={saving} onClick={() => saveSettings()}>
              {saving ? 'Enregistrement…' : 'Enregistrer'}
            </Button>
          </div>
          <p className="text-xs text-green-900/50">
            La commission couvre le change FCFA → devise du compte Meta, les frais de carte et votre marge.
          </p>
        </div>

        {campaigns.length === 0 ? (
          <EmptyState title="Aucune pub" description="Les sponsorisations lancées par les vendeurs apparaîtront ici." />
        ) : (
          <div className="rounded-xl border border-green-900/10 bg-white shadow-card overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Date</TableHead>
                  <TableHead>Produit / vendeur</TableHead>
                  <TableHead>Montant</TableHead>
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
                      <span className="block font-mono">{formatPrice(c.amount_cfa)}</span>
                      <span className="block text-xs text-green-900/60">dont {formatPrice(c.commission_cfa)} commission</span>
                    </TableCell>
                    <TableCell className="text-sm">
                      {c.duration_days} j · {c.countries.join(', ')}
                    </TableCell>
                    <TableCell className="text-xs whitespace-nowrap text-green-900/70">
                      {c.impressions.toLocaleString('fr-FR')} aff. · {c.clicks.toLocaleString('fr-FR')} clics ({ctr(c.clicks, c.impressions)})
                      <span className="block">dépensé {formatPrice(c.spend_cfa)} / {formatPrice(c.ad_budget_cfa)}</span>
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
            ? `La diffusion de « ${stopTarget.product_title} » s'arrête immédiatement chez Meta. Le budget déjà dépensé n'est pas récupérable ; aucun remboursement automatique au vendeur.`
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
