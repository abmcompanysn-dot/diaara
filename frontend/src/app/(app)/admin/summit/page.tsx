'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { ArrowLeftIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';

interface SummitRegistration {
  id: string;
  full_name: string;
  email: string;
  phone_number: string;
  profile: 'vendeur' | 'acheteur' | 'entrepreneur' | 'curieux';
  created_at: string;
}

const PROFILE_LABELS: Record<string, string> = {
  vendeur: 'Vendeur',
  acheteur: 'Acheteur',
  entrepreneur: 'Entrepreneur',
  curieux: 'Curieux',
};

export default function AdminSummitPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [registrations, setRegistrations] = useState<SummitRegistration[]>([]);
  const [count, setCount] = useState(0);

  useEffect(() => {
    api
      .getSummitRegistrations()
      .then((res) => {
        setRegistrations(res.registrations || []);
        setCount(res.count || 0);
      })
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  }, []);

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// administration" title="DIARRA Summit" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="DIARRA Summit"
        description="UCAD Dakar &amp; en ligne, le 26 octobre 2026 — inscrits et billets"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/admin" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Tableau de bord
          </Button>
        }
      />

      <section className="max-w-3xl mx-auto px-4 sm:px-6 py-10 space-y-6">
        {error && (
          <div className="p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Billets (Tier Essentiel / Business 2026 / Premium)</CardTitle>
            <CardDescription>
              Les 3 billets sont des produits catalogue standard (catégorie « Événements »), créés via
              <code className="mx-1 px-1.5 py-0.5 rounded bg-green-900/5 text-xs">
                backend/cmd/seed_summit_tickets
              </code>
              et à approuver comme n&rsquo;importe quel produit avant qu&rsquo;ils n&rsquo;apparaissent
              publiquement sur <Link href="/summit" className="underline">/summit</Link> et le catalogue.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button size="sm" render={<Link href="/admin/products" />}>
              Gérer les billets dans Produits
            </Button>
          </CardContent>
        </Card>

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Inscrits ({count})</CardTitle>
            <CardDescription>
              Chaque inscription publique (page /summit) reçoit un email de confirmation automatique.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            {registrations.length === 0 && (
              <p className="text-sm text-muted-foreground italic">Aucune inscription pour le moment.</p>
            )}
            {registrations.map((reg) => (
              <div key={reg.id} className="p-3 rounded-lg border border-border space-y-1">
                <div className="flex items-center justify-between gap-3">
                  <p className="text-sm font-semibold truncate">{reg.full_name}</p>
                  <Badge className="bg-green-100 text-green-700 shrink-0">
                    {PROFILE_LABELS[reg.profile] || reg.profile}
                  </Badge>
                </div>
                <p className="text-xs text-muted-foreground">{reg.email}</p>
                <p className="text-xs text-muted-foreground font-mono">{reg.phone_number}</p>
                <p className="text-xs text-muted-foreground">
                  {new Date(reg.created_at).toLocaleString('fr-FR')}
                </p>
              </div>
            ))}
          </CardContent>
        </Card>
      </section>
    </main>
  );
}
