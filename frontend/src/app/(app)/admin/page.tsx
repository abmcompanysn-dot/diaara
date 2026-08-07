'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';

export default function AdminDashboardPage() {
  const [stats, setStats] = useState({ total_sales: 0, total_products: 0, total_revenue: 0, pending_moderation: 0 });
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    loadStats();
  }, []);

  const loadStats = async () => {
    setLoading(true);
    try {
      const result = await api.getStats();
      setStats(result);
    } catch (err: any) {
      setError(err.message || 'Impossible de charger les statistiques');
    } finally {
      setLoading(false);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// administration" title="Tableau de bord global" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Tableau de bord global"
        description="Vue d'ensemble de la plateforme et modération"
      />

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-10">
        {error && (
          <div className="mb-6 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4 mb-8">
          <div className="p-6 border rounded-xl bg-green-50/60 border-green-900/10">
            <p className="text-sm text-green-700">Ventes</p>
            <p className="text-3xl font-bold font-mono mt-1">{stats.total_sales}</p>
          </div>
          <div className="p-6 border rounded-xl bg-white shadow-card border-green-900/5">
            <p className="text-sm text-green-700">Produits</p>
            <p className="text-3xl font-bold font-mono mt-1">{stats.total_products}</p>
          </div>
          <div className="p-6 border rounded-xl bg-green-50/60 border-green-900/10">
            <p className="text-sm text-green-700">Revenus plateforme (15%)</p>
            <p className="text-3xl font-bold font-mono mt-1">
              {stats.total_revenue.toLocaleString('fr-FR')} FCFA
            </p>
          </div>
          <Link
            href="/admin/products"
            className="p-6 border rounded-xl bg-yellow-50 hover:shadow-lift transition-all border-yellow-200 group"
          >
            <p className="text-sm text-yellow-800">En attente de modération</p>
            <p className="text-3xl font-bold font-mono mt-1 text-yellow-900 group-hover:text-yellow-700">
              {stats.pending_moderation}
            </p>
          </Link>
        </div>

        <div className="grid gap-6 md:grid-cols-3">
          <Link href="/admin/products" className="p-6 border rounded-xl bg-white shadow-card border-green-900/5 hover:border-green-600/30 transition-all group">
            <h2 className="font-display font-bold text-lg text-green-950 group-hover:text-green-600 transition-colors mb-2">Modération des produits</h2>
            <p className="text-green-700 text-sm">Valider ou refuser les produits en attente</p>
          </Link>
          <Link href="/admin/users" className="p-6 border rounded-xl bg-white shadow-card border-green-900/5 hover:border-green-600/30 transition-all group">
            <h2 className="font-display font-bold text-lg text-green-950 group-hover:text-green-600 transition-colors mb-2">Gestion des utilisateurs</h2>
            <p className="text-green-700 text-sm">Attribuer les rôles vendeur/affilié, suspendre des comptes</p>
          </Link>
          <Link href="/admin/sales" className="p-6 border rounded-xl bg-white shadow-card border-green-900/5 hover:border-green-600/30 transition-all group">
            <h2 className="font-display font-bold text-lg text-green-950 group-hover:text-green-600 transition-colors mb-2">Ventes</h2>
            <p className="text-green-700 text-sm">Suivre les ventes, frais et commissions d&apos;affiliation</p>
          </Link>
        </div>
      </section>
    </main>
  );
}
