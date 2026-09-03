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
// Miroir de backend/internal/model/settings.go WhatsAppCommunitySettingKey.
const whatsappKey = (iso3: string) => `whatsapp_community_url_${iso3.toLowerCase()}`;
const WHATSAPP_GENERAL_KEY = 'whatsapp_community_url';

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
  // Liens communauté WhatsApp : clé "general" + une clé par ISO3.
  const [whatsappLinks, setWhatsappLinks] = useState<Record<string, string>>({});

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
        const wa: Record<string, string> = { general: settings[WHATSAPP_GENERAL_KEY] || '' };
        for (const country of CHECKOUT_COUNTRIES) {
          wa[country.iso3] = settings[whatsappKey(country.iso3)] || '';
        }
        setWhatsappLinks(wa);
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
      // KPay pas encore activé sur ce flux (voir la carte "Paiement à l'achat —
      // par pays" ci-dessous, non modifiable) : on envoie toujours "pawapay",
      // même si une valeur "kpay" existait en base — l'enregistrement la corrige.
      for (const country of CHECKOUT_COUNTRIES) values[checkoutProviderKey(country.iso3)] = 'pawapay';
      values[WHATSAPP_GENERAL_KEY] = (whatsappLinks.general || '').trim();
      for (const country of CHECKOUT_COUNTRIES)
        values[whatsappKey(country.iso3)] = (whatsappLinks[country.iso3] || '').trim();
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
        description="Commission, passerelles de paiement, communauté WhatsApp"
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
              Pour chaque opérateur mobile money, active ou désactive les versements vendeur.
              « Désactivé » bloque les nouvelles demandes de versement (les versements déjà
              enregistrés ne sont pas affectés). Tous les versements automatiques passent par
              PawaPay ; KPay est suspendu.
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
                    const value = gatewayOps[op.code] || 'pawapay';
                    return (
                      <div
                        key={op.code}
                        className="flex items-center justify-between gap-3 p-3 rounded-lg border border-border"
                      >
                        <div>
                          <span className="text-sm font-medium">{op.label}</span>
                          {value === 'kpay' && (
                            <span className="ml-2 text-xs text-amber-700">
                              (réglage KPay ignoré — KPay suspendu, remis à PawaPay au prochain enregistrement)
                            </span>
                          )}
                        </div>
                        <div className="flex gap-1 shrink-0">
                          {/* KPay suspendu (2026-09-03) : seuls Désactivé / PawaPay. */}
                          {(['off', 'pawapay'] as GatewayValue[]).map((opt) => {
                            const optLabel = opt === 'off' ? 'Désactivé' : 'PawaPay';
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
              l&apos;acheteur. KPay n&apos;est pas encore activé sur ce flux (intégration en
              cours) : PawaPay est utilisé pour tous les pays, ce choix n&apos;est pas
              modifiable pour l&apos;instant.
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
                  <div className="flex items-center gap-2 shrink-0">
                    <Button type="button" size="sm" variant="default" disabled>
                      PawaPay
                    </Button>
                    {value === 'kpay' && (
                      <span className="text-xs text-amber-700">
                        (réglage KPay existant ignoré — sera remis à PawaPay au prochain enregistrement)
                      </span>
                    )}
                  </div>
                </div>
              );
            })}
          </CardContent>
        </Card>

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Communauté WhatsApp</CardTitle>
            <CardDescription>
              Lien d&apos;invitation ajouté en pied des emails (bienvenue, passage vendeur,
              messages groupés et messages admin). Le lien du pays de l&apos;utilisateur est
              utilisé s&apos;il est renseigné ; sinon on retombe sur le lien général. Colle un
              lien <code>chat.whatsapp.com/…</code> ou un lien de communauté.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="space-y-1.5">
              <Label htmlFor="wa-general">Lien général (par défaut)</Label>
              <Input
                id="wa-general"
                type="url"
                placeholder="https://chat.whatsapp.com/…"
                value={whatsappLinks.general || ''}
                onChange={(e) => setWhatsappLinks((p) => ({ ...p, general: e.target.value }))}
              />
            </div>
            <div className="pt-2 space-y-2">
              <h3 className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                Liens par pays (optionnels)
              </h3>
              {CHECKOUT_COUNTRIES.map((country) => (
                <div key={country.iso3} className="flex items-center gap-3">
                  <span className="text-sm w-40 shrink-0">{country.label}</span>
                  <Input
                    type="url"
                    placeholder="(utilise le lien général)"
                    value={whatsappLinks[country.iso3] || ''}
                    onChange={(e) =>
                      setWhatsappLinks((p) => ({ ...p, [country.iso3]: e.target.value }))
                    }
                  />
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <Button onClick={handleSave} disabled={saving} className="w-full font-semibold">
          {saving ? 'Enregistrement...' : 'Enregistrer les modifications'}
        </Button>
      </section>
    </main>
  );
}
