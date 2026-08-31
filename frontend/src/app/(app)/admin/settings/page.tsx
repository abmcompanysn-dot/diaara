'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { ArrowLeftIcon, CheckIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';

// Miroir de backend/internal/payment/pawapay.go XOFOperators (code, pays,
// libellé) — pas d'endpoint public pour la lister dynamiquement.
const OPERATORS: { code: string; country: string; countryLabel: string; label: string }[] = [
  { code: 'ORANGE_SEN', country: 'SEN', countryLabel: 'Sénégal', label: 'Orange Money' },
  { code: 'WAVE_SEN', country: 'SEN', countryLabel: 'Sénégal', label: 'Wave' },
  { code: 'FREE_SEN', country: 'SEN', countryLabel: 'Sénégal', label: 'Free Money' },
  { code: 'MTN_MOMO_CIV', country: 'CIV', countryLabel: "Côte d'Ivoire", label: 'MTN MoMo' },
  { code: 'ORANGE_CIV', country: 'CIV', countryLabel: "Côte d'Ivoire", label: 'Orange Money' },
  { code: 'WAVE_CIV', country: 'CIV', countryLabel: "Côte d'Ivoire", label: 'Wave' },
  { code: 'MTN_MOMO_BEN', country: 'BEN', countryLabel: 'Bénin', label: 'MTN MoMo' },
  { code: 'MOOV_BEN', country: 'BEN', countryLabel: 'Bénin', label: 'Moov Money' },
  { code: 'MOOV_BFA', country: 'BFA', countryLabel: 'Burkina Faso', label: 'Moov Money' },
  { code: 'ORANGE_BFA', country: 'BFA', countryLabel: 'Burkina Faso', label: 'Orange Money' },
  { code: 'MTN_MOMO_CMR', country: 'CMR', countryLabel: 'Cameroun', label: 'MTN MoMo' },
  { code: 'ORANGE_CMR', country: 'CMR', countryLabel: 'Cameroun', label: 'Orange Money' },
  { code: 'AIRTEL_GAB', country: 'GAB', countryLabel: 'Gabon', label: 'Airtel Money' },
  { code: 'AIRTEL_COG', country: 'COG', countryLabel: 'Congo-Brazzaville', label: 'Airtel Money' },
  { code: 'MTN_MOMO_COG', country: 'COG', countryLabel: 'Congo-Brazzaville', label: 'MTN MoMo' },
  { code: 'VODACOM_MPESA_COD', country: 'COD', countryLabel: 'RD Congo', label: 'Vodacom M-Pesa' },
  { code: 'AIRTEL_COD', country: 'COD', countryLabel: 'RD Congo', label: 'Airtel Money' },
  { code: 'ORANGE_COD', country: 'COD', countryLabel: 'RD Congo', label: 'Orange Money' },
  { code: 'MTN_MOMO_GHA', country: 'GHA', countryLabel: 'Ghana', label: 'MTN MoMo' },
  { code: 'AIRTELTIGO_GHA', country: 'GHA', countryLabel: 'Ghana', label: 'AirtelTigo Money' },
  { code: 'VODAFONE_GHA', country: 'GHA', countryLabel: 'Ghana', label: 'Vodafone Cash' },
  { code: 'AIRTEL_NGA', country: 'NGA', countryLabel: 'Nigeria', label: 'Airtel Money' },
  { code: 'MTN_MOMO_NGA', country: 'NGA', countryLabel: 'Nigeria', label: 'MTN MoMo' },
  { code: 'MPESA_KEN', country: 'KEN', countryLabel: 'Kenya', label: 'M-Pesa' },
  { code: 'AIRTEL_RWA', country: 'RWA', countryLabel: 'Rwanda', label: 'Airtel Money' },
  { code: 'MTN_MOMO_RWA', country: 'RWA', countryLabel: 'Rwanda', label: 'MTN MoMo' },
  { code: 'AIRTEL_OAPI_UGA', country: 'UGA', countryLabel: 'Ouganda', label: 'Airtel Money' },
  { code: 'MTN_MOMO_UGA', country: 'UGA', countryLabel: 'Ouganda', label: 'MTN MoMo' },
  { code: 'AIRTEL_TZA', country: 'TZA', countryLabel: 'Tanzanie', label: 'Airtel Money' },
  { code: 'VODACOM_TZA', country: 'TZA', countryLabel: 'Tanzanie', label: 'Vodacom M-Pesa' },
  { code: 'TIGO_TZA', country: 'TZA', countryLabel: 'Tanzanie', label: 'Tigo Pesa' },
  { code: 'HALOTEL_TZA', country: 'TZA', countryLabel: 'Tanzanie', label: 'HaloPesa' },
  { code: 'AIRTEL_OAPI_ZMB', country: 'ZMB', countryLabel: 'Zambie', label: 'Airtel Money' },
  { code: 'MTN_MOMO_ZMB', country: 'ZMB', countryLabel: 'Zambie', label: 'MTN MoMo' },
  { code: 'ZAMTEL_ZMB', country: 'ZMB', countryLabel: 'Zambie', label: 'Zamtel Money' },
  { code: 'AIRTEL_MWI', country: 'MWI', countryLabel: 'Malawi', label: 'Airtel Money' },
  { code: 'TNM_MWI', country: 'MWI', countryLabel: 'Malawi', label: 'TNM Mpamba' },
  { code: 'MOVITEL_MOZ', country: 'MOZ', countryLabel: 'Mozambique', label: 'Movitel' },
  { code: 'VODACOM_MOZ', country: 'MOZ', countryLabel: 'Mozambique', label: 'Vodacom M-Pesa' },
  { code: 'MPESA_LSO', country: 'LSO', countryLabel: 'Lesotho', label: 'M-Pesa' },
  { code: 'ORANGE_SLE', country: 'SLE', countryLabel: 'Sierra Leone', label: 'Orange Money' },
  { code: 'MPESA_ETH', country: 'ETH', countryLabel: 'Éthiopie', label: 'Safaricom M-Pesa' },
];

// Miroir de backend/internal/payment/kpay.go KPayProviderCodes — WAVE_SEN et
// WAVE_CIV volontairement absents (suspendus côté KPay).
const KPAY_SUPPORTED = new Set([
  'MTN_MOMO_BEN', 'MOOV_BEN', 'MTN_MOMO_CMR', 'ORANGE_CMR', 'MTN_MOMO_CIV',
  'ORANGE_CIV', 'VODACOM_MPESA_COD', 'AIRTEL_COD', 'ORANGE_COD', 'AIRTEL_GAB',
  'MPESA_KEN', 'AIRTEL_COG', 'MTN_MOMO_COG', 'AIRTEL_RWA', 'MTN_MOMO_RWA',
  'FREE_SEN', 'ORANGE_SEN', 'ORANGE_SLE', 'AIRTEL_OAPI_UGA', 'MTN_MOMO_UGA',
  'AIRTEL_OAPI_ZMB', 'MTN_MOMO_ZMB', 'ZAMTEL_ZMB',
]);

// Miroir de backend/internal/payment/pawapay.go CountryCurrency (pays du
// checkout Payment Page).
const CHECKOUT_COUNTRIES: { iso3: string; label: string }[] = [
  { iso3: 'SEN', label: 'Sénégal' },
  { iso3: 'CIV', label: "Côte d'Ivoire" },
  { iso3: 'BEN', label: 'Bénin' },
  { iso3: 'BFA', label: 'Burkina Faso' },
  { iso3: 'CMR', label: 'Cameroun' },
  { iso3: 'GAB', label: 'Gabon' },
  { iso3: 'COG', label: 'Congo-Brazzaville' },
  { iso3: 'COD', label: 'RD Congo' },
  { iso3: 'GHA', label: 'Ghana' },
  { iso3: 'NGA', label: 'Nigeria' },
  { iso3: 'KEN', label: 'Kenya' },
  { iso3: 'RWA', label: 'Rwanda' },
  { iso3: 'UGA', label: 'Ouganda' },
  { iso3: 'TZA', label: 'Tanzanie' },
  { iso3: 'ZMB', label: 'Zambie' },
  { iso3: 'MWI', label: 'Malawi' },
  { iso3: 'MOZ', label: 'Mozambique' },
  { iso3: 'LSO', label: 'Lesotho' },
  { iso3: 'SLE', label: 'Sierra Leone' },
  { iso3: 'ETH', label: 'Éthiopie' },
];

type GatewayValue = 'off' | 'pawapay' | 'kpay';
type CheckoutValue = 'pawapay' | 'kpay';

const gatewayOpKey = (code: string) => `gateway_op_${code.toLowerCase()}`;
const checkoutProviderKey = (iso3: string) => `checkout_provider_${iso3.toLowerCase()}`;

function groupByCountry<T extends { country: string; countryLabel: string }>(items: T[]) {
  const groups: { country: string; countryLabel: string; items: T[] }[] = [];
  for (const item of items) {
    let group = groups.find((g) => g.country === item.country);
    if (!group) {
      group = { country: item.country, countryLabel: item.countryLabel, items: [] };
      groups.push(group);
    }
    group.items.push(item);
  }
  return groups;
}

export default function AdminSettingsPage() {
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState('');
  const [commissionRate, setCommissionRate] = useState('15');
  const [gatewayOps, setGatewayOps] = useState<Record<string, GatewayValue>>({});
  const [checkoutProviders, setCheckoutProviders] = useState<Record<string, CheckoutValue>>({});

  useEffect(() => {
    api
      .getAdminSettings()
      .then(({ settings }) => {
        setCommissionRate(settings.commission_rate_pct || '15');
        const g: Record<string, GatewayValue> = {};
        for (const op of OPERATORS) {
          const v = settings[gatewayOpKey(op.code)];
          g[op.code] = v === 'off' || v === 'kpay' ? v : 'pawapay';
        }
        setGatewayOps(g);
        const c: Record<string, CheckoutValue> = {};
        for (const country of CHECKOUT_COUNTRIES) {
          const v = settings[checkoutProviderKey(country.iso3)];
          c[country.iso3] = v === 'kpay' ? 'kpay' : 'pawapay';
        }
        setCheckoutProviders(c);
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
      for (const op of OPERATORS) values[gatewayOpKey(op.code)] = gatewayOps[op.code] || 'pawapay';
      for (const country of CHECKOUT_COUNTRIES)
        values[checkoutProviderKey(country.iso3)] = checkoutProviders[country.iso3] || 'pawapay';
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
              uniquement, pas rétroactif. Palier automatique : à partir de 1 000 000 FCFA,
              la commission passe à 10%, quel que soit le taux ci-dessous.
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
            <CardTitle>Versements vendeur — par opérateur</CardTitle>
            <CardDescription>
              Pour chaque opérateur mobile money, choisis le prestataire qui traite les
              versements vendeur, ou désactive-le. « Désactivé » bloque les nouvelles demandes
              de versement (les versements déjà enregistrés ne sont pas affectés). Un opérateur
              non pris en charge par KPay ne propose pas cette option.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            {groupByCountry(OPERATORS).map((group) => (
              <div key={group.country} className="space-y-2">
                <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                  {group.countryLabel}
                </h3>
                <div className="space-y-2">
                  {group.items.map((op) => {
                    const kpaySupported = KPAY_SUPPORTED.has(op.code);
                    const value = gatewayOps[op.code] || 'pawapay';
                    return (
                      <div
                        key={op.code}
                        className="flex items-center justify-between gap-3 p-3 rounded-lg border border-border"
                      >
                        <div>
                          <span className="text-sm font-medium">{op.label}</span>
                          {!kpaySupported && (
                            <span className="ml-2 text-xs text-muted-foreground">
                              (non pris en charge par KPay)
                            </span>
                          )}
                        </div>
                        <div className="flex gap-1 shrink-0">
                          {(['off', 'pawapay', 'kpay'] as GatewayValue[]).map((opt) => {
                            if (opt === 'kpay' && !kpaySupported) return null;
                            const optLabel = opt === 'off' ? 'Désactivé' : opt === 'pawapay' ? 'PawaPay' : 'KPay';
                            return (
                              <Button
                                key={opt}
                                type="button"
                                size="sm"
                                variant={value === opt ? 'default' : 'outline'}
                                onClick={() =>
                                  setGatewayOps((prev) => ({ ...prev, [op.code]: opt }))
                                }
                              >
                                {optLabel}
                              </Button>
                            );
                          })}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Paiement à l&apos;achat — par pays</CardTitle>
            <CardDescription>
              Prestataire mobile money utilisé au moment du paiement, par pays de
              l&apos;acheteur (le paiement par carte bancaire ou PayPal passe toujours par
              KPay, quel que soit ce réglage).
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            {CHECKOUT_COUNTRIES.map((country) => {
              const value = checkoutProviders[country.iso3] || 'pawapay';
              return (
                <div
                  key={country.iso3}
                  className="flex items-center justify-between gap-3 p-3 rounded-lg border border-border"
                >
                  <span className="text-sm font-medium">{country.label}</span>
                  <div className="flex gap-1 shrink-0">
                    {(['pawapay', 'kpay'] as CheckoutValue[]).map((opt) => (
                      <Button
                        key={opt}
                        type="button"
                        size="sm"
                        variant={value === opt ? 'default' : 'outline'}
                        onClick={() =>
                          setCheckoutProviders((prev) => ({ ...prev, [country.iso3]: opt }))
                        }
                      >
                        {opt === 'pawapay' ? 'PawaPay' : 'KPay'}
                      </Button>
                    ))}
                  </div>
                </div>
              );
            })}
          </CardContent>
        </Card>

        <Button onClick={handleSave} disabled={saving} className="w-full font-semibold">
          {saving ? 'Enregistrement...' : 'Enregistrer les modifications'}
        </Button>
      </section>
    </main>
  );
}
