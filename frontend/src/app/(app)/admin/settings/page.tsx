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
import { LOGICAL_OPERATORS, type LogicalOperator } from '@/lib/operators';

// Libellés de pays pour le groupement du tableau — LOGICAL_OPERATORS
// (lib/operators.ts) porte déjà les codes ISO3, ce mapping ne sert qu'à
// l'affichage.
const COUNTRY_LABELS: Record<string, string> = {
  SEN: 'Sénégal', CIV: "Côte d'Ivoire", BEN: 'Bénin', BFA: 'Burkina Faso',
  CMR: 'Cameroun', GAB: 'Gabon', COG: 'Congo-Brazzaville', COD: 'RD Congo',
  GHA: 'Ghana', NGA: 'Nigeria', KEN: 'Kenya', RWA: 'Rwanda', UGA: 'Ouganda',
  TZA: 'Tanzanie', ZMB: 'Zambie', MWI: 'Malawi', MOZ: 'Mozambique',
  LSO: 'Lesotho', SLE: 'Sierra Leone', ETH: 'Éthiopie', MLI: 'Mali', TGO: 'Togo',
};

// OPERATORS — tableau unique "Mobile Money" (achat ET versement, même
// réglage admin — voir model.GatewayOperatorSettingKey côté backend).
// Fusion PawaPay/PayDunya par opérateur physique (voir LOGICAL_OPERATORS).
const OPERATORS: (LogicalOperator & { countryLabel: string })[] = LOGICAL_OPERATORS.map((op) => ({
  ...op,
  countryLabel: COUNTRY_LABELS[op.country] || op.country,
}));


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

type GatewayValue = 'off' | 'pawapay' | 'paydunya';
type CheckoutValue = 'pawapay' | 'paydunya';

// Pays couverts par PayDunya (voir backend/internal/payment/paydunya_operators.go
// PayDunyaOperators) — ailleurs, seul "pawapay" reste sélectionnable pour le
// checkout par pays (interrupteur général, niveau 1).
const PAYDUNYA_COUNTRIES = new Set(['SEN', 'BEN', 'CIV', 'TGO', 'MLI', 'BFA', 'CMR']);

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
  const [cardPaymentEnabled, setCardPaymentEnabled] = useState(true);
  const [yesMicroTicketAmount, setYesMicroTicketAmount] = useState('600');
  const [gatewayOps, setGatewayOps] = useState<Record<string, GatewayValue>>({});
  const [checkoutProviders, setCheckoutProviders] = useState<Record<string, CheckoutValue>>({});
  // Liens communauté WhatsApp : clé "general" + une clé par ISO3.
  const [whatsappLinks, setWhatsappLinks] = useState<Record<string, string>>({});

  useEffect(() => {
    api
      .getAdminSettings()
      .then(({ settings }) => {
        setCommissionRate(settings.commission_rate_pct || '15');
        setCardPaymentEnabled(settings.card_payment_enabled !== 'false');
        setYesMicroTicketAmount(settings.yes_micro_ticket_amount_cfa || '600');
        const g: Record<string, GatewayValue> = {};
        for (const op of OPERATORS) {
          const v = settings[gatewayOpKey(op.provider)];
          const fallback: GatewayValue = op.pawaPayProvider ? 'pawapay' : 'paydunya';
          g[op.provider] = v === 'off' || v === 'pawapay' || v === 'paydunya' ? v : fallback;
        }
        setGatewayOps(g);
        const c: Record<string, CheckoutValue> = {};
        for (const country of CHECKOUT_COUNTRIES) {
          const v = settings[checkoutProviderKey(country.iso3)];
          c[country.iso3] = v === 'paydunya' && PAYDUNYA_COUNTRIES.has(country.iso3) ? 'paydunya' : 'pawapay';
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
    const microTicket = parseInt(yesMicroTicketAmount, 10);
    if (isNaN(microTicket) || microTicket <= 0) {
      setError('Le montant du ticket de discussion (YES) doit être un nombre positif.');
      return;
    }
    setSaving(true);
    try {
      const values: Record<string, string> = {
        commission_rate_pct: String(rate),
        card_payment_enabled: String(cardPaymentEnabled),
        yes_micro_ticket_amount_cfa: String(microTicket),
      };
      for (const op of OPERATORS)
        values[gatewayOpKey(op.provider)] = gatewayOps[op.provider] || (op.pawaPayProvider ? 'pawapay' : 'paydunya');
      for (const country of CHECKOUT_COUNTRIES)
        values[checkoutProviderKey(country.iso3)] =
          checkoutProviders[country.iso3] === 'paydunya' && PAYDUNYA_COUNTRIES.has(country.iso3)
            ? 'paydunya'
            : 'pawapay';
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
            <CardTitle>Ticket d&apos;entrée « Discuter avec le vendeur » (YES, bêta)</CardTitle>
            <CardDescription>
              Montant payé par l&apos;acheteur pour ouvrir une conversation avec le vendeur, avant
              de payer le solde. Ne s&apos;applique qu&apos;aux vendeurs pour qui cette bêta est
              activée (voir /admin/users).
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 max-w-xs">
              <Label htmlFor="yes-micro-ticket">Montant (FCFA)</Label>
              <Input
                id="yes-micro-ticket"
                type="number"
                min={1}
                step="1"
                value={yesMicroTicketAmount}
                onChange={(e) => setYesMicroTicketAmount(e.target.value)}
              />
            </div>
          </CardContent>
        </Card>

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Paiement carte bancaire / PayPal</CardTitle>
            <CardDescription>
              Active ou désactive le bouton « Carte bancaire / PayPal » au checkout (géré par
              PayPal). Désactivé, seul Mobile Money (PawaPay) reste proposé aux acheteurs — utile
              en cas de souci côté PayPal, sans avoir besoin de redéployer.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex gap-2">
              {([true, false] as const).map((opt) => (
                <Button
                  key={String(opt)}
                  type="button"
                  size="sm"
                  variant={cardPaymentEnabled === opt ? 'default' : 'outline'}
                  onClick={() => setCardPaymentEnabled(opt)}
                >
                  {opt ? 'Activé' : 'Désactivé'}
                </Button>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Mobile Money — par opérateur</CardTitle>
            <CardDescription>
              Pour chaque opérateur, choisis le prestataire (PawaPay ou PayDunya) ou désactive-le.
              Ce réglage pilote À LA FOIS l&apos;achat (l&apos;opérateur n&apos;apparaît plus au
              checkout si désactivé) et les versements vendeur. Une ligne sans réglage particulier
              suit l&apos;interrupteur général du pays (voir « Paiement à l&apos;achat — par pays »
              ci-dessous). PayDunya n&apos;est proposé que pour les opérateurs qu&apos;il couvre.
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
                    const pawaPayAvailable = Boolean(op.pawaPayProvider);
                    const paydunyaAvailable = Boolean(op.payDunyaProvider);
                    const value = gatewayOps[op.provider] || (pawaPayAvailable ? 'pawapay' : 'paydunya');
                    const options: { opt: GatewayValue; label: string }[] = [
                      { opt: 'off', label: 'Désactivé' },
                      ...(pawaPayAvailable ? [{ opt: 'pawapay' as GatewayValue, label: 'PawaPay' }] : []),
                      ...(paydunyaAvailable ? [{ opt: 'paydunya' as GatewayValue, label: 'PayDunya' }] : []),
                    ];
                    return (
                      <div
                        key={op.provider}
                        className="flex items-center justify-between gap-3 p-3 rounded-lg border border-border"
                      >
                        <span className="text-sm font-medium">{op.label}</span>
                        <div className="flex gap-1 shrink-0">
                          {options.map(({ opt, label: optLabel }) => (
                            <Button
                              key={opt}
                              type="button"
                              size="sm"
                              variant={value === opt ? 'default' : 'outline'}
                              onClick={() =>
                                setGatewayOps((prev) => ({ ...prev, [op.provider]: opt }))
                              }
                            >
                              {optLabel}
                            </Button>
                          ))}
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
              l&apos;acheteur. PayDunya n&apos;est disponible que pour les pays qu&apos;il
              couvre (Sénégal, Bénin, Côte d&apos;Ivoire, Togo, Mali, Burkina Faso,
              Cameroun) — PawaPay reste le seul choix ailleurs.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            {CHECKOUT_COUNTRIES.map((country) => {
              const value = checkoutProviders[country.iso3] || 'pawapay';
              const paydunyaAvailable = PAYDUNYA_COUNTRIES.has(country.iso3);
              return (
                <div
                  key={country.iso3}
                  className="flex items-center justify-between gap-3 p-3 rounded-lg border border-border"
                >
                  <span className="text-sm font-medium">{country.label}</span>
                  <div className="flex items-center gap-2 shrink-0">
                    <Button
                      type="button"
                      size="sm"
                      variant={value === 'pawapay' ? 'default' : 'outline'}
                      onClick={() => setCheckoutProviders((p) => ({ ...p, [country.iso3]: 'pawapay' }))}
                    >
                      PawaPay
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant={value === 'paydunya' ? 'default' : 'outline'}
                      disabled={!paydunyaAvailable}
                      onClick={() => setCheckoutProviders((p) => ({ ...p, [country.iso3]: 'paydunya' }))}
                    >
                      PayDunya
                    </Button>
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
