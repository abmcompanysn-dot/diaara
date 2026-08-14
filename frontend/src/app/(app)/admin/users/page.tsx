'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { ROLE_LABELS, formatPrice } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';
import { SearchIcon, MoreVerticalIcon, ArrowLeftIcon } from '@/components/icons';

interface User {
  id: string;
  email: string;
  is_admin: boolean;
  roles: string[];
  locked_until: string | null;
  created_at: string;
  products_sold: number;
  revenue_generated_cfa: number;
}

type RoleFilter = 'all' | 'client' | 'vendeur' | 'closer' | 'admin';
type StatusFilter = 'all' | 'active' | 'suspended';
type SortBy = 'date' | 'revenue';
const PAGE_SIZES = [10, 20, 50] as const;

function isSuspended(u: User) {
  return !!u.locked_until && new Date(u.locked_until) > new Date();
}

function initials(email: string) {
  return email.slice(0, 2).toUpperCase();
}

function RowMenu({
  user,
  onRole,
  onSuspend,
  onReactivate,
}: {
  user: User;
  onRole: (role: string, action: 'grant' | 'revoke') => void;
  onSuspend: () => void;
  onReactivate: () => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onClick);
    return () => document.removeEventListener('mousedown', onClick);
  }, []);

  const suspended = isSuspended(user);

  return (
    <div className="relative inline-block" ref={ref}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-label="Actions"
        className="w-9 h-9 inline-flex items-center justify-center rounded-lg hover:bg-green-900/5 text-green-900/60"
      >
        <MoreVerticalIcon size={18} />
      </button>
      {open && (
        <div className="absolute right-0 mt-1 w-56 bg-white rounded-xl shadow-lift border border-green-900/10 py-1.5 z-20 text-sm">
          {!user.is_admin && (
            <>
              <p className="px-3 pt-1 pb-1 text-[10px] uppercase tracking-widest text-green-900/40 font-mono">
                Rôles
              </p>
              {Object.keys(ROLE_LABELS).map((role) => (
                <button
                  key={role}
                  onClick={() => {
                    onRole(role, user.roles?.includes(role) ? 'revoke' : 'grant');
                    setOpen(false);
                  }}
                  className="w-full text-left px-3 py-2 hover:bg-green-900/5 text-green-950"
                >
                  {user.roles?.includes(role) ? `Retirer ${ROLE_LABELS[role]}` : `Accorder ${ROLE_LABELS[role]}`}
                </button>
              ))}
              <div className="my-1 border-t border-green-900/10" />
              {suspended ? (
                <button
                  onClick={() => {
                    onReactivate();
                    setOpen(false);
                  }}
                  className="w-full text-left px-3 py-2 hover:bg-green-900/5 text-green-700"
                >
                  Réactiver le compte
                </button>
              ) : (
                <button
                  onClick={() => {
                    onSuspend();
                    setOpen(false);
                  }}
                  className="w-full text-left px-3 py-2 hover:bg-destructive/10 text-destructive"
                >
                  Suspendre le compte
                </button>
              )}
            </>
          )}
          {user.is_admin && (
            <p className="px-3 py-2 text-green-900/40 text-xs">Compte administrateur</p>
          )}
        </div>
      )}
    </div>
  );
}

export default function AdminUsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [toSuspend, setToSuspend] = useState<User | null>(null);

  const [search, setSearch] = useState('');
  const [roleFilter, setRoleFilter] = useState<RoleFilter>('all');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [sortBy, setSortBy] = useState<SortBy>('date');
  const [pageSize, setPageSize] = useState<(typeof PAGE_SIZES)[number]>(20);
  const [page, setPage] = useState(1);

  useEffect(() => {
    loadUsers();
  }, []);

  useEffect(() => {
    setPage(1);
  }, [search, roleFilter, statusFilter, sortBy]);

  const loadUsers = async () => {
    setLoading(true);
    try {
      const result = await api.getUsers();
      setUsers(result.users);
    } catch (err: any) {
      setError(friendlyError(err));
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
      setError(friendlyError(err));
    } finally {
      setToSuspend(null);
    }
  };

  const handleReactivate = async (id: string) => {
    try {
      await api.reactivateUser(id);
      loadUsers();
    } catch (err: any) {
      setError(friendlyError(err));
    }
  };

  const handleSetRole = async (id: string, role: string, action: 'grant' | 'revoke') => {
    try {
      await api.setRole(id, role, action);
      loadUsers();
    } catch (err: any) {
      setError(friendlyError(err));
    }
  };

  const filtered = useMemo(() => {
    let list = users;

    if (search.trim()) {
      const q = search.trim().toLowerCase();
      list = list.filter((u) => u.email.toLowerCase().includes(q));
    }

    if (roleFilter !== 'all') {
      list = list.filter((u) => {
        if (roleFilter === 'admin') return u.is_admin;
        if (roleFilter === 'client') return !u.is_admin && (!u.roles || u.roles.length === 0);
        return u.roles?.includes(roleFilter);
      });
    }

    if (statusFilter !== 'all') {
      list = list.filter((u) => (statusFilter === 'suspended' ? isSuspended(u) : !isSuspended(u)));
    }

    list = [...list].sort((a, b) =>
      sortBy === 'revenue'
        ? b.revenue_generated_cfa - a.revenue_generated_cfa
        : new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    );

    return list;
  }, [users, search, roleFilter, statusFilter, sortBy]);

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize));
  const pageItems = filtered.slice((page - 1) * pageSize, page * pageSize);

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
            <ArrowLeftIcon size={16} className="mr-2" />
            Dashboard
          </Button>
        }
      />

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-10">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {/* Recherche + filtres */}
        <div className="flex flex-wrap gap-3 mb-5">
          <div className="relative flex-1 min-w-55">
            <SearchIcon size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-green-900/40" />
            <Input
              placeholder="Rechercher par email..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9 bg-white"
            />
          </div>
          <Select value={roleFilter} onValueChange={(v) => setRoleFilter((v as RoleFilter) || 'all')}>
            <SelectTrigger className="w-40 bg-white">
              <SelectValue placeholder="Rôle" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Tous les rôles</SelectItem>
              <SelectItem value="client">Clients</SelectItem>
              <SelectItem value="vendeur">Vendeurs</SelectItem>
              <SelectItem value="closer">Affiliés</SelectItem>
              <SelectItem value="admin">Admins</SelectItem>
            </SelectContent>
          </Select>
          <Select value={statusFilter} onValueChange={(v) => setStatusFilter((v as StatusFilter) || 'all')}>
            <SelectTrigger className="w-37.5 bg-white">
              <SelectValue placeholder="Statut" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Tous les statuts</SelectItem>
              <SelectItem value="active">Actif</SelectItem>
              <SelectItem value="suspended">Suspendu</SelectItem>
            </SelectContent>
          </Select>
          <Select value={sortBy} onValueChange={(v) => setSortBy((v as SortBy) || 'date')}>
            <SelectTrigger className="w-47.5 bg-white">
              <SelectValue placeholder="Trier par" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="date">Date d&apos;inscription</SelectItem>
              <SelectItem value="revenue">Revenu généré</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {filtered.length === 0 ? (
          <EmptyState title="Aucun utilisateur" description="Aucun compte ne correspond à ces filtres." />
        ) : (
          <>
            <div className="rounded-xl border border-green-900/10 bg-white shadow-card overflow-hidden overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Utilisateur</TableHead>
                    <TableHead>Inscrit le</TableHead>
                    <TableHead>Rôles</TableHead>
                    <TableHead>Performance</TableHead>
                    <TableHead>Statut</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {pageItems.map((user) => {
                    const suspended = isSuspended(user);
                    return (
                      <TableRow key={user.id}>
                        <TableCell>
                          <div className="flex items-center gap-3">
                            <span className="w-8 h-8 rounded-full bg-green-900/10 text-green-900/70 text-xs font-bold flex items-center justify-center shrink-0">
                              {initials(user.email)}
                            </span>
                            <span className="truncate max-w-55">{user.email}</span>
                          </div>
                        </TableCell>
                        <TableCell className="text-sm">
                          {new Date(user.created_at).toLocaleDateString('fr-FR')}
                        </TableCell>
                        <TableCell>
                          {user.is_admin ? (
                            <Badge className="bg-purple-100 text-purple-700 hover:bg-purple-100">Admin</Badge>
                          ) : (
                            <div className="flex flex-wrap gap-1">
                              {user.roles?.map((role) => (
                                <Badge key={role}>{ROLE_LABELS[role] || role}</Badge>
                              ))}
                              {(!user.roles || user.roles.length === 0) && (
                                <Badge variant="outline">Client</Badge>
                              )}
                            </div>
                          )}
                        </TableCell>
                        <TableCell className="text-sm">
                          {user.products_sold > 0 ? (
                            <>
                              <p className="font-medium">{user.products_sold} vente(s)</p>
                              <p className="text-xs text-green-900/50 font-mono">
                                {formatPrice(user.revenue_generated_cfa)}
                              </p>
                            </>
                          ) : (
                            <span className="text-green-900/30">—</span>
                          )}
                        </TableCell>
                        <TableCell>
                          <span className="inline-flex items-center gap-1.5">
                            <span
                              className={`w-2 h-2 rounded-full ${suspended ? 'bg-red-500' : 'bg-green-500'}`}
                              aria-hidden
                            />
                            <span className="text-sm">{suspended ? 'Suspendu' : 'Actif'}</span>
                          </span>
                        </TableCell>
                        <TableCell className="text-right">
                          <RowMenu
                            user={user}
                            onRole={(role, action) => handleSetRole(user.id, role, action)}
                            onSuspend={() => setToSuspend(user)}
                            onReactivate={() => handleReactivate(user.id)}
                          />
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>

            {/* Pagination */}
            <div className="flex flex-wrap items-center justify-between gap-3 mt-4">
              <div className="flex items-center gap-2 text-sm text-green-900/60">
                <span>Afficher</span>
                <Select
                  value={String(pageSize)}
                  onValueChange={(v) => {
                    setPageSize(Number(v) as (typeof PAGE_SIZES)[number]);
                    setPage(1);
                  }}
                >
                  <SelectTrigger className="w-20 bg-white h-8">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {PAGE_SIZES.map((n) => (
                      <SelectItem key={n} value={String(n)}>
                        {n}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <span>par page &middot; {filtered.length} résultat(s)</span>
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                >
                  Précédent
                </Button>
                <span className="text-sm text-green-900/60 font-mono">
                  {page} / {totalPages}
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= totalPages}
                  onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                >
                  Suivant
                </Button>
              </div>
            </div>
          </>
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
