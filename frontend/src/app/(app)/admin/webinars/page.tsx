'use client';

// Admin → Webinaires : création et pilotage des webinaires Yes.abmcy
// (jusqu'à 1000 spectateurs). DIARRA ne fait que créer / démarrer / terminer
// et lire les statistiques ; la diffusion vidéo, le tchat, les questions, les
// sondages, l'inscription et le replay se passent entièrement sur
// l'interface Yes.abmcy — jamais intégrés ici.

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { friendlyError } from '@/lib/error-messages';
import { ArrowLeftIcon } from '@/components/icons';

interface Webinar {
  id: string;
  title: string;
  description?: string;
  scheduled_start_at?: string;
  estimated_duration_minutes?: number;
  access_type?: string;
  status?: string;
  recording_url?: string;
  [key: string]: unknown; // champs YES non typés (liens d'inscription, etc.)
}

interface Stats {
  registered_count: number;
  attended_count: number;
  average_watch_minutes: number;
}

interface RegField {
  key: string;
  label: string;
  required: boolean;
}

const STATUS_LABELS: Record<string, string> = {
  scheduled: 'Programmé',
  live: 'En direct',
  started: 'En direct',
  ended: 'Terminé',
  cancelled: 'Annulé',
};

// Liens renvoyés par YES (inscription, salle, replay...) : tout champ
// "*_url" non vide, affiché tel quel — la forme exacte de l'objet YES n'est
// pas figée par la doc au-delà des champs principaux.
function webinarLinks(w: Webinar): { label: string; href: string }[] {
  return Object.entries(w)
    .filter(([k, v]) => k.endsWith('_url') && typeof v === 'string' && /^https?:\/\//.test(v))
    .map(([k, v]) => ({
      label:
        k === 'recording_url'
          ? 'Replay'
          : k.replace(/_url$/, '').replace(/_/g, ' ').replace(/^./, (c) => c.toUpperCase()),
      href: v as string,
    }));
}

function slugKey(label: string): string {
  return label
    .normalize('NFD')
    .replace(/[̀-ͯ]/g, '')
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_|_$/g, '');
}

function defaultStart(): string {
  const d = new Date(Date.now() + 24 * 3600 * 1000);
  d.setMinutes(0, 0, 0);
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:00`;
}

function Toggle({ id, label, checked, onChange }: { id: string; label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label htmlFor={id} className="flex items-center gap-2 text-sm text-green-950 cursor-pointer">
      <input id={id} type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} className="h-4 w-4 accent-green-800" />
      {label}
    </label>
  );
}

export default function AdminWebinarsPage() {
  const [webinars, setWebinars] = useState<Webinar[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [msg, setMsg] = useState('');
  const [busyId, setBusyId] = useState<string | null>(null);
  const [stats, setStats] = useState<Record<string, Stats>>({});
  const [endTarget, setEndTarget] = useState<Webinar | null>(null);
  const [startTarget, setStartTarget] = useState<Webinar | null>(null);
  const [recordOnStart, setRecordOnStart] = useState(true);

  // Formulaire de création
  const [showForm, setShowForm] = useState(false);
  const [creating, setCreating] = useState(false);
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [start, setStart] = useState(defaultStart);
  const [duration, setDuration] = useState(60);
  const [accessType, setAccessType] = useState<'public' | 'private'>('public');
  const [micOn, setMicOn] = useState(false);
  const [camOn, setCamOn] = useState(false);
  const [chatOn, setChatOn] = useState(true);
  const [qaOn, setQaOn] = useState(true);
  const [fields, setFields] = useState<RegField[]>([]);

  useEffect(() => {
    load();
  }, []);

  async function load() {
    setLoading(true);
    try {
      const r = await api.adminListWebinars();
      const list = [...(r.webinars || [])].sort((a, b) =>
        String(b.scheduled_start_at || '').localeCompare(String(a.scheduled_start_at || ''))
      );
      setWebinars(list);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  }

  function resetForm() {
    setTitle('');
    setDescription('');
    setStart(defaultStart());
    setDuration(60);
    setAccessType('public');
    setMicOn(false);
    setCamOn(false);
    setChatOn(true);
    setQaOn(true);
    setFields([]);
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setError('');
    setMsg('');
    if (!title.trim()) {
      setError('Le titre est obligatoire.');
      return;
    }
    const startDate = new Date(start);
    if (isNaN(startDate.getTime())) {
      setError('Date de début invalide.');
      return;
    }
    setCreating(true);
    try {
      await api.adminCreateWebinar({
        title: title.trim(),
        description: description.trim(),
        scheduled_start_at: startDate.toISOString(),
        estimated_duration_minutes: duration > 0 ? duration : 60,
        access_type: accessType,
        participant_mic_enabled_by_default: micOn,
        participant_camera_enabled_by_default: camOn,
        chat_enabled: chatOn,
        qa_enabled: qaOn,
        custom_registration_fields: fields
          .filter((f) => f.label.trim())
          .map((f) => ({ key: f.key || slugKey(f.label), label: f.label.trim(), required: f.required })),
      });
      setMsg(`Webinaire « ${title.trim()} » créé sur Yes.abmcy.`);
      resetForm();
      setShowForm(false);
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setCreating(false);
    }
  }

  async function handleStart(w: Webinar) {
    setStartTarget(null);
    setBusyId(w.id);
    setError('');
    setMsg('');
    try {
      await api.adminStartWebinar(w.id, recordOnStart);
      setMsg(
        `« ${w.title} » est démarré${recordOnStart ? ' (enregistrement activé)' : ''}. L’hôte anime depuis l’interface Yes.abmcy (portail partenaire → onglet Webinaires).`
      );
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
    }
  }

  async function handleEnd(w: Webinar) {
    setEndTarget(null);
    setBusyId(w.id);
    setError('');
    setMsg('');
    try {
      const r = await api.adminEndWebinar(w.id);
      setMsg(
        r.webinar?.recording_url
          ? `« ${w.title} » est terminé. Le replay est disponible.`
          : `« ${w.title} » est terminé.`
      );
      await load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
    }
  }

  async function handleStats(w: Webinar) {
    setBusyId(w.id);
    setError('');
    try {
      const s = await api.adminWebinarStats(w.id);
      setStats((prev) => ({ ...prev, [w.id]: s }));
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setBusyId(null);
    }
  }

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// administration" title="Webinaires" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Webinaires"
        description="Jusqu’à 1000 participants — diffusion, tchat, questions, sondages et replay hébergés par Yes.abmcy"
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Button size="sm" onClick={() => setShowForm((v) => !v)}>
              {showForm ? 'Fermer le formulaire' : 'Nouveau webinaire'}
            </Button>
            <Button variant="outline" size="sm" render={<Link href="/admin" />}>
              <ArrowLeftIcon size={16} className="mr-2" />
              Dashboard
            </Button>
          </div>
        }
      />

      <section className="max-w-5xl mx-auto px-4 sm:px-6 py-8 space-y-6">
        {error && (
          <div className="p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}
        {msg && (
          <div className="p-3 bg-green-900/5 text-green-900 rounded text-sm" role="status">
            {msg}
          </div>
        )}

        {showForm && (
          <form onSubmit={handleCreate} className="bg-white rounded-xl border border-green-900/10 shadow-card p-5 space-y-4">
            <h2 className="text-base font-semibold text-green-950">Nouveau webinaire</h2>

            <div className="space-y-1.5">
              <Label htmlFor="wb-title">Titre</Label>
              <Input id="wb-title" value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Ex : Lancement de la nouvelle collection" className="bg-white" />
            </div>

            <div className="space-y-1.5">
              <Label htmlFor="wb-desc">Description</Label>
              <Textarea id="wb-desc" value={description} onChange={(e) => setDescription(e.target.value)} rows={3} className="bg-white" />
            </div>

            <div className="grid gap-4 sm:grid-cols-3">
              <div className="space-y-1.5">
                <Label htmlFor="wb-start">Début</Label>
                <Input id="wb-start" type="datetime-local" value={start} onChange={(e) => setStart(e.target.value)} className="bg-white" />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="wb-duration">Durée estimée (min)</Label>
                <Input
                  id="wb-duration"
                  type="number"
                  min={5}
                  step={5}
                  value={duration}
                  onChange={(e) => setDuration(parseInt(e.target.value, 10) || 0)}
                  className="bg-white"
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="wb-access">Accès</Label>
                <select
                  id="wb-access"
                  value={accessType}
                  onChange={(e) => setAccessType(e.target.value as 'public' | 'private')}
                  className="h-9 w-full rounded-md border border-input bg-white px-3 text-sm"
                >
                  <option value="public">Public (lien d’inscription ouvert)</option>
                  <option value="private">Privé (sur invitation)</option>
                </select>
              </div>
            </div>

            <div className="grid gap-2 sm:grid-cols-2">
              <Toggle id="wb-chat" label="Tchat activé" checked={chatOn} onChange={setChatOn} />
              <Toggle id="wb-qa" label="Questions / réponses activées" checked={qaOn} onChange={setQaOn} />
              <Toggle id="wb-mic" label="Micro des participants ouvert par défaut" checked={micOn} onChange={setMicOn} />
              <Toggle id="wb-cam" label="Caméra des participants ouverte par défaut" checked={camOn} onChange={setCamOn} />
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <Label>Champs d’inscription supplémentaires</Label>
                <Button type="button" variant="outline" size="sm" onClick={() => setFields((f) => [...f, { key: '', label: '', required: false }])}>
                  + Ajouter un champ
                </Button>
              </div>
              {fields.length === 0 && (
                <p className="text-xs text-green-900/50">Aucun — Yes.abmcy demande déjà le nom et l’email.</p>
              )}
              {fields.map((f, i) => (
                <div key={i} className="flex flex-wrap items-center gap-2">
                  <Input
                    value={f.label}
                    onChange={(e) =>
                      setFields((all) => all.map((x, j) => (j === i ? { ...x, label: e.target.value, key: slugKey(e.target.value) } : x)))
                    }
                    placeholder="Ex : Entreprise, Pays, Téléphone…"
                    className="bg-white flex-1 min-w-48"
                  />
                  <Toggle
                    id={`wb-field-${i}`}
                    label="Obligatoire"
                    checked={f.required}
                    onChange={(v) => setFields((all) => all.map((x, j) => (j === i ? { ...x, required: v } : x)))}
                  />
                  <Button type="button" variant="ghost" size="sm" onClick={() => setFields((all) => all.filter((_, j) => j !== i))}>
                    Retirer
                  </Button>
                </div>
              ))}
            </div>

            <div className="flex justify-end gap-2 pt-2">
              <Button type="button" variant="ghost" onClick={() => setShowForm(false)}>
                Annuler
              </Button>
              <Button type="submit" disabled={creating}>
                {creating ? 'Création…' : 'Créer le webinaire'}
              </Button>
            </div>
          </form>
        )}

        {webinars.length === 0 ? (
          <EmptyState
            title="Aucun webinaire"
            description="Créez votre premier webinaire : Yes.abmcy gère l’inscription, les rappels, la diffusion et le replay."
          />
        ) : (
          <div className="space-y-3">
            {webinars.map((w) => {
              const status = (w.status || '').toLowerCase();
              const ended = status === 'ended' || status === 'cancelled' || status === 'completed';
              const live = status === 'live' || status === 'started' || status === 'in_progress';
              const s = stats[w.id];
              return (
                <div key={w.id} className="bg-white rounded-xl border border-green-900/10 shadow-card p-4">
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="font-medium text-green-950">{w.title}</p>
                      <p className="text-xs text-green-900/60 mt-0.5">
                        {w.scheduled_start_at ? new Date(w.scheduled_start_at).toLocaleString('fr-FR') : 'Date non définie'}
                        {w.estimated_duration_minutes ? ` · ${w.estimated_duration_minutes} min` : ''}
                        {w.access_type ? ` · ${w.access_type === 'private' ? 'Privé' : 'Public'}` : ''}
                      </p>
                      {w.description && <p className="text-sm text-green-900/70 mt-1.5 line-clamp-2">{w.description}</p>}
                    </div>
                    {w.status && (
                      <Badge variant="outline" className={live ? 'border-red-300 bg-red-50 text-red-700' : ''}>
                        {STATUS_LABELS[status] || w.status}
                      </Badge>
                    )}
                  </div>

                  {webinarLinks(w).length > 0 && (
                    <div className="flex flex-wrap gap-x-4 gap-y-1 mt-3 text-sm">
                      {webinarLinks(w).map((l) => (
                        <a key={l.href} href={l.href} target="_blank" rel="noopener noreferrer" className="text-green-700 hover:underline">
                          {l.label} ↗
                        </a>
                      ))}
                    </div>
                  )}

                  {s && (
                    <div className="grid grid-cols-3 gap-2 mt-3 text-center">
                      <div className="rounded-lg bg-green-900/5 p-2">
                        <p className="text-lg font-semibold text-green-950">{s.registered_count}</p>
                        <p className="text-[11px] text-green-900/60">inscrits</p>
                      </div>
                      <div className="rounded-lg bg-green-900/5 p-2">
                        <p className="text-lg font-semibold text-green-950">
                          {s.attended_count}
                          {s.registered_count > 0 && (
                            <span className="text-xs font-normal text-green-900/60">
                              {' '}
                              ({Math.round((s.attended_count / s.registered_count) * 100)} %)
                            </span>
                          )}
                        </p>
                        <p className="text-[11px] text-green-900/60">présents</p>
                      </div>
                      <div className="rounded-lg bg-green-900/5 p-2">
                        <p className="text-lg font-semibold text-green-950">{Math.round(s.average_watch_minutes)} min</p>
                        <p className="text-[11px] text-green-900/60">visionnage moyen</p>
                      </div>
                    </div>
                  )}

                  <div className="flex flex-wrap gap-2 mt-3 pt-3 border-t border-green-900/5">
                    {!ended && !live && (
                      <Button size="sm" disabled={busyId === w.id} onClick={() => setStartTarget(w)}>
                        Démarrer
                      </Button>
                    )}
                    {!ended && (
                      <Button size="sm" variant="outline" disabled={busyId === w.id} onClick={() => setEndTarget(w)}>
                        Terminer
                      </Button>
                    )}
                    <Button size="sm" variant="ghost" disabled={busyId === w.id} onClick={() => handleStats(w)}>
                      {busyId === w.id ? '…' : s ? 'Actualiser les statistiques' : 'Statistiques'}
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </section>

      {startTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" role="dialog" aria-modal="true">
          <div className="w-full max-w-md rounded-xl bg-white p-5 shadow-lift space-y-4">
            <h3 className="text-base font-semibold text-green-950">Démarrer « {startTarget.title} » ?</h3>
            <p className="text-sm text-green-900/70">
              Le webinaire passe en direct sur Yes.abmcy. L’hôte anime ensuite depuis l’interface Yes.abmcy.
            </p>
            <Toggle id="wb-record" label="Enregistrer (replay disponible après la fin)" checked={recordOnStart} onChange={setRecordOnStart} />
            <div className="flex justify-end gap-2">
              <Button variant="ghost" onClick={() => setStartTarget(null)}>
                Annuler
              </Button>
              <Button onClick={() => handleStart(startTarget)}>Démarrer</Button>
            </div>
          </div>
        </div>
      )}

      <ConfirmDialog
        open={!!endTarget}
        title="Terminer ce webinaire ?"
        description={endTarget ? `« ${endTarget.title} » sera clôturé pour tous les participants. Cette action est définitive.` : undefined}
        confirmLabel="Terminer"
        cancelLabel="Annuler"
        danger
        onConfirm={() => endTarget && handleEnd(endTarget)}
        onCancel={() => setEndTarget(null)}
      />
    </main>
  );
}
