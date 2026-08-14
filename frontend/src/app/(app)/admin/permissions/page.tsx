'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { friendlyError } from '@/lib/error-messages';
import { cn } from '@/lib/utils';

interface Admin {
  id: string;
  email: string;
  permissions: string[];
  created_at: string;
}

const SCOPES: { key: string; label: string }[] = [
  { key: 'moderation', label: 'Modération produits' },
  { key: 'users', label: 'Gestion utilisateurs' },
  { key: 'finance', label: 'Finance (ventes, remboursements, versements)' },
  { key: 'infra', label: 'Infrastructure' },
];

type PermAction = { adminId: string; email: string; scope: string; label: string; action: 'grant' | 'revoke' };

export default function AdminPermissionsPage() {
  const [admins, setAdmins] = useState<Admin[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [toToggle, setToToggle] = useState<PermAction | null>(null);
  const [toDemote, setToDemote] = useState<Admin | null>(null);

  useEffect(() => {
    loadAdmins();
  }, []);

  const loadAdmins = async () => {
    setLoading(true);
    try {
      const result = await api.getAdmins();
      setAdmins(result.admins);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  };

  const handleTogglePermission = async () => {
    if (!toToggle) return;
    try {
      await api.setAdminPermission(toToggle.adminId, toToggle.scope, toToggle.action);
      await loadAdmins();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setToToggle(null);
    }
  };

  const handleDemote = async () => {
    if (!toDemote) return;
    try {
      await api.setAdminStatus(toDemote.id, 'revoke');
      await loadAdmins();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setToDemote(null);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// administration" title="Droits admin" description="Gestion des accès administrateur" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Droits admin"
        description="Gestion des accès administrateur — réservé aux admins à accès complet"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/admin/users" />}>
            ← Utilisateurs
          </Button>
        }
      />

      <section className="max-w-4xl mx-auto px-4 sm:px-6 py-10">
        <p className="text-sm text-green-900/60 mb-6">
          Pour promouvoir un utilisateur en admin, utilisez le menu d&apos;un utilisateur sur la page{' '}
          <Link href="/admin/users" className="text-green-700 underline">Utilisateurs</Link>. Un admin sans aucun
          scope coché ci-dessous a l&apos;accès complet.
        </p>

        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {admins.length === 0 ? (
          <EmptyState title="Aucun administrateur" description="Aucun compte n'a le statut administrateur." />
        ) : (
          <div className="space-y-4">
            {admins.map((admin) => {
              const fullAccess = admin.permissions.length === 0;
              return (
                <div key={admin.id} className="p-5 border rounded-xl bg-white shadow-card border-green-900/10">
                  <div className="flex items-start justify-between gap-3 mb-3">
                    <div>
                      <p className="font-semibold text-green-950">{admin.email}</p>
                      <p className="text-xs text-green-900/40 mt-0.5">
                        Admin depuis le {new Date(admin.created_at).toLocaleDateString('fr-FR')}
                      </p>
                    </div>
                    {fullAccess && (
                      <Badge className="bg-lime/20 text-green-800 hover:bg-lime/20 shrink-0">Accès complet</Badge>
                    )}
                  </div>

                  <div className="flex flex-wrap gap-2 mb-4">
                    {SCOPES.map((scope) => {
                      const active = admin.permissions.includes(scope.key);
                      return (
                        <button
                          key={scope.key}
                          type="button"
                          onClick={() =>
                            setToToggle({
                              adminId: admin.id,
                              email: admin.email,
                              scope: scope.key,
                              label: scope.label,
                              action: active ? 'revoke' : 'grant',
                            })
                          }
                          className={cn(
                            'px-3 py-1.5 rounded-full text-xs font-medium border transition-colors',
                            active
                              ? 'bg-green-100 text-green-700 border-green-200 hover:bg-green-200'
                              : 'bg-white text-green-900/50 border-green-900/15 hover:bg-green-50'
                          )}
                        >
                          {scope.label} {active ? '✓' : '+'}
                        </button>
                      );
                    })}
                  </div>

                  <Button
                    variant="outline"
                    size="sm"
                    className="text-destructive hover:text-destructive hover:bg-destructive/10"
                    onClick={() => setToDemote(admin)}
                  >
                    Retirer les droits admin
                  </Button>
                </div>
              );
            })}
          </div>
        )}
      </section>

      <ConfirmDialog
        open={!!toToggle}
        title={toToggle?.action === 'grant' ? 'Accorder ce scope ?' : 'Retirer ce scope ?'}
        description={
          toToggle
            ? `${toToggle.action === 'grant' ? 'Accorder' : 'Retirer'} l'accès "${toToggle.label}" à ${toToggle.email}. Ses sessions actuelles seront terminées, il devra se reconnecter.`
            : undefined
        }
        confirmLabel={toToggle?.action === 'grant' ? 'Accorder' : 'Retirer'}
        cancelLabel="Annuler"
        danger={toToggle?.action === 'revoke'}
        onConfirm={handleTogglePermission}
        onCancel={() => setToToggle(null)}
      />

      <ConfirmDialog
        open={!!toDemote}
        title="Retirer les droits admin ?"
        description={
          toDemote
            ? `${toDemote.email} perdra tout accès à l'administration DIARRA et sera déconnecté de ses sessions actuelles.`
            : undefined
        }
        confirmLabel="Retirer les droits admin"
        cancelLabel="Annuler"
        danger
        onConfirm={handleDemote}
        onCancel={() => setToDemote(null)}
      />
    </main>
  );
}
