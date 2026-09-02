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
  phone: string | null;
  country?: string; // ISO3 déduit de l'indicatif, "" si inconnu
  country_label?: string; // nom lisible ou "—"
  is_admin: boolean;
  roles: string[];
  locked_until: string | null;
  created_at: string;
  products_sold: number;
  revenue_generated_cfa: number;
}

interface CountryBucket {
  country: string;
  country_label: string;
  users: number;
  vendors: number;
  with_phone: number;
}

// waLink construit un lien de discussion WhatsApp direct vers un numéro.
// wa.me exige le numéro en chiffres seuls (indicatif compris, sans "+").
function waLink(phone: string | null | undefined): string | null {
  if (!phone) return null;
  const digits = phone.replace(/\D/g, '');
  return digits.length >= 8 ? `https://wa.me/${digits}` : null;
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
  onPromote,
  onMessage,
}: {
  user: User;
  onRole: (role: string, action: 'grant' | 'revoke') => void;
  onSuspend: () => void;
  onReactivate: () => void;
  onPromote: () => void;
  onMessage: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [menuPos, setMenuPos] = useState<{ top: number; left: number } | null>(null);
  const ref = useRef<HTMLDivElement>(null);
  const btnRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onClick);
    return () => document.removeEventListener('mousedown', onClick);
  }, []);

  const suspended = isSuspended(user);

  // Position en `fixed` (calculée depuis le bouton) plutôt qu'`absolute` dans
  // la ligne du tableau : le tableau est un conteneur de scroll horizontal
  // (overflow-x-auto), qui coupe tout menu positionné en absolute dès que la
  // ligne est proche du bas visible. `fixed` échappe à ce clipping.
  const toggleOpen = () => {
    if (!open && btnRef.current) {
      const rect = btnRef.current.getBoundingClientRect();
      const menuHeight = 260; // estimation, suffisant pour décider haut/bas
      const openUpward = rect.bottom + menuHeight > window.innerHeight;
      setMenuPos({
        top: openUpward ? rect.top - menuHeight : rect.bottom + 4,
        left: Math.min(rect.right - 224, window.innerWidth - 232), // 224 = largeur du menu (w-56)
      });
    }
    setOpen((v) => !v);
  };

  return (
    <div className="relative inline-block" ref={ref}>
      <button
        ref={btnRef}
        type="button"
        onClick={toggleOpen}
        aria-label="Actions"
        className="w-9 h-9 inline-flex items-center justify-center rounded-lg hover:bg-green-900/5 text-green-900/60"
      >
        <MoreVerticalIcon size={18} />
      </button>
      {open && menuPos && (
        <div
          data-row-menu
          style={{ position: 'fixed', top: menuPos.top, left: menuPos.left }}
          className="w-56 bg-white rounded-xl shadow-lift border border-green-900/10 py-1.5 z-50 text-sm"
        >
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
              <div className="my-1 border-t border-green-900/10" />
              <button
                onClick={() => {
                  onPromote();
                  setOpen(false);
                }}
                className="w-full text-left px-3 py-2 hover:bg-green-900/5 text-green-950"
              >
                Promouvoir en admin
              </button>
            </>
          )}
          <div className="my-1 border-t border-green-900/10" />
          <button
            onClick={() => {
              onMessage();
              setOpen(false);
            }}
            className="w-full text-left px-3 py-2 hover:bg-green-900/5 text-green-950"
          >
            Envoyer un message
          </button>
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
  const [toRole, setToRole] = useState<{ user: User; role: string; action: 'grant' | 'revoke' } | null>(null);
  const [toPromote, setToPromote] = useState<User | null>(null);
  const [messageTarget, setMessageTarget] = useState<User | null>(null);
  const [messageSubject, setMessageSubject] = useState('');
  const [messageBody, setMessageBody] = useState('');
  const [sendingMessage, setSendingMessage] = useState(false);
  const [messageError, setMessageError] = useState('');
  const [messageSent, setMessageSent] = useState(false);

  const [search, setSearch] = useState('');
  const [roleFilter, setRoleFilter] = useState<RoleFilter>('all');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [countryFilter, setCountryFilter] = useState<string>('all'); // 'all' | ISO3 | 'UNKNOWN'
  const [sortBy, setSortBy] = useState<SortBy>('date');
  const [pageSize, setPageSize] = useState<(typeof PAGE_SIZES)[number]>(20);
  const [page, setPage] = useState(1);

  const [countryBuckets, setCountryBuckets] = useState<CountryBucket[]>([]);

  // Message groupé par pays
  const [groupMsgOpen, setGroupMsgOpen] = useState(false);
  const [groupCountry, setGroupCountry] = useState<string>('all'); // 'all' | ISO3 | 'UNKNOWN'
  const [groupSubject, setGroupSubject] = useState('');
  const [groupHtml, setGroupHtml] = useState('');
  const [groupSending, setGroupSending] = useState(false);
  const [groupError, setGroupError] = useState('');
  const [groupResult, setGroupResult] = useState('');

  useEffect(() => {
    loadUsers();
    api
      .getUsersByCountry()
      .then((r) => setCountryBuckets(r.countries))
      .catch(() => {});
  }, []);

  useEffect(() => {
    setPage(1);
  }, [search, roleFilter, statusFilter, countryFilter, sortBy]);

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

  const handleSendGroupMessage = async () => {
    if (!groupSubject.trim() || !groupHtml.trim()) {
      setGroupError('Le sujet et le message sont requis.');
      return;
    }
    setGroupError('');
    setGroupResult('');
    setGroupSending(true);
    try {
      const payload: { subject: string; html: string; country?: string; test_only?: boolean } = {
        subject: groupSubject.trim(),
        html: groupHtml.trim(),
      };
      if (groupCountry !== 'all') payload.country = groupCountry;
      const r = await api.sendBroadcast(payload);
      setGroupResult(
        `Envoi lancé vers ${r.recipients ?? 0} destinataire(s)${
          groupCountry !== 'all' ? ` (${countryLabelOf(groupCountry)})` : ''
        }.`
      );
    } catch (err: any) {
      setGroupError(friendlyError(err));
    } finally {
      setGroupSending(false);
    }
  };

  const countryLabelOf = (code: string) => {
    if (code === 'all') return 'tous les pays';
    if (code === 'UNKNOWN') return 'pays non déterminé';
    return countryBuckets.find((b) => b.country === code)?.country_label || code;
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

  const handleSetRole = async () => {
    if (!toRole) return;
    try {
      await api.setRole(toRole.user.id, toRole.role, toRole.action);
      loadUsers();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setToRole(null);
    }
  };

  const openMessage = (user: User) => {
    setMessageTarget(user);
    setMessageSubject('');
    setMessageBody('');
    setMessageError('');
    setMessageSent(false);
  };

  const handleSendMessage = async () => {
    if (!messageTarget || !messageSubject.trim() || !messageBody.trim()) {
      setMessageError('Le sujet et le message sont requis.');
      return;
    }
    setMessageError('');
    setSendingMessage(true);
    try {
      await api.sendUserMessage(messageTarget.id, { subject: messageSubject.trim(), message: messageBody.trim() });
      setMessageSent(true);
    } catch (err: any) {
      setMessageError(friendlyError(err));
    } finally {
      setSendingMessage(false);
    }
  };

  const handlePromote = async () => {
    if (!toPromote) return;
    try {
      await api.setAdminStatus(toPromote.id, 'grant');
      loadUsers();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setToPromote(null);
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

    if (countryFilter !== 'all') {
      list = list.filter((u) =>
        countryFilter === 'UNKNOWN' ? !u.country : u.country === countryFilter
      );
    }

    list = [...list].sort((a, b) =>
      sortBy === 'revenue'
        ? b.revenue_generated_cfa - a.revenue_generated_cfa
        : new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    );

    return list;
  }, [users, search, roleFilter, statusFilter, countryFilter, sortBy]);

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
          <div className="flex items-center gap-2">
            <Button variant="outline" size="sm" onClick={() => setGroupMsgOpen(true)}>
              Message groupé par pays
            </Button>
            <Button variant="outline" size="sm" render={<Link href="/admin" />}>
              <ArrowLeftIcon size={16} className="mr-2" />
              Dashboard
            </Button>
          </div>
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
          <Select value={countryFilter} onValueChange={(v) => setCountryFilter(v || 'all')}>
            <SelectTrigger className="w-44 bg-white">
              <SelectValue placeholder="Pays" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Tous les pays</SelectItem>
              {countryBuckets.map((b) => (
                <SelectItem key={b.country || 'unknown'} value={b.country || 'UNKNOWN'}>
                  {b.country_label} ({b.users})
                </SelectItem>
              ))}
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
                    <TableHead>Téléphone / Pays</TableHead>
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
                        <TableCell className="text-sm text-green-900/70 whitespace-nowrap">
                          {user.phone ? (
                            <span className="flex items-center gap-2">
                              {waLink(user.phone) ? (
                                <a
                                  href={waLink(user.phone)!}
                                  target="_blank"
                                  rel="noopener noreferrer"
                                  className="text-green-700 hover:underline"
                                  title="Ouvrir la discussion WhatsApp"
                                >
                                  {user.phone}
                                </a>
                              ) : (
                                user.phone
                              )}
                            </span>
                          ) : (
                            '—'
                          )}
                          <span className="block text-[11px] text-green-900/40">
                            {user.country_label && user.country_label !== '—'
                              ? user.country_label
                              : 'pays inconnu'}
                          </span>
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
                            onRole={(role, action) => setToRole({ user, role, action })}
                            onSuspend={() => setToSuspend(user)}
                            onReactivate={() => handleReactivate(user.id)}
                            onPromote={() => setToPromote(user)}
                            onMessage={() => openMessage(user)}
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

      <ConfirmDialog
        open={!!toRole}
        title={toRole?.action === 'grant' ? 'Accorder ce rôle ?' : 'Retirer ce rôle ?'}
        description={
          toRole
            ? `${toRole.action === 'grant' ? 'Accorder' : 'Retirer'} le rôle ${ROLE_LABELS[toRole.role] || toRole.role} à ${toRole.user.email}. L'utilisateur sera déconnecté de ses sessions actuelles et devra se reconnecter.`
            : undefined
        }
        confirmLabel={toRole?.action === 'grant' ? 'Accorder' : 'Retirer'}
        cancelLabel="Annuler"
        danger={toRole?.action === 'revoke'}
        onConfirm={handleSetRole}
        onCancel={() => setToRole(null)}
      />

      {messageTarget && (
        <div className="fixed inset-0 z-100 flex items-center justify-center p-4">
          <div
            className="absolute inset-0 bg-green-950/60 backdrop-blur-sm"
            onClick={() => setMessageTarget(null)}
            aria-hidden
          />
          <div
            role="dialog"
            aria-modal="true"
            className="relative w-full max-w-md rounded-2xl bg-white shadow-lift border border-green-900/10 p-6"
          >
            <h2 className="font-display font-bold text-lg text-green-950">
              Envoyer un message à {messageTarget.email}
            </h2>

            {messageSent ? (
              <>
                <p className="text-sm text-green-700 mt-4">Message envoyé avec succès.</p>
                <div className="mt-6 flex justify-end">
                  <Button onClick={() => setMessageTarget(null)}>Fermer</Button>
                </div>
              </>
            ) : (
              <>
                {messageError && (
                  <div className="mt-3 p-3 bg-destructive/10 text-destructive rounded text-sm">{messageError}</div>
                )}
                <div className="mt-4 space-y-3">
                  <div className="space-y-1.5">
                    <label htmlFor="msg-subject" className="text-sm font-medium text-green-900">
                      Sujet
                    </label>
                    <Input
                      id="msg-subject"
                      value={messageSubject}
                      onChange={(e) => setMessageSubject(e.target.value)}
                      placeholder="Ex: À propos de votre paiement"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label htmlFor="msg-body" className="text-sm font-medium text-green-900">
                      Message
                    </label>
                    <textarea
                      id="msg-body"
                      value={messageBody}
                      onChange={(e) => setMessageBody(e.target.value)}
                      rows={6}
                      className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                      placeholder="Bonjour, ..."
                    />
                  </div>
                </div>
                <div className="mt-6 flex justify-end gap-3">
                  <Button variant="outline" onClick={() => setMessageTarget(null)}>
                    Annuler
                  </Button>
                  <Button onClick={handleSendMessage} disabled={sendingMessage}>
                    {sendingMessage ? 'Envoi…' : 'Envoyer'}
                  </Button>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      {groupMsgOpen && (
        <div className="fixed inset-0 z-100 flex items-center justify-center p-4">
          <div
            className="absolute inset-0 bg-green-950/60 backdrop-blur-sm"
            onClick={() => setGroupMsgOpen(false)}
            aria-hidden
          />
          <div
            role="dialog"
            aria-modal="true"
            className="relative w-full max-w-lg rounded-2xl bg-white shadow-lift border border-green-900/10 p-6 max-h-[90vh] overflow-y-auto"
          >
            <h2 className="font-display font-bold text-lg text-green-950">Message groupé par pays</h2>
            <p className="text-sm text-green-900/60 mt-1">
              Envoie un email à tous les comptes du pays choisi (pays déduit de l&apos;indicatif du
              téléphone). Le lien de la communauté WhatsApp du pays est ajouté automatiquement en pied
              s&apos;il est configuré dans les réglages.
            </p>

            {groupResult ? (
              <>
                <p className="text-sm text-green-700 mt-4">{groupResult}</p>
                <div className="mt-6 flex justify-end">
                  <Button
                    onClick={() => {
                      setGroupMsgOpen(false);
                      setGroupResult('');
                      setGroupSubject('');
                      setGroupHtml('');
                    }}
                  >
                    Fermer
                  </Button>
                </div>
              </>
            ) : (
              <>
                {groupError && (
                  <div className="mt-3 p-3 bg-destructive/10 text-destructive rounded text-sm">
                    {groupError}
                  </div>
                )}
                <div className="mt-4 space-y-3">
                  <div className="space-y-1.5">
                    <label className="text-sm font-medium text-green-900">Pays ciblé</label>
                    <Select value={groupCountry} onValueChange={(v) => setGroupCountry(v || 'all')}>
                      <SelectTrigger className="bg-white">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="all">Tous les pays</SelectItem>
                        {countryBuckets.map((b) => (
                          <SelectItem key={b.country || 'unknown'} value={b.country || 'UNKNOWN'}>
                            {b.country_label} — {b.users} compte(s)
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-1.5">
                    <label htmlFor="grp-subject" className="text-sm font-medium text-green-900">
                      Sujet
                    </label>
                    <Input
                      id="grp-subject"
                      value={groupSubject}
                      onChange={(e) => setGroupSubject(e.target.value)}
                      placeholder="Ex: Rejoignez la communauté DIARRA"
                    />
                  </div>
                  <div className="space-y-1.5">
                    <label htmlFor="grp-body" className="text-sm font-medium text-green-900">
                      Message (HTML autorisé)
                    </label>
                    <textarea
                      id="grp-body"
                      value={groupHtml}
                      onChange={(e) => setGroupHtml(e.target.value)}
                      rows={7}
                      className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono"
                      placeholder="<p>Bonjour,</p><p>...</p>"
                    />
                  </div>
                </div>
                <div className="mt-6 flex justify-end gap-3">
                  <Button variant="outline" onClick={() => setGroupMsgOpen(false)}>
                    Annuler
                  </Button>
                  <Button
                    variant="outline"
                    disabled={groupSending}
                    onClick={async () => {
                      if (!groupSubject.trim() || !groupHtml.trim()) {
                        setGroupError('Le sujet et le message sont requis.');
                        return;
                      }
                      setGroupError('');
                      setGroupSending(true);
                      try {
                        const r = await api.sendBroadcast({
                          subject: groupSubject.trim(),
                          html: groupHtml.trim(),
                          test_only: true,
                          ...(groupCountry !== 'all' ? { country: groupCountry } : {}),
                        });
                        setGroupResult(`Email de test envoyé à ${r.to}. Vérifiez le rendu avant l'envoi réel.`);
                      } catch (err: any) {
                        setGroupError(friendlyError(err));
                      } finally {
                        setGroupSending(false);
                      }
                    }}
                  >
                    Test (à moi)
                  </Button>
                  <Button onClick={handleSendGroupMessage} disabled={groupSending}>
                    {groupSending ? 'Envoi…' : `Envoyer (${countryLabelOf(groupCountry)})`}
                  </Button>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      <ConfirmDialog
        open={!!toPromote}
        title="Promouvoir en administrateur ?"
        description={
          toPromote
            ? `${toPromote.email} aura accès complet à l'administration DIARRA (modération, utilisateurs, finance, remboursements). Cette action est réservée aux admins à accès complet.`
            : undefined
        }
        confirmLabel="Promouvoir"
        cancelLabel="Annuler"
        danger
        onConfirm={handlePromote}
        onCancel={() => setToPromote(null)}
      />
    </main>
  );
}
