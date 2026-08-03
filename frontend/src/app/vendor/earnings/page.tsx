'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';

interface Payout {
  id: string;
  amount_cfa: number;
  status: string;
  requested_at: string;
}

export default function VendorEarningsPage() {
  const [totalEarned, setTotalEarned] = useState(0);
  const [available, setAvailable] = useState(0);
  const [history, setHistory] = useState<Payout[]>([]);
  const [amount, setAmount] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');

  useEffect(() => {
    loadEarnings();
  }, []);

  const loadEarnings = async () => {
    setLoading(true);
    try {
      const result = await api.getVendorEarnings();
      setTotalEarned(result.total_earned);
      setAvailable(result.available);
      setHistory(result.history);
    } catch (err: any) {
      setError(err.message || 'Impossible de charger les revenus');
    } finally {
      setLoading(false);
    }
  };

  const handlePayout = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setMessage('');

    const amountNum = parseInt(amount, 10);
    if (isNaN(amountNum) || amountNum <= 0) {
      setError('Montant invalide');
      return;
    }

    try {
      await api.requestPayout(amountNum);
      setMessage('Demande de versement envoyée !');
      setAmount('');
      loadEarnings();
    } catch (err: any) {
      setError(err.message || 'Demande de versement échouée');
    }
  };

  const formatPrice = (price: number) => `${price.toLocaleString()} FCFA`;

  if (loading) return <main className="p-8 text-center">Chargement...</main>;

  return (
    <main className="min-h-screen p-8 max-w-4xl mx-auto">
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Mes revenus</h1>
          <p className="text-green-700">Vos gains de vente et versements</p>
        </div>
        <Link href="/vendor/products" className="px-4 py-2 border rounded hover:bg-green-50">
          Mes produits
        </Link>
      </header>

      {error && <div className="mb-4 p-3 bg-red-50 text-red-600 rounded">{error}</div>}
      {message && <div className="mb-4 p-3 bg-green-50 text-green-700 rounded">{message}</div>}

      <div className="grid gap-4 md:grid-cols-3 mb-8">
        <div className="p-6 border rounded-lg bg-green-50">
          <p className="text-sm text-green-700">Total gagné</p>
          <p className="text-2xl font-bold">{formatPrice(totalEarned)}</p>
        </div>
        <div className="p-6 border rounded-lg bg-green-50">
          <p className="text-sm text-green-700">Disponible</p>
          <p className="text-2xl font-bold">{formatPrice(available)}</p>
        </div>
        <div className="p-6 border rounded-lg">
          <p className="text-sm text-green-700">Versements effectués</p>
          <p className="text-2xl font-bold">{history.length}</p>
        </div>
      </div>

      <section className="mb-8 p-6 border rounded-lg">
        <h2 className="font-semibold text-lg mb-4">Demander un versement</h2>
        <form onSubmit={handlePayout} className="flex gap-4">
          <input
            type="number"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            placeholder="Montant en FCFA"
            className="flex-1 p-2 border rounded"
            min={1}
            max={available}
          />
          <button
            type="submit"
            disabled={available <= 0}
            className="px-6 py-2 gradient-green text-white rounded hover:opacity-95 disabled:opacity-50"
          >
            Demander
          </button>
        </form>
        <p className="mt-2 text-xs text-green-900/50">
          Commission plateforme : 15 % sur chaque vente · Versements traités sous 48h
        </p>
      </section>

      <section>
        <h2 className="font-semibold text-lg mb-4">Historique des versements</h2>
        {history.length === 0 ? (
          <p className="text-green-900/50">Aucun versement pour le moment.</p>
        ) : (
          <table className="w-full border rounded">
            <thead className="bg-green-50">
              <tr>
                <th className="p-3 text-left">Montant</th>
                <th className="p-3 text-left">Date</th>
                <th className="p-3 text-left">Statut</th>
              </tr>
            </thead>
            <tbody>
              {history.map((payout) => (
                <tr key={payout.id} className="border-t">
                  <td className="p-3">{formatPrice(payout.amount_cfa)}</td>
                  <td className="p-3">
                    {new Date(payout.requested_at).toLocaleDateString('fr-FR')}
                  </td>
                  <td className="p-3">
                    <span className={`px-2 py-1 rounded text-xs ${
                      payout.status === 'paid'
                        ? 'bg-green-100 text-green-700'
                        : payout.status === 'failed'
                          ? 'bg-red-100 text-red-700'
                          : 'bg-yellow-100 text-yellow-700'
                    }`}>
                      {payout.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </main>
  );
}
