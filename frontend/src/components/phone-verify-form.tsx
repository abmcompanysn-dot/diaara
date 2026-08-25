'use client';

import { useState } from 'react';
import type { ConfirmationResult } from 'firebase/auth';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { sendPhoneVerificationCode, confirmPhoneCode, firebaseEnabled } from '@/lib/firebase';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { friendlyError } from '@/lib/error-messages';

interface PhoneVerifyFormProps {
  phone: string;
  onVerified: () => void;
  onSkip?: () => void;
  skipLabel?: string;
}

// Vérification du téléphone via Firebase Phone Auth : Firebase envoie et
// vérifie le SMS lui-même (reCAPTCHA invisible), le backend ne fait que
// valider l'ID token obtenu. Remplace l'ancien flux OTP maison pour le
// canal SMS (toujours utilisé côté email, voir OtpForm).
export function PhoneVerifyForm({ phone, onVerified, onSkip, skipLabel }: PhoneVerifyFormProps) {
  const { refresh } = useAuth();
  const [confirmation, setConfirmation] = useState<ConfirmationResult | null>(null);
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [sending, setSending] = useState(false);
  const [verifying, setVerifying] = useState(false);

  const handleSend = async () => {
    setError('');
    setSending(true);
    try {
      const result = await sendPhoneVerificationCode(phone, 'recaptcha-container');
      setConfirmation(result);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSending(false);
    }
  };

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!confirmation || !/^\d{6}$/.test(code)) {
      setError('Le code doit contenir 6 chiffres');
      return;
    }
    setError('');
    setVerifying(true);
    try {
      const idToken = await confirmPhoneCode(confirmation, code);
      await api.verifyPhoneFirebase(idToken);
      await refresh();
      onVerified();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setVerifying(false);
    }
  };

  if (!firebaseEnabled) {
    return (
      <p className="text-sm text-green-900/60">
        La vérification par téléphone n&apos;est pas disponible pour le moment.
      </p>
    );
  }

  return (
    <div className="space-y-4">
      <div id="recaptcha-container" />

      {error && <div className="p-3 bg-destructive/10 text-destructive rounded-lg text-sm">{error}</div>}

      {!confirmation ? (
        <Button type="button" onClick={handleSend} disabled={sending} className="w-full h-10 font-semibold">
          {sending ? 'Envoi…' : `Envoyer le code par SMS au ${phone}`}
        </Button>
      ) : (
        <form onSubmit={handleVerify} className="space-y-4">
          <div className="space-y-2">
            <label htmlFor="phone-otp" className="text-sm font-medium text-green-900">
              Code reçu par SMS
            </label>
            <Input
              id="phone-otp"
              inputMode="numeric"
              autoComplete="one-time-code"
              pattern="[0-9]{6}"
              maxLength={6}
              placeholder="••••••"
              value={code}
              onChange={(e) => setCode(e.target.value.replace(/\D/g, ''))}
              className="h-12 text-center text-2xl tracking-[0.4em] font-mono"
              required
            />
          </div>
          <Button type="submit" disabled={verifying || code.length !== 6} className="w-full h-10 font-semibold">
            {verifying ? 'Vérification…' : 'Vérifier'}
          </Button>
        </form>
      )}

      {onSkip && (
        <Button type="button" variant="ghost" onClick={onSkip} className="w-full text-sm text-green-900/50">
          {skipLabel || 'Plus tard'}
        </Button>
      )}
    </div>
  );
}
