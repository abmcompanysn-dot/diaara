'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { CheckIcon } from '@/components/icons';
import { PAYOUT_COUNTRIES, PAYDUNYA_COUNTRIES, CHECKOUT_COUNTRIES, LOGICAL_OPERATORS } from '@/lib/operators';
import { friendlyError } from '@/lib/error-messages';

interface VendorChatYesProps {
  productId: string;
  priceCfa: number;
  country: string;
  referralLinkId?: string;
}

// Regroupe les chiffres par paires — même logique que checkout-view.tsx et
// PayoutMethodForm.
function formatPhoneDisplay(digits: string): string {
  const groups: string[] = [];
  let i = 0;
  while (i < digits.length) {
    const remaining = digits.length - i;
    const take = remaining <= 3 ? remaining : 2;
    groups.push(digits.slice(i, i + take));
    i += take;
  }
  return groups.join(' ');
}

// Achat conversationnel via YES.abmcy Business (bêta, réservé aux vendeurs
// activés — voir Product.yes_chat_enabled). Remplace VendorChat (chat
// Firebase gratuit) pour ces vendeurs : un micro-ticket payant ouvre la
// discussion, le solde se règle plus tard une fois l'acheteur convaincu.
// Le paiement du micro-ticket passe par un dépôt PawaPay direct (l'acheteur
// choisit son opérateur et saisit son numéro ici, comme sur le checkout
// classique) — la Payment Page hébergée PawaPay était cassée (incident
// 2026-09-24, voir checkout-view.tsx).
export function VendorChatYes({ productId, priceCfa, country: initialCountry, referralLinkId }: VendorChatYesProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [expanded, setExpanded] = useState(false);

  const [country, setCountry] = useState(initialCountry || 'SEN');
  const [operator, setOperator] = useState('');
  const [phoneDigits, setPhoneDigits] = useState('');
  const [phoneTouched, setPhoneTouched] = useState(false);
  // Prestataire résolu par opérateur logique exact (voir checkout-view.tsx).
  const [operatorProviders, setOperatorProviders] = useState<Record<string, 'pawapay' | 'paydunya'>>({});
  // Opérateurs exigeant un code OTP obtenu par l'acheteur AVANT de payer
  // (Orange Money CI/BFA via PayDunya) — voir checkout-view.tsx.
  const [requiresOTP, setRequiresOTP] = useState<Record<string, boolean>>({});
  const [otp, setOtp] = useState('');

  useEffect(() => {
    api
      .getCheckoutConfig()
      .then((res) => {
        setOperatorProviders(res.operator_providers || {});
        setRequiresOTP(res.requires_otp || {});
      })
      .catch(() => {});
  }, []);

  const microTicketLabel = '600 FCFA'; // valeur par défaut affichée ; le montant réel exact vient du backend au moment du paiement

  const payoutCountry =
    PAYOUT_COUNTRIES.find((c) => c.code === country) ||
    PAYDUNYA_COUNTRIES.find((c) => c.code === country) ||
    PAYOUT_COUNTRIES[0];
  const operators = LOGICAL_OPERATORS.filter(
    (o) => o.country === country && Object.prototype.hasOwnProperty.call(operatorProviders, o.provider)
  );
  // Sélecteur de pays : union des pays PawaPay réellement actifs
  // (CHECKOUT_COUNTRIES) + PayDunya — voir checkout-view.tsx.
  const mobileMoneyCountries = [
    ...CHECKOUT_COUNTRIES.map((c) => ({ code: c.code, name: c.name })),
    ...PAYDUNYA_COUNTRIES.filter((c) => !CHECKOUT_COUNTRIES.some((p) => p.code === c.code)).map((c) => ({ code: c.code, name: c.name })),
  ];

  useEffect(() => {
    if (!operators.find((o) => o.provider === operator)) {
      setOperator(operators[0]?.provider || '');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [country]);

  const phoneValid = phoneDigits.length === payoutCountry.phoneLength;
  const phoneError =
    phoneTouched && phoneDigits.length > 0 && !phoneValid
      ? `Le numéro doit contenir ${payoutCountry.phoneLength} chiffres (actuellement ${phoneDigits.length}).`
      : '';
  const canPay = Boolean(operator) && phoneValid && (!requiresOTP[operator] || otp.length >= 4);

  const handlePhoneChange = (raw: string) => {
    let digits = raw.replace(/\D/g, '');
    if (country !== 'BEN') {
      digits = digits.replace(/^0+/, '');
    }
    digits = digits.slice(0, payoutCountry.phoneLength);
    setPhoneDigits(digits);
  };

  const handleOpen = async () => {
    if (!expanded) {
      setExpanded(true);
      return;
    }
    setPhoneTouched(true);
    if (!canPay) return;
    setError('');
    setLoading(true);
    try {
      const result = await api.openVendorConversation({
        product_id: productId,
        referral_link_id: referralLinkId,
        country,
        phone: phoneDigits,
        operator,
        ...(requiresOTP[operator] ? { otp } : {}),
      });
      window.location.href = result.payment_redirect_url;
    } catch (err: any) {
      setError(friendlyError(err));
      setLoading(false);
    }
  };

  return (
    <div className="rounded-xl border border-green-900/10 bg-white p-4">
      <p className="text-sm text-green-900/70">
        Posez vos questions au vendeur avant d&apos;acheter « {priceCfa.toLocaleString('fr-FR')} FCFA ».
        Un ticket d&apos;entrée de <strong>{microTicketLabel}</strong> ouvre la discussion ; vous ne payez
        le solde que si vous êtes convaincu.
      </p>

      {expanded && (
        <div className="space-y-3 mt-3">
          <div className="space-y-1.5">
            <Label htmlFor="vc-country" className="text-xs">Pays</Label>
            <Select
              value={country}
              onValueChange={(v) => {
                setCountry(v || 'SEN');
                setPhoneDigits('');
                setPhoneTouched(false);
              }}
            >
              <SelectTrigger id="vc-country" className="bg-white w-full h-9">
                <SelectValue placeholder="Choisir le pays" />
              </SelectTrigger>
              <SelectContent>
                {mobileMoneyCountries.map((c) => (
                  <SelectItem key={c.code} value={c.code}>
                    {c.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <Label className="text-xs">Opérateur mobile money</Label>
            <div className="grid grid-cols-3 gap-2">
              {operators.map((op) => {
                const active = op.provider === operator;
                return (
                  <button
                    key={op.provider}
                    type="button"
                    onClick={() => setOperator(op.provider)}
                    aria-pressed={active}
                    className={`relative flex flex-col items-center justify-center gap-1 rounded-lg border-2 p-2 h-16 transition-all ${
                      active
                        ? 'border-green-600 bg-green-50/60'
                        : 'border-green-900/10 hover:border-green-900/25 bg-white'
                    }`}
                  >
                    {active && (
                      <span className="absolute top-1 right-1 w-3.5 h-3.5 rounded-full bg-green-600 text-white flex items-center justify-center">
                        <CheckIcon size={9} />
                      </span>
                    )}
                    {op.logo ? (
                      <img src={`/payments/${op.logo}`} alt={op.label} className="max-h-5 max-w-[85%] object-contain" />
                    ) : (
                      <span className={`px-1.5 py-1 rounded text-[10px] font-bold ${op.badgeColor || 'bg-green-100'} ${op.badgeText || 'text-green-950'}`}>
                        {op.label}
                      </span>
                    )}
                  </button>
                );
              })}
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="vc-phone" className="text-xs">Numéro de téléphone</Label>
            <div className="flex items-center gap-2">
              <span className="shrink-0 h-9 px-2.5 rounded-md border border-green-900/15 bg-green-50/60 flex items-center font-mono text-xs text-green-900/70">
                +{payoutCountry.dialCode}
              </span>
              <Input
                id="vc-phone"
                type="tel"
                inputMode="numeric"
                placeholder={`${payoutCountry.phoneLength} chiffres`}
                value={formatPhoneDisplay(phoneDigits)}
                onChange={(e) => handlePhoneChange(e.target.value)}
                onBlur={() => setPhoneTouched(true)}
                className={`h-9 ${phoneError ? 'bg-white border-red-400 focus-visible:ring-red-400' : 'bg-white'}`}
              />
            </div>
            {phoneError && <p className="text-xs text-red-600">{phoneError}</p>}
          </div>

          {requiresOTP[operator] && (
            <div className="space-y-1.5">
              <Label htmlFor="vc-otp" className="text-xs">Code de confirmation Orange Money</Label>
              <Input
                id="vc-otp"
                type="text"
                inputMode="numeric"
                placeholder="Code reçu"
                value={otp}
                onChange={(e) => setOtp(e.target.value.replace(/\D/g, ''))}
                className="h-9 bg-white"
              />
              <p className="text-xs text-green-900/50">
                {country === 'CIV'
                  ? "Composez #144*82# puis l'option 2, saisissez le code reçu ici."
                  : 'Un code vous a été envoyé par SMS par Orange Money — saisissez-le ici.'}
              </p>
            </div>
          )}
        </div>
      )}

      {error && (
        <p className="text-xs text-destructive mt-2" role="alert">
          {error}
        </p>
      )}
      <Button
        onClick={handleOpen}
        disabled={loading || (expanded && !canPay)}
        className="w-full mt-3"
      >
        {loading ? 'Ouverture…' : expanded ? `Payer ${microTicketLabel} et discuter` : 'Discuter avec le vendeur'}
      </Button>
    </div>
  );
}

// Variante affichée à un visiteur non connecté : invite à se connecter avant
// d'ouvrir la conversation (même paiement en jeu, il faut un compte DIARRA).
export function VendorChatYesLoggedOut() {
  return (
    <p className="text-sm text-green-900/60">
      <Link href="/auth/login" className="text-green-700 font-medium hover:underline">
        Connectez-vous
      </Link>{' '}
      pour discuter avec le vendeur.
    </p>
  );
}
