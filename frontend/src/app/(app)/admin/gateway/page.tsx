'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { CopyIcon, CheckIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';
import { StatsTab, TransactionsTab } from './gateway-tabs';

type Tab = 'stats' | 'transactions' | 'clients';

type GatewayClient = {
  id: string;
  name: string;
  default_callback_url?: string | null;
  is_active: boolean;
  created_at: string;
};

type NewCredentials = {
  client: GatewayClient;
  api_key: string;
  hmac_secret: string;
};

function CopyField({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = useState(false);
  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Presse-papiers indisponible : rien d'autre à faire.
    }
  };
  return (
    <div>
      <p className="text-xs font-semibold text-green-900 mb-1">{label}</p>
      <div className="flex items-center gap-2">
        <code className="flex-1 bg-secondary/60 rounded-lg px-3 py-2.5 text-xs font-mono break-all">
          {value}
        </code>
        <Button variant="outline" className="h-10 shrink-0 gap-1.5" onClick={handleCopy}>
          {copied ? <CheckIcon size={14} /> : <CopyIcon size={14} />}
          {copied ? 'Copié' : 'Copier'}
        </Button>
      </div>
    </div>
  );
}

export default function AdminGatewayPage() {
  const [tab, setTab] = useState<Tab>('stats');
  const [clients, setClients] = useState<GatewayClient[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [name, setName] = useState('');
  const [callbackUrl, setCallbackUrl] = useState('');
  const [creating, setCreating] = useState(false);
  const [created, setCreated] = useState<NewCredentials | null>(null);

  const [toggleTarget, setToggleTarget] = useState<GatewayClient | null>(null);
  const [toggling, setToggling] = useState(false);

  const load = () => {
    setLoading(true);
    api
      .getGatewayClients()
      .then((r) => setClients(r.clients))
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    setCreating(true);
    setError('');
    try {
      const result = await api.createGatewayClient(name.trim(), callbackUrl.trim() || undefined);
      setCreated(result);
      setName('');
      setCallbackUrl('');
      load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setCreating(false);
    }
  };

  const handleToggle = async () => {
    if (!toggleTarget) return;
    const target = toggleTarget;
    setToggleTarget(null);
    setToggling(true);
    setError('');
    try {
      await api.setGatewayClientActive(target.id, !target.is_active);
      load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setToggling(false);
    }
  };

  const tabBtn = (id: Tab, label: string) => (
    <button
      onClick={() => setTab(id)}
      className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors ${
        tab === id ? 'bg-green-900 text-white' : 'text-green-900/60 hover:bg-green-900/5'
      }`}
    >
      {label}
    </button>
  );

  return (
    <main>
      <PageHeader
        back="/admin"
        eyebrow="// administration"
        title="Passerelle de paiement"
        description="Toutes les transactions passant par DIARRA pour des clients externes (ABMCY Core…)"
      />

      <section className="max-w-4xl mx-auto px-4 sm:px-6 py-10 space-y-6">
        <div className="flex gap-1 border-b border-green-900/10 pb-3">
          {tabBtn('stats', 'Chiffres')}
          {tabBtn('transactions', 'Transactions')}
          {tabBtn('clients', 'Clients & clés')}
        </div>

        {tab === 'stats' && <StatsTab />}
        {tab === 'transactions' && <TransactionsTab />}

        {tab === 'clients' && (loading ? (
          <PageLoader />
        ) : (
        <div className="space-y-6">
        {error && (
          <div className="p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {created && (
          <div className="p-6 border-2 border-green-700 rounded-xl bg-green-50 space-y-4">
            <div>
              <h2 className="font-display font-bold text-lg text-green-950">
                « {created.client.name} » créé
              </h2>
              <p className="text-sm text-green-900/70">
                Copiez ces deux valeurs <strong>maintenant</strong> — elles ne seront plus jamais
                affichées. DIARRA n&apos;en garde que les empreintes ; si elles sont perdues, il faut
                créer un nouveau client.
              </p>
            </div>
            <CopyField label="Clé API (X-Gateway-Key)" value={created.api_key} />
            <CopyField label="Secret HMAC (signature des requêtes)" value={created.hmac_secret} />
            <Button variant="outline" className="h-10" onClick={() => setCreated(null)}>
              J&apos;ai copié, masquer
            </Button>
          </div>
        )}

        <form
          onSubmit={handleCreate}
          className="p-6 border rounded-xl bg-white shadow-card border-green-900/10 space-y-4"
        >
          <h2 className="font-display font-bold text-lg text-green-950">Nouveau client</h2>
          <div>
            <label className="text-sm font-semibold text-green-900 block mb-1" htmlFor="gw-name">
              Nom
            </label>
            <input
              id="gw-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="ABMCY Core"
              className="w-full h-10 px-3 rounded-lg border border-green-900/15 text-sm"
              required
            />
          </div>
          <div>
            <label className="text-sm font-semibold text-green-900 block mb-1" htmlFor="gw-cb">
              URL de callback par défaut <span className="text-green-900/40">(optionnel)</span>
            </label>
            <input
              id="gw-cb"
              value={callbackUrl}
              onChange={(e) => setCallbackUrl(e.target.value)}
              placeholder="https://core.diarra.app/webhooks/diarra"
              className="w-full h-10 px-3 rounded-lg border border-green-900/15 text-sm"
              type="url"
            />
            <p className="text-xs text-green-900/50 mt-1">
              Indicatif seulement — chaque transaction doit passer son propre <code>callback_url</code>{' '}
              pour être relayée.
            </p>
          </div>
          <Button type="submit" className="h-10" disabled={creating || !name.trim()}>
            {creating ? 'Création...' : 'Créer le client'}
          </Button>
        </form>

        <div className="p-6 border rounded-xl bg-white shadow-card border-green-900/10">
          <h2 className="font-display font-bold text-lg text-green-950 mb-4">
            Clients ({clients.length})
          </h2>
          {clients.length === 0 ? (
            <p className="text-sm text-green-900/50 italic">Aucun client pour le moment.</p>
          ) : (
            <ul className="divide-y divide-green-900/10">
              {clients.map((c) => (
                <li key={c.id} className="py-3 flex items-center justify-between gap-4">
                  <div className="min-w-0">
                    <p className="text-sm font-semibold text-green-950 flex items-center gap-2">
                      {c.name}
                      <span
                        className={
                          c.is_active
                            ? 'text-[10px] font-bold uppercase px-1.5 py-0.5 rounded bg-green-100 text-green-800'
                            : 'text-[10px] font-bold uppercase px-1.5 py-0.5 rounded bg-red-100 text-red-800'
                        }
                      >
                        {c.is_active ? 'actif' : 'désactivé'}
                      </span>
                    </p>
                    <p className="text-xs text-green-900/50 font-mono truncate">{c.id}</p>
                    {c.default_callback_url && (
                      <p className="text-xs text-green-900/50 truncate">{c.default_callback_url}</p>
                    )}
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-9 shrink-0"
                    disabled={toggling}
                    onClick={() => setToggleTarget(c)}
                  >
                    {c.is_active ? 'Désactiver' : 'Réactiver'}
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </div>
        </div>
        ))}
      </section>

      <ConfirmDialog
        open={!!toggleTarget}
        title={toggleTarget?.is_active ? 'Désactiver ce client ?' : 'Réactiver ce client ?'}
        description={
          toggleTarget?.is_active
            ? `« ${toggleTarget?.name} » ne pourra plus appeler la passerelle — toute requête signée avec sa clé sera refusée.`
            : `« ${toggleTarget?.name} » pourra de nouveau appeler la passerelle avec sa clé existante.`
        }
        confirmLabel={toggleTarget?.is_active ? 'Désactiver' : 'Réactiver'}
        cancelLabel="Annuler"
        danger={toggleTarget?.is_active}
        onConfirm={handleToggle}
        onCancel={() => setToggleTarget(null)}
      />
    </main>
  );
}
