'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { TrashIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';

interface Announcement {
  id: string;
  title: string;
  body: string;
  created_at: string;
}

// "Dernières mises à jour" — visibles sur le dashboard de TOUS les vendeurs
// (pas d'email, pas de ciblage par vendeur) dès publication. Voir
// backend/internal/handler/announcement_handler.go.
export default function AdminAnnouncementsPage() {
  const [items, setItems] = useState<Announcement[]>([]);
  const [loading, setLoading] = useState(true);
  const [title, setTitle] = useState('');
  const [body, setBody] = useState('');
  const [publishing, setPublishing] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<Announcement | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  const canPublish = title.trim().length > 0 && body.trim().length > 0;

  const load = async () => {
    try {
      const result = await api.getAdminAnnouncements();
      setItems(result.announcements);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handlePublish = async () => {
    setError('');
    setMessage('');
    setPublishing(true);
    try {
      await api.createAnnouncement({ title: title.trim(), body: body.trim() });
      setTitle('');
      setBody('');
      setMessage('Publié — visible immédiatement sur le dashboard de tous les vendeurs.');
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setPublishing(false);
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    setBusyId(deleteTarget.id);
    setDeleteTarget(null);
    try {
      await api.deleteAnnouncement(deleteTarget.id);
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
    }
  };

  if (loading) return <PageLoader />;

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Dernières mises à jour"
        description="Message visible immédiatement sur le dashboard de tous les vendeurs — pas d'email envoyé."
        back="/admin"
      />

      <div className="max-w-2xl mx-auto px-4 sm:px-6 py-6 space-y-6">
        <Card>
          <CardHeader>
            <CardTitle>Publier un message</CardTitle>
            <CardDescription>Visible sur /vendor pour tous les comptes vendeur, dès publication.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {error && (
              <div className="p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
                {error}
              </div>
            )}
            {message && (
              <div className="p-3 bg-green-50 text-green-800 rounded text-sm" role="status">
                {message}
              </div>
            )}
            <div className="space-y-2">
              <Label htmlFor="ann-title">Titre</Label>
              <Input id="ann-title" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Ex : Nouveau moyen de paiement disponible" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="ann-body">Message</Label>
              <Textarea id="ann-body" value={body} onChange={(e) => setBody(e.target.value)} rows={4} placeholder="Détails du message…" />
            </div>
            <Button onClick={handlePublish} disabled={!canPublish || publishing}>
              {publishing ? 'Publication…' : 'Publier'}
            </Button>
          </CardContent>
        </Card>

        <div>
          <h2 className="font-display font-bold text-green-950 mb-3">Historique</h2>
          {items.length === 0 ? (
            <EmptyState title="Aucune mise à jour publiée" description="Les messages publiés apparaîtront ici." />
          ) : (
            <div className="space-y-2">
              {items.map((a) => (
                <div key={a.id} className="p-4 rounded-xl border border-green-900/10 bg-white shadow-card flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="font-semibold text-green-950">{a.title}</p>
                    <p className="text-sm text-green-900/70 mt-1 whitespace-pre-wrap">{a.body}</p>
                    <p className="text-xs text-green-900/50 mt-2">
                      {new Date(a.created_at).toLocaleString('fr-FR', { dateStyle: 'medium', timeStyle: 'short' })}
                    </p>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={busyId === a.id}
                    onClick={() => setDeleteTarget(a)}
                    aria-label="Supprimer"
                  >
                    <TrashIcon size={15} />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      <ConfirmDialog
        open={!!deleteTarget}
        title="Supprimer cette mise à jour ?"
        description={deleteTarget ? `« ${deleteTarget.title} » disparaîtra du dashboard des vendeurs.` : ''}
        onConfirm={handleDelete}
        onCancel={() => setDeleteTarget(null)}
      />
    </main>
  );
}
