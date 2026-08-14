'use client';

import { useEffect, useState } from 'react';
import { useAuth } from '@/lib/auth';
import { Card, CardContent } from '@/components/ui/card';
import { PageHeader } from '@/components/page-header';
import { EmptyState } from '@/components/empty-state';
import { firebaseEnabled } from '@/lib/firebase';
import { listenToVendorInbox, type ConversationSummary } from '@/lib/chat';
import { VendorChat } from '@/components/vendor-chat';

const AVATAR_COLORS = ['#0E6B46', '#B8860B', '#5A3FBF', '#1F5FBF', '#B23B72', '#0E9E6D'];

function initials(name: string): string {
  return name
    .split(' ')
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0]?.toUpperCase())
    .join('');
}

function avatarColor(id: string): string {
  let hash = 0;
  for (const ch of id) hash = (hash * 31 + ch.charCodeAt(0)) % AVATAR_COLORS.length;
  return AVATAR_COLORS[hash];
}

export default function VendorMessagesPage() {
  const { user } = useAuth();
  const [conversations, setConversations] = useState<ConversationSummary[]>([]);
  const [activeBuyerId, setActiveBuyerId] = useState<string | null>(null);
  const [unavailable, setUnavailable] = useState(false);

  useEffect(() => {
    if (!firebaseEnabled || !user) return;
    const unsubscribe = listenToVendorInbox(user.id, setConversations, () => setUnavailable(true));
    return unsubscribe;
  }, [user]);

  if (!firebaseEnabled || unavailable) {
    return (
      <main>
        <PageHeader back="/vendor" eyebrow="// espace vendeur" title="Messages" />
        <section className="max-w-4xl mx-auto px-4 sm:px-6 py-10">
          <EmptyState title="Messagerie non configurée" description="La messagerie en direct n'est pas encore activée sur cette plateforme." />
        </section>
      </main>
    );
  }

  const active = conversations.find((c) => c.buyerId === activeBuyerId);

  return (
    <main>
      <PageHeader
        back="/vendor"
        eyebrow="// espace vendeur"
        title="Messages"
        description="Discutez directement avec vos acheteurs"
      />

      <section className="max-w-4xl mx-auto px-4 sm:px-6 py-6">
        {conversations.length === 0 ? (
          <EmptyState title="Aucune conversation" description="Les messages de vos clients apparaîtront ici en direct." />
        ) : (
          <div className="grid gap-6 sm:grid-cols-3">
            <div className="sm:col-span-1 space-y-2">
              {conversations.map((c) => (
                <button
                  key={c.id}
                  type="button"
                  onClick={() => setActiveBuyerId(c.buyerId)}
                  className={`w-full text-left p-3 rounded-2xl border flex items-center gap-3 ${
                    activeBuyerId === c.buyerId ? 'border-green-600 bg-green-50/60' : 'border-green-900/10 bg-white hover:bg-green-900/2'
                  }`}
                >
                  <span
                    className="w-10 h-10 rounded-xl flex items-center justify-center text-white text-xs font-extrabold shrink-0"
                    style={{ background: avatarColor(c.buyerId) }}
                  >
                    {initials(c.buyerName)}
                  </span>
                  <div className="min-w-0">
                    <p className="text-sm font-semibold text-green-950 truncate">{c.buyerName}</p>
                    <p className="text-xs text-green-900/50 truncate mt-0.5">{c.lastMessage}</p>
                  </div>
                </button>
              ))}
            </div>
            <div className="sm:col-span-2">
              {active && user ? (
                <VendorChat
                  buyerId={active.buyerId}
                  vendorId={user.id}
                  buyerName={active.buyerName}
                  vendorName={active.vendorName}
                  role="vendor"
                />
              ) : (
                <Card className="border-green-900/5 h-full">
                  <CardContent className="p-6 flex items-center justify-center h-80 text-sm text-green-900/50">
                    Sélectionnez une conversation
                  </CardContent>
                </Card>
              )}
            </div>
          </div>
        )}
      </section>
    </main>
  );
}
