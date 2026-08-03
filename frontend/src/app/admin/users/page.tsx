'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';

interface User {
  id: string;
  email: string;
  is_admin: boolean;
  roles: string[];
  locked_until: string | null;
  created_at: string;
}

const ROLE_LABELS: Record<string, string> = {
  vendeur: 'Vendeur',
  closer: 'Affilié',
};

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

  const handleSetRole = async (id: string, role: string, action: 'grant' | 'revoke') => {
    try {
      await api.setRole(id, role, action);
      loadUsers();
    } catch (err: any) {
      alert(err.message || 'Action échouée');
    }
  };

  if (loading) return <main className="p-8 text-center">Chargement...</main>;

  return (
    <main className="min-h-screen p-8 max-w-6xl mx-auto">
      <header className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Utilisateurs</h1>
          <p className="text-green-700">{users.length} compte(s)</p>
        </div>
        <Button variant="outline" size="sm" render={<Link href="/admin" />}>
          ← Dashboard
        </Button>
      </header>

      {error && <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded">{error}</div>}

      <div className="rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Email</TableHead>
              <TableHead>Inscrit le</TableHead>
              <TableHead>Rôle</TableHead>
              <TableHead>Statut</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {users.map((user) => (
              <TableRow key={user.id}>
                <TableCell>{user.email}</TableCell>
                <TableCell>{new Date(user.created_at).toLocaleDateString('fr-FR')}</TableCell>
                <TableCell>
                  {user.is_admin ? (
                    <Badge variant="secondary" className="bg-purple-100 text-purple-700 hover:bg-purple-100">
                      Admin
                    </Badge>
                  ) : (
                    <>
                      {user.roles?.map((role) => (
                        <Badge key={role} className="mr-1">
                          {ROLE_LABELS[role] || role}
                        </Badge>
                      ))}
                      {(!user.roles || user.roles.length === 0) && (
                        <Badge variant="outline">Client</Badge>
                      )}
                    </>
                  )}
                </TableCell>
                <TableCell>
                  {user.locked_until ? (
                    <Badge variant="destructive">Suspendu</Badge>
                  ) : (
                    <Badge variant="secondary">Actif</Badge>
                  )}
                </TableCell>
                <TableCell className="text-right">
                  {!user.is_admin && !user.locked_until && (
                    <span className="flex justify-end gap-2">
                      {Object.keys(ROLE_LABELS).map((role) =>
                        user.roles?.includes(role) ? (
                          <Button
                            key={role}
                            variant="outline"
                            size="xs"
                            onClick={() => handleSetRole(user.id, role, 'revoke')}
                            title={`Retirer le rôle ${ROLE_LABELS[role]}`}
                          >
                            - {ROLE_LABELS[role]}
                          </Button>
                        ) : (
                          <Button
                            key={role}
                            size="xs"
                            onClick={() => handleSetRole(user.id, role, 'grant')}
                            title={`Ajouter le rôle ${ROLE_LABELS[role]}`}
                          >
                            + {ROLE_LABELS[role]}
                          </Button>
                        )
                      )}
                      <Button
                        variant="ghost"
                        size="xs"
                        className="text-destructive hover:text-destructive hover:bg-destructive/10"
                        onClick={() => handleSuspend(user.id)}
                      >
                        Suspendre
                      </Button>
                    </span>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </main>
  );
}
