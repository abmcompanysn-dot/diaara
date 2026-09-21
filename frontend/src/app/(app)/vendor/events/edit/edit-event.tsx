'use client';

import { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
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
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const [offers, setOffers] = useState<Offer[]>([]);
  const [addingOffer, setAddingOffer] = useState(false);
  const [editingOfferId, setEditingOfferId] = useState<string | null>(null);
  const [offerToDelete, setOfferToDelete] = useState<Offer | null>(null);
  const [offerError, setOfferError] = useState('');

  const loadOffers = () => {
    if (!id) return;
    api
      .getEvent(id)
      .then((res) => setOffers(res.offers || []))
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
        setOffers(res.offers || []);
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
    setSubmitting(true);
    try {
      await api.updateEvent(id, {
        title: title.trim() || undefined,
        description: description.trim() || undefined,
        cover_image_key: coverKey,
        event_date: eventDate || undefined,
        meeting_link: meetingLink.trim() || undefined,
      });
      router.push('/vendor/events');
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSubmitting(false);
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

              <Button type="submit" disabled={submitting} className="w-full font-semibold">
                {submitting ? 'Enregistrement...' : 'Enregistrer'}
              </Button>
            </form>
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
    </main>
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
