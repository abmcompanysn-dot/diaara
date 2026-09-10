'use client';

import { useCallback, useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { friendlyError } from '@/lib/error-messages';

const fmtCFA = (n: number) => `${(n || 0).toLocaleString('fr-FR')} F`;
const fmtDate = (s?: string) =>
  s ? new Date(s).toLocaleString('fr-FR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' }) : '—';

const STATUS_STYLE: Record<string, string> = {
  completed: 'bg-green-100 text-green-800',
  pending: 'bg-amber-100 text-amber-800',
  processing: 'bg-amber-100 text-amber-800',
  failed: 'bg-red-100 text-red-800',
  cancelled: 'bg-red-100 text-red-800',
};
const TYPE_LABEL: Record<string, string> = {
  deposit: 'Encaissement',
  payout: 'Versement',
  refund: 'Remboursement',
};

function Badge({ value }: { value: string }) {
  return (
    <span
      className={`text-[10px] font-bold uppercase px-1.5 py-0.5 rounded ${
        STATUS_STYLE[value] || 'bg-gray-100 text-gray-700'
      }`}
    >
      {value}
    </span>
  );
}

// ---------------------------------------------------------------- Chiffres

export function StatsTab() {
  const [since, setSince] = useState(30);
  const [d, setD] = useState<any>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    setError('');
    api
      .getGatewayStats(since)
      .then(setD)
      .catch((e: any) => setError(friendlyError(e)));
  }, [since]);

  if (error) return <div className="p-3 bg-destructive/10 text-destructive rounded text-sm">{error}</div>;
  if (!d) return <p className="text-sm text-green-900/50">Chargement…</p>;

  const kpi = (label: string, value: string, sub?: string) => (
    <div className="p-4 border rounded-xl bg-white border-green-900/10">
      <div className="text-[11px] uppercase tracking-wide text-green-900/50">{label}</div>
      <div className="text-xl font-bold text-green-950 mt-1">{value}</div>
      {sub && <div className="text-xs text-green-900/50 mt-0.5">{sub}</div>}
    </div>
  );

  const AggTable = ({ title, rows, labelMap }: { title: string; rows: any[]; labelMap?: Record<string, string> }) => (
    <div className="p-5 border rounded-xl bg-white border-green-900/10">
      <h3 className="font-display font-bold text-green-950 mb-3">{title}</h3>
      <table className="w-full text-sm">
        <tbody>
          {(rows || []).map((r) => (
            <tr key={r.key} className="border-t border-green-900/5">
              <td className="py-2">{labelMap?.[r.key] || r.key}</td>
              <td className="py-2 text-right text-green-900/60">{r.count}</td>
              <td className="py-2 text-right font-medium">{fmtCFA(r.total_cfa)}</td>
            </tr>
          ))}
          {(!rows || rows.length === 0) && (
            <tr>
              <td className="py-2 text-green-900/40 italic">Aucune donnée</td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );

  return (
    <div className="space-y-5">
      <div className="flex gap-2">
        {[
          { d: 7, l: '7 j' },
          { d: 30, l: '30 j' },
          { d: 0, l: 'Tout' },
        ].map((p) => (
          <Button
            key={p.d}
            size="sm"
            variant={since === p.d ? 'default' : 'outline'}
            onClick={() => setSince(p.d)}
          >
            {p.l}
          </Button>
        ))}
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        {kpi('Volume total', fmtCFA(d.total_cfa), `${d.total_count} transactions`)}
        {kpi('Abouties', String(d.completed), 'completed')}
        {kpi('Échouées', String(d.failed), 'failed')}
        {kpi('En attente', String(d.pending), 'pending')}
      </div>

      <AggTable title="Par type" rows={d.by_type} labelMap={TYPE_LABEL} />
      <AggTable title="Par agrégateur" rows={d.by_provider} />
      <AggTable title="Par statut" rows={d.by_status} />

      <div className="p-5 border rounded-xl bg-white border-green-900/10">
        <h3 className="font-display font-bold text-green-950 mb-3">Par client</h3>
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-green-900/50 text-xs">
              <th className="py-1">Client</th>
              <th className="py-1 text-right">Transactions</th>
              <th className="py-1 text-right">Volume</th>
              <th className="py-1 text-right">Abouties</th>
            </tr>
          </thead>
          <tbody>
            {(d.by_client || []).map((c: any) => (
              <tr key={c.client_id} className="border-t border-green-900/5">
                <td className="py-2">{c.client_name}</td>
                <td className="py-2 text-right text-green-900/60">{c.count}</td>
                <td className="py-2 text-right font-medium">{fmtCFA(c.total_cfa)}</td>
                <td className="py-2 text-right">{c.completed}</td>
              </tr>
            ))}
            {(!d.by_client || d.by_client.length === 0) && (
              <tr>
                <td className="py-2 text-green-900/40 italic">Aucune donnée</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// ------------------------------------------------------------ Transactions

export function TransactionsTab() {
  const [rows, setRows] = useState<any[]>([]);
  const [type, setType] = useState('');
  const [status, setStatus] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busyId, setBusyId] = useState('');
  const [checkMsg, setCheckMsg] = useState<Record<string, string>>({});

  const load = useCallback(() => {
    setLoading(true);
    setError('');
    api
      .getGatewayTransactions({ type, status, limit: 100 })
      .then((r) => setRows(r.transactions || []))
      .catch((e: any) => setError(friendlyError(e)))
      .finally(() => setLoading(false));
  }, [type, status]);

  useEffect(load, [load]);

  const check = async (id: string) => {
    setBusyId(id);
    setCheckMsg((m) => ({ ...m, [id]: '' }));
    try {
      const res = await api.checkGatewayTransaction(id);
      if (res.provider_status === 'NOT_FOUND') {
        setCheckMsg((m) => ({ ...m, [id]: 'Jamais confirmé chez l’agrégateur' }));
      } else if (res.transaction) {
        setRows((rs) => rs.map((r) => (r.id === id ? { ...r, ...res.transaction, client_name: r.client_name } : r)));
        setCheckMsg((m) => ({ ...m, [id]: `→ ${res.transaction.status}` }));
      }
    } catch (e: any) {
      setCheckMsg((m) => ({ ...m, [id]: friendlyError(e) }));
    } finally {
      setBusyId('');
    }
  };

  return (
    <div className="space-y-4">
      {error && <div className="p-3 bg-destructive/10 text-destructive rounded text-sm">{error}</div>}

      <div className="flex flex-wrap gap-2">
        <select value={type} onChange={(e) => setType(e.target.value)} className="h-9 px-2 rounded border border-green-900/15 text-sm">
          <option value="">Tous les types</option>
          <option value="deposit">Encaissements</option>
          <option value="payout">Versements</option>
          <option value="refund">Remboursements</option>
        </select>
        <select value={status} onChange={(e) => setStatus(e.target.value)} className="h-9 px-2 rounded border border-green-900/15 text-sm">
          <option value="">Tous les statuts</option>
          <option value="pending">En attente</option>
          <option value="processing">En cours</option>
          <option value="completed">Abouties</option>
          <option value="failed">Échouées</option>
          <option value="cancelled">Annulées</option>
        </select>
        <Button size="sm" variant="outline" onClick={load}>
          Rafraîchir
        </Button>
      </div>

      <div className="border rounded-xl bg-white border-green-900/10 overflow-x-auto">
        <table className="w-full text-sm min-w-[820px]">
          <thead>
            <tr className="text-left text-green-900/50 text-xs border-b border-green-900/10">
              <th className="p-3">Créé</th>
              <th className="p-3">Client</th>
              <th className="p-3">Réf. client</th>
              <th className="p-3">Type</th>
              <th className="p-3">Montant</th>
              <th className="p-3">Agrégateur</th>
              <th className="p-3">Statut</th>
              <th className="p-3"></th>
            </tr>
          </thead>
          <tbody>
            {rows.map((t) => (
              <tr key={t.id} className="border-b border-green-900/5">
                <td className="p-3 text-green-900/60">{fmtDate(t.created_at)}</td>
                <td className="p-3">{t.client_name}</td>
                <td className="p-3 font-mono text-xs">{t.client_ref}</td>
                <td className="p-3">{TYPE_LABEL[t.type] || t.type}</td>
                <td className="p-3">{fmtCFA(t.amount_cfa)}</td>
                <td className="p-3 text-green-900/60">{t.provider}</td>
                <td className="p-3">
                  <Badge value={t.status} />
                  {t.failure_reason && (
                    <div className="text-[11px] text-green-900/50 mt-0.5">{t.failure_reason}</div>
                  )}
                </td>
                <td className="p-3 whitespace-nowrap">
                  {t.type === 'deposit' && (t.status === 'pending' || t.status === 'processing') && (
                    <Button
                      size="sm"
                      variant="outline"
                      className="h-8"
                      disabled={busyId === t.id}
                      onClick={() => check(t.id)}
                    >
                      {busyId === t.id ? '…' : 'Vérifier'}
                    </Button>
                  )}
                  {checkMsg[t.id] && (
                    <span className="ml-2 text-[11px] text-green-900/60">{checkMsg[t.id]}</span>
                  )}
                </td>
              </tr>
            ))}
            {!loading && rows.length === 0 && (
              <tr>
                <td colSpan={8} className="p-4 text-green-900/40 italic">
                  Aucune transaction.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
