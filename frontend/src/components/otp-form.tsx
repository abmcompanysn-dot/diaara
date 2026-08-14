'use client';

import { useState } from 'react';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { friendlyError } from '@/lib/error-messages';

interface OtpFormProps {
  channel: 'email' | 'sms';
  title: string;
  description: string;
  onVerified: () => void;
  onSkip?: () => void;
  skipLabel?: string;
}

// Formulaire de saisie du code OTP (6 chiffres) avec renvoi (cooldown 1 min).
// Partagé par les pages /auth/verify-email et /auth/verify-phone.
export function OtpForm({ channel, title, description, onVerified, onSkip, skipLabel }: OtpFormProps) {
  const { refresh } = useAuth();
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [info, setInfo] = useState('');
  const [verifying, setVerifying] = useState(false);
  const [sending, setSending] = useState(false);
  const [cooldown, setCooldown] = useState(0);

  const startCooldown = () => {
    setCooldown(60);
    const timer = window.setInterval(() => {
      setCooldown((c) => {
        if (c <= 1) {
          window.clearInterval(timer);
          return 0;
        }
        return c - 1;
      });
    }, 1000);
  };

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!/^\d{6}$/.test(code)) {
      setError('Le code doit contenir 6 chiffres');
      return;
    }
    setError('');
    setVerifying(true);
    try {
      await api.verifyOtp(channel, code);
      await refresh();
      onVerified();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setVerifying(false);
    }
  };

  const handleResend = async () => {
    setError('');
    setSending(true);
    try {
      const result = await api.sendOtp(channel);
      if (result.dev_code) setInfo(`Code de test (dev) : ${result.dev_code}`);
      else setInfo('Code renvoyé. Vérifiez vos messages.');
      startCooldown();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSending(false);
    }
  };

  return (
    <form onSubmit={handleVerify} className="space-y-4">
      {description && (
        <p className="text-sm text-green-900/60">{description}</p>
      )}

      {info && <div className="p-3 bg-green-50 text-green-700 rounded-lg text-sm">{info}</div>}
      {error && (
        <div className="p-3 bg-destructive/10 text-destructive rounded-lg text-sm">{error}</div>
      )}

      <div className="space-y-2">
        <label
          htmlFor={`otp-${channel}`}
          className="text-sm font-medium text-green-900"
        >
          Code à 6 chiffres
        </label>
        <Input
          id={`otp-${channel}`}
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
        {verifying ? 'Vérification...' : 'Vérifier'}
      </Button>

      <div className="flex flex-col items-center gap-2">
        <Button
          type="button"
          variant="link"
          onClick={handleResend}
          disabled={sending || cooldown > 0}
          className="text-sm text-green-600 hover:text-green-500"
        >
          {cooldown > 0 ? `Renvoyer dans ${cooldown}s` : 'Renvoyer le code'}
        </Button>
        {onSkip && (
          <Button type="button" variant="ghost" onClick={onSkip} className="text-sm text-green-900/50">
            {skipLabel || 'Plus tard'}
          </Button>
        )}
      </div>
    </form>
  );
}
