'use client';

import { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { ArrowLeftIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';

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

  useEffect(() => {
    if (!id) return;
    api
      .getVendorEvents()
      .then((res) => {
        const event = (res.events || []).find((e: any) => e.id === id);
        if (!event) {
          setError('Événement introuvable.');
          return;
        }
        setTitle(event.title || '');
        setDescription(event.description || '');
        setEventDate(event.event_date ? event.event_date.slice(0, 10) : '');
        setMeetingLink(event.meeting_link || '');
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
            <CardDescription>
              Les offres (prix, gratuit/payant) ne peuvent pas être modifiées ici pour l&rsquo;instant.
            </CardDescription>
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
      </section>
    </main>
  );
}
