'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { friendlyError } from '@/lib/error-messages';
import { ArrowLeftIcon } from '@/components/icons';

interface ActivityLogEntry {
  id: string;
  admin_email?: string;
  action: string;
  target_type: string;
  target_id?: string;
  description: string;
  created_at: string;
}

// Couleur du badge par famille d'action — juste pour repérer vite le type
// d'action en scannant la liste (remboursement/suppression en rouge,
// modération en orange, le reste neutre).
function actionBadgeClass(action: string): string {
  if (action.includes('refund') || action.includes('deleted') || action.includes('suspended') || action.includes('rejected')) {
    return 'bg-red-50 text-red-700 border-red-200';
  }
  if (action.includes('admin_') || action.includes('permission')) {
    return 'bg-amber-50 text-amber-700 border-amber-200';
  }
  return 'bg-green-50 text-green-700 border-green-200';
}

export default function AdminActivityLogPage() {
  const [logs, setLogs] = useState<ActivityLogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const perPage = 30;

  useEffect(() => {
    load(page);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

  async function load(p: number) {
    setLoading(true);
    setError('');
    try {
      const res = await api.getActivityLog(p);
      setLogs(res.logs);
      setTotal(res.total);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / perPage));

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Journal d'activité"
        description="Historique des actions effectuées par les administrateurs — qui a fait quoi, et quand."
        actions={
          <Button variant="outline" size="sm" render={<Link href="/admin" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Tableau de bord
          </Button>
        }
      />

      <section className="max-w-3xl mx-auto px-4 sm:px-6 py-10 space-y-4">
        {error && (
          <div className="p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {loading ? (
          <PageLoader />
        ) : logs.length === 0 ? (
          <EmptyState title="Aucune action enregistrée" description="Le journal se remplit au fur et à mesure des actions admin." />
        ) : (
          <div className="space-y-2">
            {logs.map((log) => (
              <div
                key={log.id}
                className="flex items-start justify-between gap-3 p-4 bg-white rounded-xl shadow-card border border-green-900/5"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2 flex-wrap mb-1">
                    <Badge variant="outline" className={actionBadgeClass(log.action)}>
                      {log.action}
                    </Badge>
                    <span className="text-xs text-green-900/50">
                      {log.admin_email || 'Système (automatique)'}
                    </span>
                  </div>
                  <p className="text-sm text-green-950">{log.description}</p>
                </div>
                <span className="text-xs text-green-900/40 font-mono shrink-0 whitespace-nowrap">
                  {new Date(log.created_at).toLocaleString('fr-FR')}
                </span>
              </div>
            ))}
          </div>
        )}

        {!loading && totalPages > 1 && (
          <div className="flex justify-center items-center gap-3 pt-4">
            <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
              Précédent
            </Button>
            <span className="text-sm text-green-900/60">
              Page {page} / {totalPages}
            </span>
            <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
              Suivant
            </Button>
          </div>
        )}
      </section>
    </main>
  );
}
