'use client';

import { useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { useAuth } from '@/lib/auth';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { firebaseEnabled } from '@/lib/firebase';
import { listenToConversation, sendMessage, type ChatMessage } from '@/lib/chat';

interface VendorChatProps {
  buyerId: string;
  vendorId: string;
  buyerName: string;
  vendorName: string;
  /** Rôle de l'utilisateur connecté dans cette conversation. */
  role: 'buyer' | 'vendor';
}

export function VendorChat({ buyerId, vendorId, buyerName, vendorName, role }: VendorChatProps) {
  const { user } = useAuth();
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [body, setBody] = useState('');
  const [sending, setSending] = useState(false);
  const [unavailable, setUnavailable] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!firebaseEnabled) return;
    const unsubscribe = listenToConversation(buyerId, vendorId, setMessages, () => setUnavailable(true));
    return unsubscribe;
  }, [buyerId, vendorId]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages.length]);

  if (!firebaseEnabled) return null;

  if (!user) {
    return (
      <p className="text-sm text-green-900/60">
        <Link href="/auth/login" className="text-green-700 font-medium hover:underline">
          Connectez-vous
        </Link>{' '}
        pour contacter {role === 'buyer' ? 'le vendeur' : 'ce client'}.
      </p>
    );
  }

  if (unavailable) {
    return <p className="text-sm text-green-900/50">Messagerie temporairement indisponible.</p>;
  }

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!body.trim() || sending) return;
    setSending(true);
    try {
      await sendMessage(buyerId, vendorId, buyerName, vendorName, user.id, role, body.trim());
      setBody('');
    } catch {
      setUnavailable(true);
    } finally {
      setSending(false);
    }
  };

  return (
    <div className="flex flex-col h-80 border border-green-900/10 rounded-xl bg-white overflow-hidden">
      <div className="flex-1 overflow-y-auto p-3 space-y-2">
        {messages.length === 0 ? (
          <p className="text-xs text-green-900/40 text-center mt-8">Aucun message pour l&apos;instant — dites bonjour !</p>
        ) : (
          messages.map((m) => (
            <div
              key={m.id}
              className={`max-w-[80%] rounded-2xl px-3 py-2 text-sm ${
                m.senderRole === role ? 'ml-auto bg-green-950 text-white' : 'bg-green-50 text-green-950'
              }`}
            >
              {m.body}
            </div>
          ))
        )}
        <div ref={bottomRef} />
      </div>
      <form onSubmit={handleSend} className="flex items-center gap-2 p-2 border-t border-green-900/10">
        <Input
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder="Votre message…"
          className="bg-white"
        />
        <Button type="submit" size="sm" disabled={sending || !body.trim()}>
          Envoyer
        </Button>
      </form>
    </div>
  );
}
