'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/page-header';
import { ArrowLeftIcon, TrashIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';

const MAX_OFFERS = 3;

interface OfferDraft {
  title: string;
  isFree: boolean;
  priceCFA: string;
}

const emptyOffer = (): OfferDraft => ({ title: '', isFree: false, priceCFA: '' });

export default function NewEventPage() {
  const router = useRouter();
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [eventDate, setEventDate] = useState('');
  const [meetingLink, setMeetingLink] = useState('');
  const [coverFile, setCoverFile] = useState<File | null>(null);
  const [coverKey, setCoverKey] = useState('');
  const [coverPreview, setCoverPreview] = useState('');
  const [offers, setOffers] = useState<OfferDraft[]>([{ title: 'Standard', isFree: false, priceCFA: '' }]);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const hasFreeOffer = offers.some((o) => o.isFree);

  const handleCoverSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0] || null;
    setCoverFile(f);
    if (!f) {
      setCoverKey('');
      setCoverPreview('');
      return;
    }
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

  const updateOffer = (index: number, patch: Partial<OfferDraft>) => {
    setOffers((prev) => prev.map((o, i) => (i === index ? { ...o, ...patch } : o)));
  };

  const addOffer = () => {
    if (offers.length >= MAX_OFFERS) return;
    setOffers((prev) => [...prev, emptyOffer()]);
  };

  const removeOffer = (index: number) => {
    setOffers((prev) => prev.filter((_, i) => i !== index));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!title.trim()) {
      setError('Le titre est requis.');
      return;
    }
    if (offers.length === 0) {
      setError('Ajoutez au moins une offre.');
      return;
    }
    for (const offer of offers) {
      if (!offer.title.trim()) {
        setError('Chaque offre doit avoir un titre.');
        return;
      }
      if (!offer.isFree && (!offer.priceCFA || parseInt(offer.priceCFA, 10) <= 0)) {
        setError(`Indiquez un prix pour l'offre « ${offer.title} ».`);
        return;
      }
    }
    if (hasFreeOffer && !meetingLink.trim()) {
      setError('Un lien de visio (Zoom, Meet...) est requis pour une offre gratuite : il sera envoyé aux inscrits.');
      return;
    }

    setSubmitting(true);
    try {
      await api.createEvent({
        title: title.trim(),
        description: description.trim() || undefined,
        cover_image_key: coverKey || undefined,
        event_date: eventDate || undefined,
        meeting_link: meetingLink.trim() || undefined,
        offers: offers.map((o) => ({
          title: o.title.trim(),
          is_free: o.isFree,
          price_cfa: o.isFree ? undefined : parseInt(o.priceCFA, 10),
        })),
      });
      router.push('/vendor/events');
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <main className="min-h-screen">
      <PageHeader
        eyebrow="// espace vendeur"
        title="Nouvel événement"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/vendor/events" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Mes événements
          </Button>
        }
      />

      <section className="max-w-2xl mx-auto px-4 sm:px-6 py-10 space-y-6">
        {error && (
          <div className="p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Informations générales</CardTitle>
            <CardDescription>
              Votre événement sera visible publiquement une fois approuvé par un administrateur.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="title">Titre *</Label>
                <Input id="title" value={title} onChange={(e) => setTitle(e.target.value)} required />
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
                <Label htmlFor="cover">Image de couverture (optionnel)</Label>
                {coverPreview && (
                  <img
                    src={coverPreview}
                    alt="Aperçu de la couverture"
                    className="w-full h-40 object-cover rounded-lg border border-border"
                  />
                )}
                <Input id="cover" type="file" accept="image/*" onChange={handleCoverSelect} />
              </div>

              <div className="p-4 border border-border rounded-lg bg-secondary/60 space-y-2">
                <Label htmlFor="meeting-link">
                  Lien de visio (Zoom, Meet...) {hasFreeOffer && <span className="text-destructive">*</span>}
                </Label>
                <Input
                  id="meeting-link"
                  type="url"
                  placeholder="https://meet.google.com/..."
                  value={meetingLink}
                  onChange={(e) => setMeetingLink(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">
                  Envoyé automatiquement par email à chaque inscrit d&rsquo;une offre gratuite. Requis dès qu&rsquo;une
                  offre est gratuite.
                </p>
              </div>

              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <Label className="text-sm font-semibold">Offres ({offers.length}/{MAX_OFFERS})</Label>
                  {offers.length < MAX_OFFERS && (
                    <Button type="button" variant="outline" size="sm" onClick={addOffer}>
                      + Ajouter une offre
                    </Button>
                  )}
                </div>

                {offers.map((offer, i) => (
                  <div key={i} className="p-4 border border-border rounded-lg space-y-3">
                    <div className="flex items-center justify-between gap-2">
                      <div className="flex-1 space-y-2">
                        <Label htmlFor={`offer-title-${i}`} className="text-xs">
                          Titre de l&rsquo;offre
                        </Label>
                        <Input
                          id={`offer-title-${i}`}
                          value={offer.title}
                          onChange={(e) => updateOffer(i, { title: e.target.value })}
                          placeholder="Ex: Standard, VIP..."
                        />
                      </div>
                      {offers.length > 1 && (
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          className="text-destructive hover:text-destructive hover:bg-destructive/10 mt-6"
                          onClick={() => removeOffer(i)}
                        >
                          <TrashIcon size={16} />
                        </Button>
                      )}
                    </div>

                    <label className="flex items-center gap-2 cursor-pointer">
                      <Checkbox
                        checked={offer.isFree}
                        onCheckedChange={(checked) => updateOffer(i, { isFree: checked === true, priceCFA: '' })}
                      />
                      <span className="text-sm">Offre gratuite</span>
                    </label>

                    {!offer.isFree && (
                      <div className="space-y-1">
                        <Label htmlFor={`offer-price-${i}`} className="text-xs">
                          Prix (FCFA)
                        </Label>
                        <Input
                          id={`offer-price-${i}`}
                          type="number"
                          min={1}
                          value={offer.priceCFA}
                          onChange={(e) => updateOffer(i, { priceCFA: e.target.value })}
                        />
                      </div>
                    )}
                  </div>
                ))}
              </div>

              <Button type="submit" disabled={submitting} className="w-full font-semibold">
                {submitting ? 'Publication en cours...' : "Publier l'événement"}
              </Button>
            </form>
          </CardContent>
        </Card>
      </section>
    </main>
  );
}
