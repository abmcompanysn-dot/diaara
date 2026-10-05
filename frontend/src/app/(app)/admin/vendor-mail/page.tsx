'use client';

import { useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { friendlyError } from '@/lib/error-messages';
import { cn } from '@/lib/utils';
import { ArrowLeftIcon } from '@/components/icons';

interface Thread {
  id: string;
  vendor_id: string;
  vendor_email: string;
  vendor_shop: string;
  subject: string;
  updated_at: string;
}

interface Message {
  id: string;
  thread_id: string;
  direction: 'outbound' | 'inbound';
  status: string;
  subject: string;
  body: string;
  created_at: string;
}

interface VendorOption {
  id: string;
  email: string;
  shop_name?: string;
  roles: string[];
}

type Tab = 'drafts' | 'threads' | 'start';

const TABS: { key: Tab; label: string }[] = [
  { key: 'drafts', label: 'À valider' },
  { key: 'threads', label: 'Conversations' },
  { key: 'start', label: 'Démarrer' },
];

export default function VendorMailPage() {
  const [tab, setTab] = useState<Tab>('drafts');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [drafts, setDrafts] = useState<Message[]>([]);
  const [threads, setThreads] = useState<Thread[]>([]);
  const [editingDraft, setEditingDraft] = useState<Record<string, string>>({});
  const [busyId, setBusyId] = useState<string | null>(null);
  const [approvedCount, setApprovedCount] = useState(0);

  const [activeThread, setActiveThread] = useState<Thread | null>(null);
  const [activeMessages, setActiveMessages] = useState<Message[]>([]);
  const [replyBody, setReplyBody] = useState('');
  const [replyBusy, setReplyBusy] = useState(false);

  const [vendors, setVendors] = useState<VendorOption[]>([]);
  const [startVendorId, setStartVendorId] = useState('');
  const [startSubject, setStartSubject] = useState('');
  const [startBody, setStartBody] = useState('');
  const [startBusy, setStartBusy] = useState(false);
  const [startDone, setStartDone] = useState('');
  const [vendorSearch, setVendorSearch] = useState('');

  useEffect(() => {
    load();
  }, []);

  async function load() {
    setLoading(true);
    try {
      const [d, t, u] = await Promise.all([
        api.listVendorMailDrafts(),
        api.listVendorMailThreads(),
        api.getUsers(),
      ]);
      setDrafts(d.drafts || []);
      setThreads(t.threads || []);
      setVendors((u.users || []).filter((user: any) => user.roles?.includes('vendeur')));
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  }

  async function openThread(thread: Thread) {
    setActiveThread(thread);
    setReplyBody('');
    try {
      const r = await api.getVendorMailThread(thread.id);
      setActiveMessages(r.messages || []);
    } catch (err: any) {
      setError(friendlyError(err));
    }
  }

  async function handleApprove(id: string) {
    setBusyId(id);
    try {
      const edited = editingDraft[id];
      if (edited !== undefined) {
        await api.updateVendorMailDraft(id, edited);
      }
      await api.approveVendorMailDraft(id);
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
    }
  }

  async function handleReject(id: string) {
    setBusyId(id);
    try {
      await api.rejectVendorMailDraft(id);
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
    }
  }

  // Espacement entre deux envois (ms) pour ne pas faire passer le serveur
  // mail pour un spammeur (beaucoup de providers pénalisent les rafales).
  const APPROVE_ALL_DELAY_MS = 3000;

  function sleep(ms: number) {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  async function handleApproveAll() {
    setBusyId('all');
    try {
      for (let i = 0; i < drafts.length; i++) {
        const d = drafts[i];
        const edited = editingDraft[d.id];
        if (edited !== undefined) {
          await api.updateVendorMailDraft(d.id, edited);
        }
        await api.approveVendorMailDraft(d.id);
        setApprovedCount(i + 1);
        if (i < drafts.length - 1) {
          await sleep(APPROVE_ALL_DELAY_MS);
        }
      }
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
      setApprovedCount(0);
    }
  }

  async function handleSendReply() {
    if (!activeThread || !replyBody.trim()) return;
    setReplyBusy(true);
    try {
      await api.draftVendorMailReply(activeThread.id, replyBody.trim());
      setReplyBody('');
      await load();
      setTab('drafts');
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setReplyBusy(false);
    }
  }

  async function handleStart() {
    if (!startVendorId || !startSubject.trim() || !startBody.trim()) return;
    setStartBusy(true);
    setStartDone('');
    try {
      await api.startVendorMailThread({
        vendor_id: startVendorId,
        subject: startSubject.trim(),
        body: startBody.trim(),
      });
      setStartSubject('');
      setStartBody('');
      setStartVendorId('');
      setStartDone('Brouillon créé — à valider dans l\'onglet « À valider ».');
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setStartBusy(false);
    }
  }

  const filteredVendors = useMemo(() => {
    const q = vendorSearch.trim().toLowerCase();
    if (!q) return vendors;
    return vendors.filter(
      (v) => v.email.toLowerCase().includes(q) || (v.shop_name || '').toLowerCase().includes(q)
    );
  }, [vendors, vendorSearch]);

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Discussion vendeurs"
        description="atekossibrunel@diarra.app — aucun email ne part sans validation explicite ici"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/admin" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Tableau de bord
          </Button>
        }
      />

      <section className="max-w-4xl mx-auto px-4 sm:px-6 py-10">
        <div className="flex gap-1.5 mb-6 border-b border-green-900/10 overflow-x-auto">
          {TABS.map((t) => (
            <button
              key={t.key}
              type="button"
              onClick={() => {
                setTab(t.key);
                setActiveThread(null);
              }}
              className={cn(
                'px-4 py-2.5 text-sm font-medium whitespace-nowrap border-b-2 -mb-px transition-colors',
                tab === t.key
                  ? 'border-lime text-green-950'
                  : 'border-transparent text-green-900/50 hover:text-green-900'
              )}
            >
              {t.label}
              {t.key === 'drafts' && drafts.length > 0 && (
                <span className="ml-1.5 inline-flex items-center justify-center w-5 h-5 rounded-full bg-amber-100 text-amber-800 text-[11px] font-semibold">
                  {drafts.length}
                </span>
              )}
            </button>
          ))}
        </div>

        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {loading ? (
          <PageLoader />
        ) : tab === 'drafts' ? (
          drafts.length === 0 ? (
            <EmptyState title="Aucun brouillon en attente" description="Rien à valider pour le moment." />
          ) : (
            <div className="space-y-4">
              <div className="flex justify-end">
                <Button size="sm" onClick={handleApproveAll} disabled={busyId !== null}>
                  {busyId === 'all'
                    ? `Envoi en cours… (${approvedCount}/${drafts.length}, ~3 s entre chaque)`
                    : `Valider et envoyer tout (${drafts.length})`}
                </Button>
              </div>
              {drafts.map((d) => (
                <div key={d.id} className="p-5 border rounded-xl bg-white shadow-card border-green-900/10 space-y-3">
                  <div className="flex items-center justify-between gap-3">
                    <p className="font-semibold text-green-950 text-sm">{d.subject}</p>
                    <Badge variant="outline">{new Date(d.created_at).toLocaleString('fr-FR')}</Badge>
                  </div>
                  <Textarea
                    value={editingDraft[d.id] ?? d.body}
                    onChange={(e) => setEditingDraft((prev) => ({ ...prev, [d.id]: e.target.value }))}
                    className="min-h-32 text-sm"
                  />
                  <div className="flex gap-2 justify-end">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleReject(d.id)}
                      disabled={busyId !== null}
                    >
                      Écarter
                    </Button>
                    <Button
                      size="sm"
                      onClick={() => handleApprove(d.id)}
                      disabled={busyId !== null}
                      className="bg-green-600 text-white hover:bg-green-500"
                    >
                      {busyId === d.id ? 'Envoi…' : 'Valider et envoyer'}
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )
        ) : tab === 'threads' ? (
          activeThread ? (
            <div className="space-y-4">
              <Button variant="outline" size="sm" onClick={() => setActiveThread(null)}>
                <ArrowLeftIcon size={16} className="mr-2" />
                Toutes les conversations
              </Button>
              <p className="font-semibold text-green-950">{activeThread.vendor_shop || activeThread.vendor_email}</p>
              <div className="space-y-3">
                {activeMessages.map((m) => (
                  <div
                    key={m.id}
                    className={cn(
                      'p-4 rounded-xl border max-w-[85%]',
                      m.direction === 'inbound'
                        ? 'bg-green-50/60 border-green-900/10 mr-auto'
                        : 'bg-white border-green-900/10 ml-auto'
                    )}
                  >
                    <p className="text-xs text-muted-foreground mb-1">
                      {m.direction === 'inbound' ? 'Vendeur' : 'DIARRA'} · {m.status} ·{' '}
                      {new Date(m.created_at).toLocaleString('fr-FR')}
                    </p>
                    <p className="text-sm whitespace-pre-wrap">{m.body}</p>
                  </div>
                ))}
              </div>
              <div className="pt-4 border-t border-green-900/10 space-y-2">
                <Textarea
                  value={replyBody}
                  onChange={(e) => setReplyBody(e.target.value)}
                  placeholder="Rédiger une réponse (sera mise en brouillon, à valider ensuite dans « À valider »)"
                  className="min-h-24 text-sm"
                />
                <div className="flex justify-end">
                  <Button size="sm" onClick={handleSendReply} disabled={replyBusy || !replyBody.trim()}>
                    {replyBusy ? 'Préparation…' : 'Mettre en brouillon'}
                  </Button>
                </div>
              </div>
            </div>
          ) : threads.length === 0 ? (
            <EmptyState title="Aucune conversation" description="Démarrez-en une depuis l'onglet « Démarrer »." />
          ) : (
            <div className="space-y-3">
              {threads.map((t) => (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => openThread(t)}
                  className="w-full text-left p-4 border rounded-xl bg-white shadow-card border-green-900/10 hover:border-lime transition-colors"
                >
                  <div className="flex items-center justify-between gap-3">
                    <p className="font-semibold text-green-950 text-sm">{t.vendor_shop || t.vendor_email}</p>
                    <span className="text-xs text-muted-foreground">
                      {new Date(t.updated_at).toLocaleDateString('fr-FR')}
                    </span>
                  </div>
                  <p className="text-xs text-muted-foreground mt-1">{t.subject}</p>
                </button>
              ))}
            </div>
          )
        ) : (
          <div className="space-y-4">
            <div>
              <label className="text-xs font-medium text-green-900/70 mb-1.5 block">Vendeur</label>
              <Input
                placeholder="Rechercher un vendeur (email ou boutique)"
                value={vendorSearch}
                onChange={(e) => setVendorSearch(e.target.value)}
                className="mb-2"
              />
              <div className="max-h-48 overflow-y-auto border border-green-900/10 rounded-lg divide-y divide-green-900/5">
                {filteredVendors.map((v) => (
                  <button
                    key={v.id}
                    type="button"
                    onClick={() => setStartVendorId(v.id)}
                    className={cn(
                      'w-full text-left px-3 py-2 text-sm hover:bg-green-50/60 transition-colors',
                      startVendorId === v.id && 'bg-green-50 font-semibold'
                    )}
                  >
                    {v.shop_name || v.email} <span className="text-muted-foreground">({v.email})</span>
                  </button>
                ))}
                {filteredVendors.length === 0 && (
                  <p className="px-3 py-2 text-sm text-muted-foreground italic">Aucun vendeur trouvé</p>
                )}
              </div>
            </div>
            <div>
              <label className="text-xs font-medium text-green-900/70 mb-1.5 block">Sujet</label>
              <Input value={startSubject} onChange={(e) => setStartSubject(e.target.value)} placeholder="Ex : Votre avis sur DIARRA" />
            </div>
            <div>
              <label className="text-xs font-medium text-green-900/70 mb-1.5 block">Message</label>
              <Textarea
                value={startBody}
                onChange={(e) => setStartBody(e.target.value)}
                className="min-h-32 text-sm"
                placeholder="Rédigez le premier message…"
              />
            </div>
            {startDone && <p className="text-sm text-green-700">{startDone}</p>}
            <div className="flex justify-end">
              <Button
                onClick={handleStart}
                disabled={startBusy || !startVendorId || !startSubject.trim() || !startBody.trim()}
              >
                {startBusy ? 'Création…' : 'Créer le brouillon'}
              </Button>
            </div>
          </div>
        )}
      </section>
    </main>
  );
}
