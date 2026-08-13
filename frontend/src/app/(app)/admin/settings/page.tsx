'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { ArrowLeftIcon, CheckIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';

const GATEWAYS: { key: string; label: string }[] = [
  { key: 'gateway_orange_money', label: 'Orange Money' },
  { key: 'gateway_wave', label: 'Wave' },
  { key: 'gateway_mtn_momo', label: 'MTN MoMo' },
  { key: 'gateway_free_money', label: 'Free Money' },
  { key: 'gateway_moov_money', label: 'Moov Money' },
];

export default function AdminSettingsPage() {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState('');
  const [commissionRate, setCommissionRate] = useState('15');
  const [gateways, setGateways] = useState<Record<string, boolean>>({});

  useEffect(() => {
    api
      .getAdminSettings()
      .then(({ settings }) => {
        setCommissionRate(settings.commission_rate_pct || '15');
        const g: Record<string, boolean> = {};
        for (const gw of GATEWAYS) g[gw.key] = settings[gw.key] !== 'false';
        setGateways(g);
      })
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  }, []);

  const handleSave = async () => {
    setError('');
    setSaved(false);
    const rate = parseFloat(commissionRate);
    if (isNaN(rate) || rate < 0 || rate > 100) {
      setError('Le taux de commission doit être entre 0 et 100.');
      return;
    }
    setSaving(true);
    try {
      const values: Record<string, string> = { commission_rate_pct: String(rate) };
      for (const gw of GATEWAYS) values[gw.key] = String(gateways[gw.key] !== false);
      await api.updateAdminSettings(values);
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSaving(false);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// administration" title="Paramètres" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Paramètres de la plateforme"
        description="Commission, passerelles de paiement"
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
        {saved && (
          <div className="p-3 bg-green-50 text-green-700 rounded text-sm flex items-center gap-2" role="status">
            <CheckIcon size={16} /> Paramètres enregistrés.
          </div>
        )}

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Commission plateforme</CardTitle>
            <CardDescription>
              Pourcentage prélevé sur chaque vente. S&apos;applique aux nouvelles ventes
              uniquement, pas rétroactif.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 max-w-xs">
              <Label htmlFor="rate">Taux (%)</Label>
              <Input
                id="rate"
                type="number"
                min={0}
                max={100}
                step="0.1"
                value={commissionRate}
                onChange={(e) => setCommissionRate(e.target.value)}
              />
            </div>
          </CardContent>
        </Card>

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Passerelles de paiement</CardTitle>
            <CardDescription>
              Désactiver un opérateur bloque les nouvelles demandes de versement vers ce
              moyen mobile money (les versements déjà enregistrés ne sont pas affectés).
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {GATEWAYS.map((gw) => (
              <label
                key={gw.key}
                className="flex items-center justify-between p-3 rounded-lg border border-border cursor-pointer"
              >
                <span className="text-sm font-medium">{gw.label}</span>
                <Checkbox
                  checked={gateways[gw.key] !== false}
                  onCheckedChange={(checked) =>
                    setGateways((prev) => ({ ...prev, [gw.key]: checked === true }))
                  }
                />
              </label>
            ))}
          </CardContent>
        </Card>

        <Button onClick={handleSave} disabled={saving} className="w-full font-semibold">
          {saving ? 'Enregistrement...' : 'Enregistrer les modifications'}
        </Button>
      </section>
    </main>
  );
}
