'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { Button } from '@/components/ui/button';
import { firebaseEnabled, signInWithGoogle } from '@/lib/firebase';
import { friendlyError } from '@/lib/error-messages';

export function GoogleLoginButton({ onError }: { onError: (message: string) => void }) {
  const router = useRouter();
  const { login } = useAuth();
  const [loading, setLoading] = useState(false);

  // Masqué tant que le projet Firebase n'est pas configuré (NEXT_PUBLIC_FIREBASE_*).
  if (!firebaseEnabled) return null;

  const handleClick = async () => {
    setLoading(true);
    onError('');
    try {
      const idToken = await signInWithGoogle();
      const result = await api.googleLogin(idToken);
      await login(result.access_token);
      router.push('/dashboard');
    } catch (err: any) {
      onError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      <div className="flex items-center gap-3 my-4">
        <div className="h-px flex-1 bg-green-900/10" />
        <span className="text-xs text-green-900/40 font-mono">ou</span>
        <div className="h-px flex-1 bg-green-900/10" />
      </div>
      <Button
        type="button"
        variant="outline"
        onClick={handleClick}
        disabled={loading}
        className="w-full h-10 font-medium gap-2"
      >
        <svg width="16" height="16" viewBox="0 0 48 48" aria-hidden>
          <path fill="#FFC107" d="M43.6 20.5H42V20H24v8h11.3C33.9 32.6 29.4 36 24 36c-6.6 0-12-5.4-12-12s5.4-12 12-12c3.1 0 5.9 1.2 8 3.1l5.7-5.7C34.6 6.5 29.6 4.5 24 4.5 12.7 4.5 3.5 13.7 3.5 25S12.7 45.5 24 45.5 44.5 36.3 44.5 25c0-1.5-.2-3-.9-4.5z"/>
          <path fill="#FF3D00" d="M6.3 14.7l6.6 4.8C14.6 15.5 18.9 12.5 24 12.5c3.1 0 5.9 1.2 8 3.1l5.7-5.7C34.6 6.5 29.6 4.5 24 4.5c-7.7 0-14.4 4.4-17.7 10.2z"/>
          <path fill="#4CAF50" d="M24 45.5c5.5 0 10.4-1.9 14.2-5.1l-6.6-5.4c-2 1.5-4.6 2.5-7.6 2.5-5.4 0-9.9-3.4-11.5-8.2l-6.6 5.1C9.5 40.9 16.2 45.5 24 45.5z"/>
          <path fill="#1976D2" d="M43.6 20.5H42V20H24v8h11.3c-.7 2-2 3.8-3.7 5l6.6 5.4C41.5 35.6 44.5 30.7 44.5 25c0-1.5-.2-3-.9-4.5z"/>
        </svg>
        {loading ? 'Connexion…' : 'Continuer avec Google'}
      </Button>
    </>
  );
}
