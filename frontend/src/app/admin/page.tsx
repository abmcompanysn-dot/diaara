'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';

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

  if (loading) return <main className="p-8 text-center">Chargement...</main>;

  return (
    <main className="min-h-screen p-8 max-w-5xl mx-auto">
      <header className="mb-8">
        <h1 className="text-3xl font-bold">Administration</h1>
        <p className="text-green-700">Tableau de bord global</p>
      </header>

      {error && <div className="mb-4 p-3 bg-red-50 text-red-600 rounded">{error}</div>}

      <div className="grid gap-4 md:grid-cols-4 mb-8">
        <div className="p-6 border rounded-lg bg-green-50">
          <p className="text-sm text-green-700">Ventes</p>
          <p className="text-2xl font-bold">{stats.total_sales}</p>
        </div>
        <div className="p-6 border rounded-lg">
          <p className="text-sm text-green-700">Produits</p>
          <p className="text-2xl font-bold">{stats.total_products}</p>
        </div>
        <div className="p-6 border rounded-lg bg-green-50">
          <p className="text-sm text-green-700">Revenus plateforme (15%)</p>
          <p className="text-2xl font-bold">{stats.total_revenue.toLocaleString()} FCFA</p>
        </div>
        <Link href="/admin/products" className="p-6 border rounded-lg bg-yellow-50 hover:shadow">
          <p className="text-sm text-green-700">En attente de modération</p>
          <p className="text-2xl font-bold">{stats.pending_moderation}</p>
        </Link>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Link href="/admin/products" className="p-6 border rounded-lg hover:bg-green-50">
          <h2 className="font-semibold text-lg mb-2">Modération des produits</h2>
          <p className="text-green-700">Valider ou refuser les produits en attente</p>
        </Link>
        <Link href="/admin/users" className="p-6 border rounded-lg hover:bg-green-50">
          <h2 className="font-semibold text-lg mb-2">Gestion des utilisateurs</h2>
          <p className="text-green-700">Suspendre des comptes</p>
        </Link>
      </div>
    </main>
  );
}
