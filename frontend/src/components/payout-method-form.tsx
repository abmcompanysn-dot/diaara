'use client';

import { useEffect, useState } from 'react';
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
import { PAYOUT_COUNTRIES } from '@/lib/operators';

interface PayoutMethodFormProps {
  initialCountry?: string | null;
  initialOperator?: string | null;
  initialPhone?: string | null;
  onSave: (data: { phone: string; operator: string; country: string }) => void | Promise<void>;
  onCancel: () => void;
  saving: boolean;
  error: string;
}

// Regroupe les chiffres par paires, le dernier groupe absorbant le reste
// (2-3 chiffres) pour rester lisible quel que soit le nombre total de chiffres.
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

export function PayoutMethodForm({
  initialCountry,
  initialOperator,
  initialPhone,
  onSave,
  onCancel,
  saving,
  error,
}: PayoutMethodFormProps) {
  const [country, setCountry] = useState(initialCountry || 'SEN');
  const [operator, setOperator] = useState(initialOperator || '');
  // Le numéro peut être stocké avec l'indicatif (ex: 221777587999) : on ne
  // garde que les derniers chiffres attendus pour ce pays.
  const [phoneDigits, setPhoneDigits] = useState(() => {
    const len = (PAYOUT_COUNTRIES.find((c) => c.code === (initialCountry || 'SEN')) || PAYOUT_COUNTRIES[0]).phoneLength;
    return (initialPhone || '').replace(/\D/g, '').slice(-len);
  });
  const [touched, setTouched] = useState(false);

  const countryConfig = PAYOUT_COUNTRIES.find((c) => c.code === country) || PAYOUT_COUNTRIES[0];
  const operators = countryConfig.operators;

  useEffect(() => {
    if (!operators.find((o) => o.provider === operator)) {
      setOperator(operators[0]?.provider || '');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [country]);

  const handleCountryChange = (code: string | null) => {
    setCountry(code || 'SEN');
    setPhoneDigits('');
    setTouched(false);
  };

  const handlePhoneChange = (raw: string) => {
    // Bénin (229) : depuis la réforme 2021, le "01" initial fait partie du
    // numéro (pas un préfixe de tri à retirer comme ailleurs) — ex "01 90 01
    // 02 03" reste tel quel, contrairement aux autres pays où un "0" de tête
    // est un simple préfixe national à enlever avant l'indicatif.
    let digits = raw.replace(/\D/g, '');
    if (country !== 'BEN') {
      digits = digits.replace(/^0+/, '');
    }
    digits = digits.slice(0, countryConfig.phoneLength);
    setPhoneDigits(digits);
  };

  const phoneValid = phoneDigits.length === countryConfig.phoneLength;
  const phoneError =
    touched && phoneDigits.length > 0 && !phoneValid
      ? `Le numéro doit contenir ${countryConfig.phoneLength} chiffres (actuellement ${phoneDigits.length}).`
      : '';

  const canSave = Boolean(operator) && phoneValid;

  return (
    <div className="space-y-5 max-w-md">
      <div className="space-y-2">
        <Label htmlFor="pm-country">Pays</Label>
        <Select value={country} onValueChange={handleCountryChange}>
          <SelectTrigger className="bg-white">
            <SelectValue placeholder="Choisir le pays" />
          </SelectTrigger>
          <SelectContent>
            {PAYOUT_COUNTRIES.map((c) => (
              <SelectItem key={c.code} value={c.code}>
                {c.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="space-y-2">
        <Label>Opérateur mobile money</Label>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
          {operators.map((op) => {
            const active = op.provider === operator;
            return (
              <button
                key={op.provider}
                type="button"
                onClick={() => setOperator(op.provider)}
                aria-pressed={active}
                className={`relative flex flex-col items-center justify-center gap-1.5 rounded-xl border-2 p-3 h-20 transition-all ${
                  active
                    ? 'border-green-600 shadow-lift bg-green-50/60'
                    : 'border-green-900/10 hover:border-green-900/25 bg-white'
                }`}
              >
                {active && (
                  <span className="absolute top-1.5 right-1.5 w-4 h-4 rounded-full bg-green-600 text-white flex items-center justify-center">
                    <CheckIcon size={11} />
                  </span>
                )}
                {op.logo ? (
                  <>
                    <img src={`/payments/${op.logo}`} alt={op.label} className="max-h-7 max-w-[85%] object-contain" />
                    <span className="text-[11px] text-green-900/60 truncate max-w-full">{op.label}</span>
                  </>
                ) : (
                  <span className={`px-2.5 py-1.5 rounded text-xs font-bold ${op.badgeColor || 'bg-green-100'} ${op.badgeText || 'text-green-950'}`}>
                    {op.label}
                  </span>
                )}
              </button>
            );
          })}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="pm-phone">Numéro de téléphone</Label>
        <div className="flex items-center gap-2">
          <span className="shrink-0 h-10 px-3 rounded-md border border-green-900/15 bg-green-50/60 flex items-center font-mono text-sm text-green-900/70">
            +{countryConfig.dialCode}
          </span>
          <Input
            id="pm-phone"
            type="tel"
            inputMode="numeric"
            placeholder={`${countryConfig.phoneLength} chiffres`}
            value={formatPhoneDisplay(phoneDigits)}
            onChange={(e) => handlePhoneChange(e.target.value)}
            onBlur={() => setTouched(true)}
            className={phoneError ? 'bg-white border-red-400 focus-visible:ring-red-400' : 'bg-white'}
          />
        </div>
        {phoneError ? (
          <p className="text-xs text-red-600">{phoneError}</p>
        ) : (
          <p className="text-xs text-green-900/50">
            {country === 'BEN'
              ? `${countryConfig.phoneLength} chiffres, avec le 01 initial (ex: +229 01 xx xx xx xx).`
              : `${countryConfig.phoneLength} chiffres, sans le 0 initial.`}
          </p>
        )}
      </div>

      {error && (
        <div className="p-3 bg-red-50 text-red-700 rounded text-sm" role="alert">
          {error}
        </div>
      )}

      <div className="flex gap-3">
        <Button
          onClick={() => {
            setTouched(true);
            if (canSave) onSave({ phone: phoneDigits, operator, country });
          }}
          disabled={saving || !canSave}
        >
          {saving ? 'Enregistrement…' : 'Enregistrer'}
        </Button>
        <Button variant="outline" onClick={onCancel}>
          Annuler
        </Button>
      </div>
    </div>
  );
}
