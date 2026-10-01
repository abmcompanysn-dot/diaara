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
  // Lien PUBLIC d'inscription à partager (doc YES §6.1).
  registration_link?: string;
  // Lien « Rejoindre en tant qu'hôte », valable 10 minutes : gardé en
  // mémoire seulement (hostLinks), jamais stocké (doc YES §6.1/6.3).
  host_join_link?: string;
  [key: string]: unknown; // autres champs YES non typés
}

// Marge sous les 10 minutes de validité du lien hôte : au-delà, on propose
// d'en obtenir un nouveau plutôt que d'ouvrir un lien sans doute expiré.
const HOST_LINK_TTL_MS = 9 * 60 * 1000;

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

// Inscrit (doc YES Business §6.6).
interface Registration {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  phone: string;
  company: string;
  attended: boolean;
  registered_at: string;
}

// Statuts documentés par YES Business (§6) : scheduled | live | ended | cancelled.
const STATUS_LABELS: Record<string, string> = {
  scheduled: 'Programmé',
  live: 'En direct',
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

  // Inscrits : chargés à la demande, un seul panneau ouvert à la fois.
  const [regsOpenId, setRegsOpenId] = useState<string | null>(null);
  const [regs, setRegs] = useState<Record<string, Registration[]>>({});
  const [regsLoadingId, setRegsLoadingId] = useState<string | null>(null);
  const [regsError, setRegsError] = useState('');
  const [resendingId, setResendingId] = useState<string | null>(null);
  const [resentIds, setResentIds] = useState<Set<string>>(new Set());
  const [regSearch, setRegSearch] = useState('');

  // Dernier lien hôte reçu par webinaire (réponse list ou start la plus
  // récente) + heure de réception, en mémoire uniquement.
  const [hostLinks, setHostLinks] = useState<Record<string, { href: string; at: number }>>({});
  const [copiedId, setCopiedId] = useState<string | null>(null);
  // Re-rendu toutes les 30 s pour faire basculer un lien hôte en « expiré ».
  const [, setTick] = useState(0);
  useEffect(() => {
    const t = setInterval(() => setTick((n) => n + 1), 30_000);
    return () => clearInterval(t);
  }, []);

  function rememberHostLinks(list: Webinar[]) {
    const now = Date.now();
    setHostLinks((prev) => {
      const next = { ...prev };
      for (const w of list) {
        if (typeof w.host_join_link === 'string' && /^https?:\/\//.test(w.host_join_link)) {
          next[w.id] = { href: w.host_join_link, at: now };
        }
      }
      return next;
    });
  }

  async function copyRegistrationLink(w: Webinar) {
    if (!w.registration_link) return;
    try {
      await navigator.clipboard.writeText(w.registration_link);
      setCopiedId(w.id);
      setTimeout(() => setCopiedId((id) => (id === w.id ? null : id)), 2000);
    } catch {
      // Presse-papiers indisponible (navigateur ancien, page non sécurisée) :
      // le lien reste sélectionnable dans le champ affiché.
      setError('Copie impossible : sélectionnez le lien et copiez-le manuellement.');
    }
  }

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
      rememberHostLinks(list);
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
      const r = await api.adminStartWebinar(w.id, recordOnStart);
      setMsg(
        `« ${w.title} » est en direct${recordOnStart ? ' (enregistrement activé)' : ''}. Cliquez sur « Rejoindre en tant qu’hôte » pour animer.`
      );
      await load();
      // Après load : le lien de la réponse start est le plus récent.
      if (r.host_join_link) {
        setHostLinks((prev) => ({ ...prev, [w.id]: { href: r.host_join_link, at: Date.now() } }));
      }
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

  async function loadRegistrations(w: Webinar) {
    setRegsLoadingId(w.id);
    setRegsError('');
    try {
      const r = await api.adminWebinarRegistrations(w.id);
      setRegs((prev) => ({ ...prev, [w.id]: r.registrations || [] }));
    } catch (err: any) {
      setRegsError(friendlyError(err));
    } finally {
      setRegsLoadingId(null);
    }
  }

  function toggleRegistrations(w: Webinar) {
    if (regsOpenId === w.id) {
      setRegsOpenId(null);
      return;
    }
    setRegsOpenId(w.id);
    setRegSearch('');
    loadRegistrations(w);
  }

  async function handleResend(w: Webinar, reg: Registration) {
    setResendingId(reg.id);
    setRegsError('');
    setMsg('');
    try {
      await api.adminResendWebinarRegistration(w.id, reg.id);
      setResentIds((prev) => new Set(prev).add(reg.id));
      setMsg(`Email ${(w.status || '').toLowerCase() === 'live' ? '« c’est en direct »' : '« ça commence bientôt »'} renvoyé à ${reg.email}.`);
    } catch (err: any) {
      setRegsError(friendlyError(err));
    } finally {
      setResendingId(null);
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
              const ended = status === 'ended' || status === 'cancelled';
              const live = status === 'live';
              const s = stats[w.id];
              const regsOpen = regsOpenId === w.id;
              const allRegs = regs[w.id] || [];
              const q = regSearch.trim().toLowerCase();
              const shownRegs = q
                ? allRegs.filter((r) =>
                    [r.email, r.first_name, r.last_name, r.phone, r.company].some((v) => (v || '').toLowerCase().includes(q))
                  )
                : allRegs;
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

                  {(w.registration_link || live) && (
                    <div className="mt-3 space-y-2">
                      {w.registration_link && (
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="text-xs font-medium text-green-900/60">Lien d’inscription</span>
                          <input
                            readOnly
                            value={w.registration_link}
                            onFocus={(e) => e.currentTarget.select()}
                            className="h-8 min-w-0 flex-1 rounded-md border border-green-900/15 bg-green-900/5 px-2 text-xs text-green-950"
                            aria-label="Lien d’inscription du webinaire"
                          />
                          <Button size="sm" variant="outline" onClick={() => copyRegistrationLink(w)}>
                            {copiedId === w.id ? 'Copié ✓' : 'Copier le lien'}
                          </Button>
                        </div>
                      )}
                      {live &&
                        (() => {
                          const link = hostLinks[w.id];
                          const fresh = link && Date.now() - link.at < HOST_LINK_TTL_MS;
                          return fresh ? (
                            <Button
                              size="sm"
                              className="bg-red-600 hover:bg-red-700 text-white"
                              render={<a href={link.href} target="_blank" rel="noopener noreferrer" />}
                            >
                              ● Rejoindre en tant qu’hôte
                            </Button>
                          ) : (
                            <Button size="sm" variant="outline" disabled={loading} onClick={() => load()}>
                              Obtenir un nouveau lien hôte (l’ancien a expiré)
                            </Button>
                          );
                        })()}
                    </div>
                  )}

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
                    <Button size="sm" variant={regsOpen ? 'outline' : 'ghost'} onClick={() => toggleRegistrations(w)}>
                      {regsOpen ? 'Masquer les inscrits' : 'Inscrits'}
                    </Button>
                  </div>

                  {regsOpen && (
                    <div className="mt-3 pt-3 border-t border-green-900/5 space-y-3">
                      <div className="flex flex-wrap items-center justify-between gap-2">
                        <p className="text-sm font-medium text-green-950">
                          {regsLoadingId === w.id
                            ? 'Chargement des inscrits…'
                            : `${allRegs.length} inscrit(s) · ${allRegs.filter((r) => r.attended).length} présent(s) au live`}
                        </p>
                        <div className="flex items-center gap-2">
                          {allRegs.length > 0 && (
                            <Input
                              value={regSearch}
                              onChange={(e) => setRegSearch(e.target.value)}
                              placeholder="Rechercher un inscrit…"
                              className="h-8 w-52 bg-white text-sm"
                            />
                          )}
                          <Button size="sm" variant="ghost" disabled={regsLoadingId === w.id} onClick={() => loadRegistrations(w)}>
                            Actualiser
                          </Button>
                        </div>
                      </div>

                      {regsError && (
                        <p className="text-sm text-destructive" role="alert">
                          {regsError}
                        </p>
                      )}
                      {ended && allRegs.length > 0 && (
                        <p className="text-xs text-green-900/50">Webinaire terminé : la relance n’est plus disponible.</p>
                      )}

                      {regsLoadingId !== w.id && allRegs.length === 0 && !regsError && (
                        <p className="text-sm text-green-900/50">Personne n’est encore inscrit.</p>
                      )}

                      {shownRegs.length > 0 && (
                        <ul className="divide-y divide-green-900/5 rounded-lg border border-green-900/10">
                          {shownRegs.map((r) => {
                            const name = [r.first_name, r.last_name].filter(Boolean).join(' ') || '—';
                            const phoneDigits = (r.phone || '').replace(/\D/g, '');
                            return (
                              <li key={r.id} className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 p-3">
                                <div className="min-w-0">
                                  <p className="text-sm font-medium text-green-950">
                                    {name}
                                    {r.company && <span className="font-normal text-green-900/60"> · {r.company}</span>}
                                  </p>
                                  <p className="text-xs text-green-900/60 break-all">
                                    <a href={`mailto:${r.email}`} className="hover:underline">
                                      {r.email}
                                    </a>
                                    {r.phone && (
                                      <>
                                        {' · '}
                                        <a href={`tel:${r.phone}`} className="hover:underline">
                                          {r.phone}
                                        </a>
                                        {phoneDigits.length >= 8 && (
                                          <a
                                            href={`https://wa.me/${phoneDigits}`}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            className="ml-2 font-medium text-green-700 hover:underline"
                                          >
                                            WhatsApp
                                          </a>
                                        )}
                                      </>
                                    )}
                                  </p>
                                  <p className="text-[11px] text-green-900/40">
                                    Inscrit le {r.registered_at ? new Date(r.registered_at).toLocaleString('fr-FR') : '—'}
                                  </p>
                                </div>
                                <div className="flex items-center gap-2 shrink-0">
                                  <Badge
                                    variant="outline"
                                    className={r.attended ? 'border-green-300 bg-green-50 text-green-800' : 'text-green-900/50'}
                                  >
                                    {r.attended ? 'Présent' : 'Absent'}
                                  </Badge>
                                  {!ended && (
                                    <Button
                                      size="sm"
                                      variant="outline"
                                      disabled={resendingId === r.id}
                                      onClick={() => handleResend(w, r)}
                                      title={
                                        live
                                          ? 'Renvoie l’email « c’est en direct maintenant » avec le lien d’accès'
                                          : 'Renvoie l’email « ça commence bientôt » avec le lien d’accès'
                                      }
                                    >
                                      {resendingId === r.id ? 'Envoi…' : resentIds.has(r.id) ? 'Relancé ✓' : 'Relancer'}
                                    </Button>
                                  )}
                                </div>
                              </li>
                            );
                          })}
                        </ul>
                      )}
                      {q && allRegs.length > 0 && shownRegs.length === 0 && (
                        <p className="text-sm text-green-900/50">Aucun inscrit ne correspond à cette recherche.</p>
                      )}
                    </div>
                  )}
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
