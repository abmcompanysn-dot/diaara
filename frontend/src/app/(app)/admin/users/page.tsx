'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { ROLE_LABELS } from '@/lib/constants';

interface User {
  id: string;
  email: string;
  is_admin: boolean;
  roles: string[];
  locked_until: string | null;
  created_at: string;
}

export default function AdminUsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [toSuspend, setToSuspend] = useState<User | null>(null);

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

  const handleSuspend = async () => {
    if (!toSuspend) return;
    try {
      await api.suspendUser(toSuspend.id);
      loadUsers();
    } catch (err: any) {
      setError(err.message || 'Action échouée');
    } finally {
      setToSuspend(null);
    }
  };

  const handleSetRole = async (id: string, role: string, action: 'grant' | 'revoke') => {
    try {
      await api.setRole(id, role, action);
      loadUsers();
    } catch (err: any) {
      setError(err.message || 'Action échouée');
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// administration" title="Utilisateurs" description="Gestion des comptes et des rôles" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Utilisateurs"
        description={`${users.length} compte(s) enregistré(s)`}
        actions={
          <Button variant="outline" size="sm" render={<Link href="/admin" />}>
            ← Dashboard
          </Button>
        }
      />

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-10">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {users.length === 0 ? (
          <EmptyState
            title="Aucun utilisateur"
            description="Aucun compte n'a encore été créé."
          />
        ) : (
          <div className="rounded-xl border border-green-900/10 bg-white shadow-card overflow-hidden">
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
                    <TableCell className="text-sm">{new Date(user.created_at).toLocaleDateString('fr-FR')}</TableCell>
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
                        <span className="flex justify-end gap-2 flex-wrap">
                          {Object.keys(ROLE_LABELS).map((role) =>
                            user.roles?.includes(role) ? (
                              <Button
                                key={role}
                                variant="outline"
                                size="sm"
                                className="min-h-9"
                                onClick={() => handleSetRole(user.id, role, 'revoke')}
                                title={`Retirer le rôle ${ROLE_LABELS[role]}`}
                              >
                                - {ROLE_LABELS[role]}
                              </Button>
                            ) : (
                              <Button
                                key={role}
                                size="sm"
                                className="min-h-9"
                                onClick={() => handleSetRole(user.id, role, 'grant')}
                                title={`Ajouter le rôle ${ROLE_LABELS[role]}`}
                              >
                                + {ROLE_LABELS[role]}
                              </Button>
                            )
                          )}
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-destructive hover:text-destructive hover:bg-destructive/10 min-h-9"
                            onClick={() => setToSuspend(user)}
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
        )}
      </section>

      <ConfirmDialog
        open={!!toSuspend}
        title="Suspendre ce compte ?"
        description={`Le compte de ${toSuspend?.email} sera suspendu pendant 30 jours. L'utilisateur ne pourra plus se connecter.`}
        confirmLabel="Suspendre"
        cancelLabel="Annuler"
        danger
        onConfirm={handleSuspend}
        onCancel={() => setToSuspend(null)}
      />
    </main>
  );
}
