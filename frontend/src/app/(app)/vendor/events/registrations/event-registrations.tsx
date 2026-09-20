'use client';

import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { ArrowLeftIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';

interface Registration {
  id: string;
  full_name: string;
  email: string;
  phone_number: string;
  created_at: string;
}

// Inscrits aux offres GRATUITES d'un événement (les offres payantes sont
// des achats catalogue normaux, visibles depuis /vendor/sales — voir la
// séparation faite côté backend : event_registrations vs sales).
export default function EventRegistrations() {
  const searchParams = useSearchParams();
  const id = searchParams.get('id') || '';

  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [registrations, setRegistrations] = useState<Registration[]>([]);

  useEffect(() => {
    if (!id) return;
    api
      .getEventRegistrations(id)
      .then((res) => setRegistrations(res.registrations || []))
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  }, [id]);

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// espace vendeur" title="Inscrits" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// espace vendeur"
        title="Inscrits (offres gratuites)"
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
            <CardTitle>{registrations.length} inscrit(s)</CardTitle>
            <CardDescription>
              Les inscrits aux offres payantes apparaissent dans vos ventes, pas ici.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            {registrations.length === 0 && (
              <p className="text-sm text-muted-foreground italic">Aucune inscription pour le moment.</p>
            )}
            {registrations.map((reg) => (
              <div key={reg.id} className="p-3 rounded-lg border border-border space-y-1">
                <p className="text-sm font-semibold">{reg.full_name}</p>
                <p className="text-xs text-muted-foreground">{reg.email}</p>
                <p className="text-xs text-muted-foreground font-mono">{reg.phone_number}</p>
                <p className="text-xs text-muted-foreground">{new Date(reg.created_at).toLocaleString('fr-FR')}</p>
              </div>
            ))}
          </CardContent>
        </Card>
      </section>
    </main>
  );
}
