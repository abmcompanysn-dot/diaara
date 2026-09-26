'use client';

import { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api, apiOrigin } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { ArrowLeftIcon, TrashIcon, EditIcon } from '@/components/icons';
import { formatPrice } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';

const MAX_OFFERS = 3;

interface Offer {
  id: string;
  title: string;
  is_free: boolean;
  product_id?: string | null;
  product_price_cfa?: number | null;
}

interface GalleryImage {
  id: string;
  file_key: string;
}

interface ScheduleItem {
  id: string;
  time_label: string;
  title: string;
  description?: string | null;
}

// Formulaire d'ajout d'offre — le titre et le prix suffisent ; is_free et
// le lien de visio de l'événement (déjà requis pour une offre gratuite à
// la création de l'événement) ne se redemandent pas ici.
function AddOfferForm({ onAdd, onCancel }: { onAdd: (title: string, isFree: boolean, priceCFA: number) => Promise<void>; onCancel: () => void }) {
  const [title, setTitle] = useState('');
  const [isFree, setIsFree] = useState(false);
  const [price, setPrice] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim()) {
      setError('Le titre est requis.');
      return;
    }
    if (!isFree && (!price || parseInt(price, 10) <= 0)) {
      setError('Indiquez un prix pour une offre payante.');
      return;
    }
    setSaving(true);
    setError('');
    try {
      await onAdd(title.trim(), isFree, parseInt(price, 10) || 0);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3 pt-3 border-t border-border">
      <div className="space-y-2">
        <Label>Titre de l&rsquo;offre</Label>
        <Input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Ex: VIP" />
      </div>
      <label className="flex items-center gap-2 cursor-pointer">
        <Checkbox checked={isFree} onCheckedChange={(c) => setIsFree(c === true)} />
        <span className="text-sm">Offre gratuite</span>
      </label>
      {!isFree && (
        <div className="space-y-2">
          <Label>Prix (FCFA)</Label>
          <Input type="number" min={1} value={price} onChange={(e) => setPrice(e.target.value)} />
        </div>
      )}
      {error && <p className="text-xs text-red-600">{error}</p>}
      <div className="flex gap-2">
        <Button type="submit" size="sm" disabled={saving} className="font-semibold">
          {saving ? 'Ajout...' : "Ajouter l'offre"}
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={onCancel}>
          Annuler
        </Button>
      </div>
    </form>
  );
}

export default function EditEvent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const id = searchParams.get('id') || '';

  const [loading, setLoading] = useState(true);
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [eventDate, setEventDate] = useState('');
  const [meetingLink, setMeetingLink] = useState('');
  const [coverPreview, setCoverPreview] = useState('');
  const [coverKey, setCoverKey] = useState<string | undefined>(undefined);
  const [accentColor, setAccentColor] = useState('');
  const [venueName, setVenueName] = useState('');
  const [venueAddress, setVenueAddress] = useState('');
  const [venueMapUrl, setVenueMapUrl] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const [offers, setOffers] = useState<Offer[]>([]);
  const [addingOffer, setAddingOffer] = useState(false);
  const [editingOfferId, setEditingOfferId] = useState<string | null>(null);
  const [offerToDelete, setOfferToDelete] = useState<Offer | null>(null);
  const [offerError, setOfferError] = useState('');

  const [gallery, setGallery] = useState<GalleryImage[]>([]);
  const [galleryUploading, setGalleryUploading] = useState(false);
  const [galleryError, setGalleryError] = useState('');

  const [schedule, setSchedule] = useState<ScheduleItem[]>([]);
  const [addingScheduleItem, setAddingScheduleItem] = useState(false);
  const [editingScheduleId, setEditingScheduleId] = useState<string | null>(null);
  const [scheduleItemToDelete, setScheduleItemToDelete] = useState<ScheduleItem | null>(null);
  const [scheduleError, setScheduleError] = useState('');

  const loadOffers = () => {
    if (!id) return;
    api
      .getEvent(id)
      .then((res) => setOffers(res.offers || []))
      .catch(() => {});
  };

  const loadSchedule = () => {
    if (!id) return;
    api
      .getEvent(id)
      .then((res) => setSchedule(res.schedule || []))
      .catch(() => {});
  };

  useEffect(() => {
    if (!id) return;
    api
      .getEvent(id)
      .then((res) => {
        const event = res.event;
        setTitle(event.title || '');
        setDescription(event.description || '');
        setEventDate(event.event_date ? event.event_date.slice(0, 10) : '');
        setMeetingLink(event.meeting_link || '');
        setAccentColor(event.accent_color || '');
        setVenueName(event.venue_name || '');
        setVenueAddress(event.venue_address || '');
        setVenueMapUrl(event.venue_map_url || '');
        if (event.cover_image_key) {
          setCoverKey(event.cover_image_key);
          setCoverPreview(`${apiOrigin}/api/events/${id}/cover`);
        }
        setOffers(res.offers || []);
        setGallery(res.gallery || []);
        setSchedule(res.schedule || []);
      })
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  }, [id]);

  const handleCoverSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0] || null;
    if (!f) return;
    setCoverPreview(URL.createObjectURL(f));
    try {
      const form = new FormData();
      form.append('file', f);
      form.append('type', 'cover');
      const res = await api.uploadFile(form);
      setCoverKey(res.file_key);
    } catch (err: any) {
      setError(friendlyError(err));
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (accentColor && !/^#[0-9a-fA-F]{6}$/.test(accentColor)) {
      setError('La couleur doit être au format #RRGGBB.');
      return;
    }
    setSubmitting(true);
    try {
      await api.updateEvent(id, {
        title: title.trim() || undefined,
        description: description.trim() || undefined,
        cover_image_key: coverKey,
        event_date: eventDate || undefined,
        meeting_link: meetingLink.trim() || undefined,
        accent_color: accentColor || undefined,
        venue_name: venueName.trim() || undefined,
        venue_address: venueAddress.trim() || undefined,
        venue_map_url: venueMapUrl.trim() || undefined,
      });
      router.push('/vendor/events');
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSubmitting(false);
    }
  };

  const handleGallerySelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0] || null;
    if (!f) return;
    setGalleryError('');
    setGalleryUploading(true);
    try {
      const form = new FormData();
      form.append('file', f);
      form.append('type', 'cover');
      const uploaded = await api.uploadFile(form);
      const res = await api.addEventGalleryImage(id, uploaded.file_key);
      setGallery((prev) => [...prev, res.image]);
    } catch (err: any) {
      setGalleryError(friendlyError(err));
    } finally {
      setGalleryUploading(false);
      e.target.value = '';
    }
  };

  const handleDeleteGalleryImage = async (imageId: string) => {
    setGalleryError('');
    try {
      await api.deleteEventGalleryImage(imageId);
      setGallery((prev) => prev.filter((img) => img.id !== imageId));
    } catch (err: any) {
      setGalleryError(friendlyError(err));
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// espace vendeur" title="Modifier l'événement" />
        <PageLoader />
      </main>
    );

  return (
    <main className="min-h-screen">
      <PageHeader
        eyebrow="// espace vendeur"
        title="Modifier l'événement"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/vendor/events" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Mes événements
          </Button>
        }
      />

      <section className="max-w-2xl mx-auto px-4 sm:px-6 py-10">
        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Informations générales</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="title">Titre</Label>
                <Input id="title" value={title} onChange={(e) => setTitle(e.target.value)} />
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  className="min-h-28"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="event-date">Date de l&rsquo;événement</Label>
                <Input id="event-date" type="date" value={eventDate} onChange={(e) => setEventDate(e.target.value)} />
              </div>

              <div className="space-y-2">
                <Label htmlFor="cover">Image de couverture</Label>
                {coverPreview && (
                  <img src={coverPreview} alt="Aperçu" className="w-full h-40 object-cover rounded-lg border border-border" />
                )}
                <Input id="cover" type="file" accept="image/*" onChange={handleCoverSelect} />
              </div>

              <div className="space-y-2">
                <Label htmlFor="meeting-link">Lien de visio (Zoom, Meet...)</Label>
                <Input
                  id="meeting-link"
                  type="url"
                  value={meetingLink}
                  onChange={(e) => setMeetingLink(e.target.value)}
                  placeholder="https://meet.google.com/..."
                />
                <p className="text-xs text-muted-foreground">
                  Envoyé automatiquement par email à chaque inscrit d&rsquo;une offre gratuite.
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="accent-color">Couleur d&rsquo;accent de la page</Label>
                <div className="flex items-center gap-3">
                  <input
                    id="accent-color"
                    type="color"
                    value={accentColor || '#0e6b46'}
                    onChange={(e) => setAccentColor(e.target.value)}
                    className="h-10 w-14 rounded-md border border-border cursor-pointer"
                  />
                  <Input
                    value={accentColor}
                    onChange={(e) => setAccentColor(e.target.value)}
                    placeholder="#0E6B46"
                    className="max-w-32"
                  />
                  {accentColor && (
                    <button
                      type="button"
                      onClick={() => setAccentColor('')}
                      className="text-xs text-muted-foreground underline"
                    >
                      Réinitialiser
                    </button>
                  )}
                </div>
                <p className="text-xs text-muted-foreground">
                  Remplace le vert DIARRA par défaut sur la page publique de cet événement.
                </p>
              </div>

              <div className="space-y-2 pt-2 border-t border-border">
                <Label>Lieu physique (optionnel)</Label>
                <Input
                  value={venueName}
                  onChange={(e) => setVenueName(e.target.value)}
                  placeholder="Nom du lieu (ex: Salle des fêtes de...)"
                />
                <Input
                  value={venueAddress}
                  onChange={(e) => setVenueAddress(e.target.value)}
                  placeholder="Adresse complète"
                />
                <Input
                  type="url"
                  value={venueMapUrl}
                  onChange={(e) => setVenueMapUrl(e.target.value)}
                  placeholder="Lien Google Maps (optionnel)"
                />
              </div>

              <Button type="submit" disabled={submitting} className="w-full font-semibold">
                {submitting ? 'Enregistrement...' : 'Enregistrer'}
              </Button>
            </form>
          </CardContent>
        </Card>

        <Card className="shadow-card border-green-900/5 mt-6">
          <CardHeader>
            <CardTitle>Galerie photo</CardTitle>
            <CardDescription>Photos supplémentaires affichées sur la page publique (éditions précédentes, ambiance...).</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {galleryError && <p className="text-xs text-red-600">{galleryError}</p>}
            {gallery.length > 0 && (
              <div className="grid grid-cols-3 gap-2">
                {gallery.map((img) => (
                  <div key={img.id} className="relative group">
                    <img
                      src={`${apiOrigin}/api/events/gallery/${img.id}/file`}
                      alt=""
                      className="w-full h-24 object-cover rounded-lg border border-border"
                    />
                    <button
                      type="button"
                      onClick={() => handleDeleteGalleryImage(img.id)}
                      className="absolute top-1 right-1 bg-black/60 text-white rounded-full p-1 opacity-0 group-hover:opacity-100 transition-opacity"
                      aria-label="Supprimer cette photo"
                    >
                      <TrashIcon size={14} />
                    </button>
                  </div>
                ))}
              </div>
            )}
            <Input type="file" accept="image/*" onChange={handleGallerySelect} disabled={galleryUploading} />
            {galleryUploading && <p className="text-xs text-muted-foreground">Envoi en cours...</p>}
          </CardContent>
        </Card>

        <Card className="shadow-card border-green-900/5 mt-6">
          <CardHeader>
            <CardTitle>Programme</CardTitle>
            <CardDescription>Le planning horaire de l&rsquo;événement, affiché sur la page publique.</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {scheduleError && <p className="text-xs text-red-600">{scheduleError}</p>}

            {schedule.map((item) =>
              editingScheduleId === item.id ? (
                <ScheduleItemEditForm
                  key={item.id}
                  item={item}
                  onCancel={() => setEditingScheduleId(null)}
                  onSave={async (data) => {
                    await api.updateEventScheduleItem(item.id, data);
                    setEditingScheduleId(null);
                    loadSchedule();
                  }}
                />
              ) : (
                <div key={item.id} className="flex items-start justify-between gap-3 p-3 rounded-lg border border-border">
                  <div className="min-w-0">
                    <p className="text-xs font-mono text-muted-foreground">{item.time_label}</p>
                    <p className="text-sm font-semibold truncate">{item.title}</p>
                    {item.description && <p className="text-xs text-muted-foreground mt-1">{item.description}</p>}
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <button
                      type="button"
                      onClick={() => setEditingScheduleId(item.id)}
                      className="text-muted-foreground hover:text-foreground"
                      aria-label={`Modifier ${item.title}`}
                    >
                      <EditIcon size={16} />
                    </button>
                    <button
                      type="button"
                      onClick={() => setScheduleItemToDelete(item)}
                      className="text-red-600 hover:text-red-700"
                      aria-label={`Supprimer ${item.title}`}
                    >
                      <TrashIcon size={16} />
                    </button>
                  </div>
                </div>
              )
            )}

            {addingScheduleItem ? (
              <AddScheduleItemForm
                onCancel={() => setAddingScheduleItem(false)}
                onAdd={async (data) => {
                  await api.addEventScheduleItem(id, data);
                  setAddingScheduleItem(false);
                  loadSchedule();
                }}
              />
            ) : (
              <Button variant="outline" size="sm" onClick={() => setAddingScheduleItem(true)}>
                + Ajouter un créneau
              </Button>
            )}
          </CardContent>
        </Card>

        <Card className="shadow-card border-green-900/5 mt-6">
          <CardHeader>
            <CardTitle>Offres ({offers.length}/{MAX_OFFERS})</CardTitle>
            <CardDescription>
              Le titre et le prix (pour une offre payante) sont modifiables. Le type gratuit/payant ne peut pas
              changer après création — supprimez puis recréez l&rsquo;offre si besoin.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {offerError && <p className="text-xs text-red-600">{offerError}</p>}

            {offers.map((offer) =>
              editingOfferId === offer.id ? (
                <OfferEditForm
                  key={offer.id}
                  offer={offer}
                  onCancel={() => setEditingOfferId(null)}
                  onSave={async (data) => {
                    await api.updateEventOffer(offer.id, data);
                    setEditingOfferId(null);
                    loadOffers();
                  }}
                />
              ) : (
                <div key={offer.id} className="flex items-center justify-between gap-3 p-3 rounded-lg border border-border">
                  <div className="min-w-0">
                    <p className="text-sm font-semibold truncate">{offer.title}</p>
                    <p className="text-xs text-muted-foreground">
                      {offer.is_free ? 'Gratuit' : formatPrice(offer.product_price_cfa || 0)}
                    </p>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <button
                      type="button"
                      onClick={() => setEditingOfferId(offer.id)}
                      className="text-muted-foreground hover:text-foreground"
                      aria-label={`Modifier ${offer.title}`}
                    >
                      <EditIcon size={16} />
                    </button>
                    <button
                      type="button"
                      onClick={() => setOfferToDelete(offer)}
                      disabled={offers.length <= 1}
                      className="text-red-600 hover:text-red-700 disabled:opacity-30 disabled:cursor-not-allowed"
                      aria-label={`Supprimer ${offer.title}`}
                      title={offers.length <= 1 ? "Impossible de supprimer la dernière offre" : undefined}
                    >
                      <TrashIcon size={16} />
                    </button>
                  </div>
                </div>
              )
            )}

            {addingOffer ? (
              <AddOfferForm
                onCancel={() => setAddingOffer(false)}
                onAdd={async (title, isFree, priceCFA) => {
                  await api.addEventOffer(id, { title, is_free: isFree, price_cfa: isFree ? undefined : priceCFA });
                  setAddingOffer(false);
                  loadOffers();
                }}
              />
            ) : offers.length < MAX_OFFERS ? (
              <Button variant="outline" size="sm" onClick={() => setAddingOffer(true)}>
                + Ajouter une offre
              </Button>
            ) : (
              <p className="text-xs text-muted-foreground">Maximum {MAX_OFFERS} offres par événement.</p>
            )}
          </CardContent>
        </Card>
      </section>

      <ConfirmDialog
        open={!!offerToDelete}
        title="Supprimer cette offre ?"
        description={`« ${offerToDelete?.title} » ne sera plus proposée aux inscrits.`}
        confirmLabel="Supprimer"
        cancelLabel="Annuler"
        danger
        onConfirm={async () => {
          if (!offerToDelete) return;
          try {
            await api.deleteEventOffer(offerToDelete.id);
            setOfferToDelete(null);
            loadOffers();
          } catch (err: any) {
            setOfferError(friendlyError(err));
            setOfferToDelete(null);
          }
        }}
        onCancel={() => setOfferToDelete(null)}
      />

      <ConfirmDialog
        open={!!scheduleItemToDelete}
        title="Supprimer ce créneau ?"
        description={`« ${scheduleItemToDelete?.title} » sera retiré du programme.`}
        confirmLabel="Supprimer"
        cancelLabel="Annuler"
        danger
        onConfirm={async () => {
          if (!scheduleItemToDelete) return;
          try {
            await api.deleteEventScheduleItem(scheduleItemToDelete.id);
            setScheduleItemToDelete(null);
            loadSchedule();
          } catch (err: any) {
            setScheduleError(friendlyError(err));
            setScheduleItemToDelete(null);
          }
        }}
        onCancel={() => setScheduleItemToDelete(null)}
      />
    </main>
  );
}

// Formulaire d'ajout d'un créneau de programme.
function AddScheduleItemForm({
  onAdd,
  onCancel,
}: {
  onAdd: (data: { time_label: string; title: string; description?: string }) => Promise<void>;
  onCancel: () => void;
}) {
  const [timeLabel, setTimeLabel] = useState('');
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!timeLabel.trim() || !title.trim()) {
      setError('L’heure et le titre sont requis.');
      return;
    }
    setSaving(true);
    setError('');
    try {
      await onAdd({ time_label: timeLabel.trim(), title: title.trim(), description: description.trim() || undefined });
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3 pt-3 border-t border-border">
      <div className="grid grid-cols-2 gap-2">
        <div className="space-y-2">
          <Label>Heure</Label>
          <Input value={timeLabel} onChange={(e) => setTimeLabel(e.target.value)} placeholder="14h00" />
        </div>
        <div className="space-y-2">
          <Label>Titre</Label>
          <Input value={title} onChange={(e) => setTitle(e.target.value)} placeholder="Ouverture" />
        </div>
      </div>
      <div className="space-y-2">
        <Label>Description (optionnel)</Label>
        <Textarea value={description} onChange={(e) => setDescription(e.target.value)} className="min-h-16" />
      </div>
      {error && <p className="text-xs text-red-600">{error}</p>}
      <div className="flex gap-2">
        <Button type="submit" size="sm" disabled={saving} className="font-semibold">
          {saving ? 'Ajout...' : 'Ajouter'}
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={onCancel}>
          Annuler
        </Button>
      </div>
    </form>
  );
}

// Formulaire d'édition d'un créneau existant.
function ScheduleItemEditForm({
  item,
  onSave,
  onCancel,
}: {
  item: ScheduleItem;
  onSave: (data: { time_label?: string; title?: string; description?: string }) => Promise<void>;
  onCancel: () => void;
}) {
  const [timeLabel, setTimeLabel] = useState(item.time_label);
  const [title, setTitle] = useState(item.title);
  const [description, setDescription] = useState(item.description || '');
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!timeLabel.trim() || !title.trim()) {
      setError('L’heure et le titre sont requis.');
      return;
    }
    setSaving(true);
    setError('');
    try {
      await onSave({ time_label: timeLabel.trim(), title: title.trim(), description: description.trim() });
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="p-3 rounded-lg border border-border space-y-3">
      <div className="grid grid-cols-2 gap-2">
        <div className="space-y-2">
          <Label>Heure</Label>
          <Input value={timeLabel} onChange={(e) => setTimeLabel(e.target.value)} />
        </div>
        <div className="space-y-2">
          <Label>Titre</Label>
          <Input value={title} onChange={(e) => setTitle(e.target.value)} />
        </div>
      </div>
      <div className="space-y-2">
        <Label>Description</Label>
        <Textarea value={description} onChange={(e) => setDescription(e.target.value)} className="min-h-16" />
      </div>
      {error && <p className="text-xs text-red-600">{error}</p>}
      <div className="flex gap-2">
        <Button type="submit" size="sm" disabled={saving} className="font-semibold">
          {saving ? 'Enregistrement...' : 'Enregistrer'}
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={onCancel}>
          Annuler
        </Button>
      </div>
    </form>
  );
}

// Formulaire d'édition d'une offre existante — titre toujours modifiable,
// prix seulement si l'offre est payante (is_free ne change jamais ici).
function OfferEditForm({
  offer,
  onSave,
  onCancel,
}: {
  offer: Offer;
  onSave: (data: { title?: string; price_cfa?: number }) => Promise<void>;
  onCancel: () => void;
}) {
  const [title, setTitle] = useState(offer.title);
  const [price, setPrice] = useState(String(offer.product_price_cfa ?? ''));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!title.trim()) {
      setError('Le titre est requis.');
      return;
    }
    setSaving(true);
    setError('');
    try {
      await onSave({
        title: title.trim(),
        price_cfa: offer.is_free ? undefined : parseInt(price, 10),
      });
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="p-3 rounded-lg border border-border space-y-3">
      <div className="space-y-2">
        <Label>Titre</Label>
        <Input value={title} onChange={(e) => setTitle(e.target.value)} />
      </div>
      {!offer.is_free && (
        <div className="space-y-2">
          <Label>Prix (FCFA)</Label>
          <Input type="number" min={1} value={price} onChange={(e) => setPrice(e.target.value)} />
        </div>
      )}
      {error && <p className="text-xs text-red-600">{error}</p>}
      <div className="flex gap-2">
        <Button type="submit" size="sm" disabled={saving} className="font-semibold">
          {saving ? 'Enregistrement...' : 'Enregistrer'}
        </Button>
        <Button type="button" variant="outline" size="sm" onClick={onCancel}>
          Annuler
        </Button>
      </div>
    </form>
  );
}
