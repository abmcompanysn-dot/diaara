'use client';

import { useEffect, useState } from 'react';
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
//
// Repli email : si l'envoi SMS échoue (quota Firebase épuisé, numéro non
// joignable...), l'utilisateur peut confirmer via un code envoyé par email
// à la place — moins fort comme preuve de possession du numéro, mais évite
// de bloquer un compte déjà authentifié pour une raison hors de son contrôle.
export function PhoneVerifyForm({ phone, onVerified, onSkip, skipLabel }: PhoneVerifyFormProps) {
  const { refresh } = useAuth();
  const [mode, setMode] = useState<'sms' | 'email'>('sms');
  const [confirmation, setConfirmation] = useState<ConfirmationResult | null>(null);
  const [emailCodeSent, setEmailCodeSent] = useState(false);
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [smsFailed, setSmsFailed] = useState(false);
  const [sending, setSending] = useState(false);
  const [verifying, setVerifying] = useState(false);

  const handleSendSms = async () => {
    setError('');
    setSending(true);
    try {
      const result = await sendPhoneVerificationCode(phone, 'recaptcha-container');
      setConfirmation(result);
    } catch (err: any) {
      setError(friendlyError(err));
      setSmsFailed(true);
    } finally {
      setSending(false);
    }
  };

  const handleVerifySms = async (e: React.FormEvent) => {
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

  const handleSendEmailFallback = async () => {
    setError('');
    setSending(true);
    try {
      await api.sendOtp('email', 'phone_verify');
      setEmailCodeSent(true);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSending(false);
    }
  };

  const handleVerifyEmailFallback = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!/^\d{6}$/.test(code)) {
      setError('Le code doit contenir 6 chiffres');
      return;
    }
    setError('');
    setVerifying(true);
    try {
      await api.verifyOtp('email', code, 'phone_verify');
      await refresh();
      onVerified();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setVerifying(false);
    }
  };

  const switchToEmailFallback = () => {
    setMode('email');
    setError('');
    setCode('');
  };

  useEffect(() => {
    if (!firebaseEnabled && mode === 'sms') {
      switchToEmailFallback();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="space-y-4">
      <div id="recaptcha-container" />

      {error && <div className="p-3 bg-destructive/10 text-destructive rounded-lg text-sm">{error}</div>}

      {mode === 'sms' ? (
        <>
          {!confirmation ? (
            <Button type="button" onClick={handleSendSms} disabled={sending} className="w-full h-10 font-semibold">
              {sending ? 'Envoi…' : `Envoyer le code par SMS au ${phone}`}
            </Button>
          ) : (
            <form onSubmit={handleVerifySms} className="space-y-4">
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

          {smsFailed && (
            <Button type="button" variant="link" onClick={switchToEmailFallback} className="w-full text-sm">
              Le SMS ne fonctionne pas ? Confirmer par email à la place
            </Button>
          )}
        </>
      ) : (
        <>
          {!emailCodeSent ? (
            <Button
              type="button"
              onClick={handleSendEmailFallback}
              disabled={sending}
              className="w-full h-10 font-semibold"
            >
              {sending ? 'Envoi…' : 'Envoyer un code de confirmation par email'}
            </Button>
          ) : (
            <form onSubmit={handleVerifyEmailFallback} className="space-y-4">
              <div className="space-y-2">
                <label htmlFor="phone-email-otp" className="text-sm font-medium text-green-900">
                  Code reçu par email
                </label>
                <Input
                  id="phone-email-otp"
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
        </>
      )}

      {onSkip && (
        <Button type="button" variant="ghost" onClick={onSkip} className="w-full text-sm text-green-900/50">
          {skipLabel || 'Plus tard'}
        </Button>
      )}
    </div>
  );
}
