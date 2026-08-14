'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { SalesChart } from '@/components/sales-chart';
import { AdminNotificationBell } from '@/components/admin-notification-bell';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { friendlyError } from '@/lib/error-messages';
import { PAYOUT_STATUS_BADGE, PAYOUT_STATUS_LABELS } from '@/lib/constants';

interface Stats {
  total_sales: number;
  total_products: number;
  pending_moderation: number;
  active_products: number;
  total_users: number;
  total_vendors: number;
  total_closers: number;
  total_admins: number;
  total_revenue: number;
  gmv: number;
  revenue_this_month: number;
  revenue_last_month: number;
  revenue_growth_pct: number;
}

const EMPTY_STATS: Stats = {
  total_sales: 0,
  total_products: 0,
  pending_moderation: 0,
  active_products: 0,
  total_users: 0,
  total_vendors: 0,
  total_closers: 0,
  total_admins: 0,
  total_revenue: 0,
  gmv: 0,
  revenue_this_month: 0,
  revenue_last_month: 0,
  revenue_growth_pct: 0,
};

const PERIODS = [
  { label: '7 jours', days: 7 as const },
  { label: 'Ce mois', days: 30 as const },
  { label: 'Cette année', days: 365 as const },
];

function money(n: number) {
  return `${n.toLocaleString('fr-FR')} FCFA`;
}

export default function AdminDashboardPage() {
  const [stats, setStats] = useState<Stats>(EMPTY_STATS);
  const [chartData, setChartData] = useState<{ day: string; sales: number; revenue_cfa: number }[]>([]);
  const [period, setPeriod] = useState<7 | 30 | 365>(30);
  const [payouts, setPayouts] = useState<any[]>([]);
  const [activity, setActivity] = useState<{ kind: string; id: string; at: string; data: any }[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [retryTarget, setRetryTarget] = useState<string | null>(null);

  useEffect(() => {
    loadAll();
  }, []);

  useEffect(() => {
    api
      .getAnalytics(period)
      .then((res) => setChartData(res.sales_by_day))
      .catch(() => {});
  }, [period]);

  const loadAll = async () => {
    setLoading(true);
    try {
      const [statsRes, payoutsRes, activityRes] = await Promise.all([
        api.getStats(),
        api.getAdminPayouts(),
        api.getActivityFeed(),
      ]);
      setStats(statsRes);
      setPayouts(payoutsRes.payouts.slice(0, 6));
      setActivity(activityRes.activity);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  };

  const handleRetry = async (id: string) => {
    setRetryTarget(null);
    try {
      await api.retryPayout(id);
      loadAll();
    } catch (err: any) {
      setError(friendlyError(err));
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// administration" title="Tableau de bord global" />
        <PageLoader />
      </main>
    );

  const growthPositive = stats.revenue_growth_pct >= 0;

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Tableau de bord global"
        description="Vue d'ensemble de la plateforme, finances et modération"
        actions={
          <>
            <AdminNotificationBell />
            <Button variant="outline" size="sm" render={<Link href="/admin/settings" />}>
              Paramètres
            </Button>
          </>
        }
      />

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-10">
        {error && (
          <div className="mb-6 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {/* KPIs — libellés en text-xs et min-h harmonisée pour que les cartes restent
            alignées même quand un titre passe sur deux lignes sur petit écran. */}
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4 mb-8">
          <div className="min-h-[100px] p-6 border rounded-xl bg-white shadow-card border-green-900/5">
            <p className="text-xs text-green-700">Ventes totales</p>
            <p className="text-3xl font-bold font-mono mt-1">{stats.total_sales}</p>
          </div>

          <Link
            href="/admin/products"
            className="min-h-[100px] p-6 border rounded-xl bg-white shadow-card border-green-900/5 hover:shadow-lift transition-all"
          >
            <p className="text-xs text-green-700">Produits actifs / en attente</p>
            <p className="text-3xl font-bold font-mono mt-1">
              {stats.active_products}{' '}
              <span className="text-base font-normal text-yellow-700">/ {stats.pending_moderation}</span>
            </p>
          </Link>

          <div className="min-h-[100px] p-6 border rounded-xl bg-green-50/60 border-green-900/10">
            <p className="text-xs text-green-700">Revenus plateforme (ce mois)</p>
            <p className="text-3xl font-bold font-mono mt-1">{money(stats.revenue_this_month)}</p>
            <p className={`text-xs font-mono mt-1 ${growthPositive ? 'text-green-600' : 'text-red-600'}`}>
              {growthPositive ? '▲' : '▼'} {Math.abs(stats.revenue_growth_pct).toFixed(1)}% vs mois dernier
            </p>
          </div>

          <div className="min-h-[100px] p-6 border rounded-xl bg-white shadow-card border-green-900/5">
            <p className="text-xs text-green-700">Volume total (GMV)</p>
            <p className="text-3xl font-bold font-mono mt-1">{money(stats.gmv)}</p>
          </div>
        </div>

        <div className="grid gap-4 sm:grid-cols-3 mb-8">
          <Link
            href="/admin/users"
            className="min-h-[84px] p-4 border rounded-xl bg-white shadow-card border-green-900/5 hover:shadow-lift transition-all"
          >
            <p className="text-xs text-green-700">Utilisateurs</p>
            <p className="text-xl font-bold font-mono mt-1">{stats.total_users}</p>
          </Link>
          <div className="min-h-[84px] p-4 border rounded-xl bg-white shadow-card border-green-900/5">
            <p className="text-xs text-green-700">Vendeurs</p>
            <p className="text-xl font-bold font-mono mt-1">{stats.total_vendors}</p>
          </div>
          <div className="min-h-[84px] p-4 border rounded-xl bg-white shadow-card border-green-900/5">
            <p className="text-xs text-green-700">Affiliés</p>
            <p className="text-xl font-bold font-mono mt-1">{stats.total_closers}</p>
          </div>
        </div>

        {/* Graphique + Payouts */}
        <div className="grid gap-6 lg:grid-cols-3 mb-8">
          <div className="lg:col-span-2 p-6 border rounded-xl bg-white shadow-card border-green-900/5">
            <div className="flex items-center justify-between mb-4 flex-wrap gap-3">
              <h2 className="font-display font-bold text-lg text-green-950">Évolution des ventes</h2>
              <div className="flex gap-1 bg-secondary/60 rounded-lg p-1">
                {PERIODS.map((p) => (
                  <button
                    key={p.days}
                    onClick={() => setPeriod(p.days)}
                    className={`px-2.5 py-1 rounded-md text-xs font-medium transition-colors ${
                      period === p.days ? 'bg-white shadow-sm text-green-950' : 'text-green-900/50'
                    }`}
                  >
                    {p.label}
                  </button>
                ))}
              </div>
            </div>
            <SalesChart data={chartData} />
          </div>

          <div className="p-6 border rounded-xl bg-white shadow-card border-green-900/5">
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-display font-bold text-lg text-green-950">Versements récents</h2>
              <Link href="/admin/sales" className="text-xs text-primary font-medium">
                Voir tout
              </Link>
            </div>
            {payouts.length === 0 ? (
              <p className="text-sm text-green-900/50">Aucune demande de versement.</p>
            ) : (
              <div className="space-y-3">
                {payouts.map((p) => (
                  <div key={p.id} className="flex items-center justify-between gap-2 text-sm">
                    <div className="min-w-0">
                      <p className="truncate font-medium text-green-950">{p.user_email}</p>
                      <p className="font-mono text-xs text-green-900/50">{money(p.amount_cfa)}</p>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      <Badge className={PAYOUT_STATUS_BADGE[p.status]}>
                        {PAYOUT_STATUS_LABELS[p.status] || p.status}
                      </Badge>
                      {p.status === 'failed' && (
                        <Button variant="ghost" className="h-9 px-3 text-xs" onClick={() => setRetryTarget(p.id)}>
                          Relancer
                        </Button>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Modération + Activité */}
        <div className="grid gap-6 lg:grid-cols-3">
          <div className="p-6 border rounded-xl bg-white shadow-card border-green-900/5">
            <h2 className="font-display font-bold text-lg text-green-950 mb-4">Actions rapides</h2>
            <div className="space-y-2">
              <Link
                href="/admin/products"
                className="flex items-center justify-between p-3 rounded-lg hover:bg-green-900/5 transition-colors group"
              >
                <span className="text-sm font-medium text-green-950">Modération des produits</span>
                {stats.pending_moderation > 0 && (
                  <Badge className="bg-yellow-100 text-yellow-800 hover:bg-yellow-100">
                    {stats.pending_moderation}
                  </Badge>
                )}
              </Link>
              <Link
                href="/admin/users"
                className="flex items-center justify-between p-3 rounded-lg hover:bg-green-900/5 transition-colors"
              >
                <span className="text-sm font-medium text-green-950">Gestion des utilisateurs</span>
              </Link>
              <Link
                href="/admin/sales"
                className="flex items-center justify-between p-3 rounded-lg hover:bg-green-900/5 transition-colors"
              >
                <span className="text-sm font-medium text-green-950">Historique des ventes</span>
              </Link>
              <Link
                href="/admin/settings"
                className="flex items-center justify-between p-3 rounded-lg hover:bg-green-900/5 transition-colors"
              >
                <span className="text-sm font-medium text-green-950">Commission & passerelles</span>
              </Link>
              <Link
                href="/admin/permissions"
                className="flex items-center justify-between p-3 rounded-lg hover:bg-green-900/5 transition-colors"
              >
                <span className="text-sm font-medium text-green-950">Droits admin</span>
              </Link>
              <Link
                href="/admin/automation"
                className="flex items-center justify-between p-3 rounded-lg hover:bg-green-900/5 transition-colors"
              >
                <span className="text-sm font-medium text-green-950">Automatisation</span>
              </Link>
              <Link
                href="/admin/support"
                className="flex items-center justify-between p-3 rounded-lg hover:bg-green-900/5 transition-colors"
              >
                <span className="text-sm font-medium text-green-950">Support (tickets en direct)</span>
              </Link>
            </div>
          </div>

          <div className="lg:col-span-2 p-6 border rounded-xl bg-white shadow-card border-green-900/5">
            <h2 className="font-display font-bold text-lg text-green-950 mb-4">Activité récente</h2>
            {activity.length === 0 ? (
              <p className="text-sm text-green-900/50">Aucune activité récente.</p>
            ) : (
              <ul className="space-y-3">
                {activity.map((item) => (
                  <li key={`${item.kind}-${item.id}`} className="flex items-start gap-3 text-sm">
                    <span
                      className={`mt-1.5 w-1.5 h-1.5 rounded-full shrink-0 ${
                        item.kind === 'sale'
                          ? 'bg-green-500'
                          : item.kind === 'payout'
                            ? 'bg-blue-500'
                            : 'bg-purple-500'
                      }`}
                      aria-hidden
                    />
                    <div className="min-w-0 flex-1">
                      {item.kind === 'sale' && (
                        <p className="text-green-900/80">
                          Vente <span className="font-medium">{item.data.product_title}</span> —{' '}
                          {money(item.data.amount_cfa)}{' '}
                          <span className="text-xs text-green-900/40">({item.data.status})</span>
                        </p>
                      )}
                      {item.kind === 'user' && (
                        <p className="text-green-900/80">
                          Nouvel utilisateur inscrit — <span className="font-medium">{item.data.email}</span>
                        </p>
                      )}
                      {item.kind === 'payout' && (
                        <p className="text-green-900/80">
                          Demande de versement de <span className="font-medium">{item.data.email}</span> —{' '}
                          {money(item.data.amount_cfa)}{' '}
                          <span className="text-xs text-green-900/40">({item.data.status})</span>
                        </p>
                      )}
                      <p className="text-xs text-green-900/40 font-mono">
                        {new Date(item.at).toLocaleString('fr-FR', {
                          day: '2-digit',
                          month: '2-digit',
                          hour: '2-digit',
                          minute: '2-digit',
                        })}
                      </p>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      </section>

      <ConfirmDialog
        open={!!retryTarget}
        title="Relancer ce versement ?"
        description={(() => {
          const p = payouts.find((x) => x.id === retryTarget);
          return p ? `Un nouveau versement de ${money(p.amount_cfa)} vers ${p.user_email} sera envoyé via PawaPay.` : undefined;
        })()}
        confirmLabel="Relancer"
        cancelLabel="Annuler"
        onConfirm={() => retryTarget && handleRetry(retryTarget)}
        onCancel={() => setRetryTarget(null)}
      />
    </main>
  );
}
