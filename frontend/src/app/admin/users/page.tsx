'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';

interface User {
  id: string;
  email: string;
  is_admin: boolean;
  locked_until: string | null;
  created_at: string;
}

export default function AdminUsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    loadUsers();
  }, []);

  const loadUsers = async () => {
    setLoading(true);
    try {
      const result = await api.getUsers();
      setUsers(result.users);
    } catch (err: any) {
      setError(err.message || 'Impossible de charger les utilisateurs');
    } finally {
      setLoading(false);
    }
  };

  const handleSuspend = async (id: string) => {
    if (!confirm('Suspendre ce compte pour 30 jours ?')) return;
    try {
      await api.suspendUser(id);
      loadUsers();
    } catch (err: any) {
      alert(err.message || 'Action échouée');
    }
  };

  if (loading) return <main className="p-8 text-center">Chargement...</main>;

  return (
    <main className="min-h-screen p-8 max-w-5xl mx-auto">
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Utilisateurs</h1>
          <p className="text-green-700">{users.length} compte(s)</p>
        </div>
        <Link href="/admin" className="text-green-600">← Dashboard</Link>
      </header>

      {error && <div className="mb-4 p-3 bg-red-50 text-red-600 rounded">{error}</div>}

      <div className="overflow-x-auto">
        <table className="w-full border rounded">
          <thead className="bg-green-50">
            <tr>
              <th className="p-3 text-left">Email</th>
              <th className="p-3 text-left">Inscrit le</th>
              <th className="p-3 text-left">Rôle</th>
              <th className="p-3 text-left">Statut</th>
              <th className="p-3 text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            {users.map((user) => (
              <tr key={user.id} className="border-t">
                <td className="p-3">{user.email}</td>
                <td className="p-3">{new Date(user.created_at).toLocaleDateString('fr-FR')}</td>
                <td className="p-3">
                  {user.is_admin ? (
                    <span className="px-2 py-1 rounded text-xs bg-purple-100 text-purple-700">Admin</span>
                  ) : (
                    <span className="px-2 py-1 rounded text-xs bg-green-100">Client</span>
                  )}
                </td>
                <td className="p-3">
                  {user.locked_until ? (
                    <span className="px-2 py-1 rounded text-xs bg-red-100 text-red-700">Suspendu</span>
                  ) : (
                    <span className="px-2 py-1 rounded text-xs bg-green-100 text-green-700">Actif</span>
                  )}
                </td>
                <td className="p-3 text-right">
                  {!user.is_admin && !user.locked_until && (
                    <button
                      onClick={() => handleSuspend(user.id)}
                      className="text-red-600 hover:underline text-sm"
                    >
                      Suspendre
                    </button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </main>
  );
}
