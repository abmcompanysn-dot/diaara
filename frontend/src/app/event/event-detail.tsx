'use client';

import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api } from '@/lib/api';
import { formatPrice } from '@/lib/constants';
import { PageLoader } from '@/components/page-loader';
import { CheckIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';

interface EventData {
  id: string;
  title: string;
  description?: string | null;
  event_date?: string | null;
  cover_image_key?: string | null;
}

interface Offer {
  id: string;
  title: string;
  is_free: boolean;
  product_id?: string | null;
  product_slug?: string | null;
  product_price_cfa?: number | null;
}

function FreeOfferForm({ offerId }: { offerId: string }) {
  const [fullName, setFullName] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!fullName.trim() || !email.trim() || !phone.trim()) {
      setError('Merci de remplir tous les champs.');
      return;
    }
    setSubmitting(true);
    setError('');
    try {
      await api.registerFreeEventOffer(offerId, { full_name: fullName.trim(), email: email.trim(), phone: phone.trim() });
      setSent(true);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSubmitting(false);
    }
  };

  if (sent) {
    return (
      <div className="flex items-center gap-2 text-sm text-green-800 py-2">
        <CheckIcon size={16} />
        Inscription confirmée — vérifiez votre email pour le lien de connexion.
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-2 mt-3">
      <input
        type="text"
        placeholder="Nom complet"
        value={fullName}
        onChange={(e) => setFullName(e.target.value)}
        className="w-full h-10 px-3 rounded-md border border-green-900/15 text-sm focus:outline-none focus:ring-2 focus:ring-green-600/30"
      />
      <input
        type="email"
        placeholder="Email"
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        className="w-full h-10 px-3 rounded-md border border-green-900/15 text-sm focus:outline-none focus:ring-2 focus:ring-green-600/30"
      />
      <input
        type="tel"
        placeholder="Téléphone"
        value={phone}
        onChange={(e) => setPhone(e.target.value)}
        className="w-full h-10 px-3 rounded-md border border-green-900/15 text-sm focus:outline-none focus:ring-2 focus:ring-green-600/30"
      />
      {error && <p className="text-xs text-red-600">{error}</p>}
      <button
        type="submit"
        disabled={submitting}
        className="w-full h-10 rounded-md bg-[#0E6B46] text-white text-sm font-semibold hover:bg-[#0c5a3c] transition-colors disabled:opacity-60"
      >
        {submitting ? 'Inscription...' : "S'inscrire gratuitement"}
      </button>
    </form>
  );
}

export default function EventDetail() {
  const searchParams = useSearchParams();
  const id = searchParams.get('id') || '';

  const [event, setEvent] = useState<EventData | null>(null);
  const [offers, setOffers] = useState<Offer[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!id) return;
    api
      .getEvent(id)
      .then((res) => {
        setEvent(res.event);
        setOffers(res.offers || []);
      })
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  }, [id]);

  if (loading) return <PageLoader />;

  if (error || !event) {
    return (
      <section className="max-w-2xl mx-auto px-4 py-20 text-center">
        <p className="text-green-900/70">{error || 'Événement introuvable.'}</p>
        <Link href="/catalog" className="text-forest underline text-sm mt-2 inline-block">
          Retour au catalogue
        </Link>
      </section>
    );
  }

  return (
    <>
      <section className="gradient-green text-white relative overflow-hidden">
        <div className="wax-pattern absolute inset-0" aria-hidden />
        <div className="relative max-w-3xl mx-auto px-4 py-16 text-center">
          {event.event_date && (
            <p className="font-mono text-sm text-green-300 uppercase tracking-widest mb-4">
              //{' '}
              {new Date(event.event_date).toLocaleDateString('fr-FR', {
                day: 'numeric',
                month: 'long',
                year: 'numeric',
              })}
            </p>
          )}
          <h1 className="font-display text-3xl sm:text-4xl font-bold tracking-tight">{event.title}</h1>
          {event.description && <p className="mt-5 text-white/75 max-w-xl mx-auto">{event.description}</p>}
        </div>
      </section>

      <section className="py-16 max-w-4xl mx-auto px-4">
        <h2 className="font-display text-2xl font-bold text-green-950 text-center mb-8">Offres</h2>
        <div className="grid sm:grid-cols-2 gap-6" style={{ gridTemplateColumns: offers.length === 1 ? '1fr' : undefined }}>
          {offers.map((offer) => (
            <div key={offer.id} className="rounded-2xl border border-green-900/10 bg-white shadow-lift p-6">
              <h3 className="font-display text-lg font-bold text-green-950">{offer.title}</h3>
              <p className="mt-1 text-2xl font-bold text-forest">
                {offer.is_free ? 'Gratuit' : formatPrice(offer.product_price_cfa || 0)}
              </p>
              {offer.is_free ? (
                <FreeOfferForm offerId={offer.id} />
              ) : (
                <Link
                  href={`/checkout?product=${offer.product_slug || offer.product_id}`}
                  className="mt-4 h-11 rounded-md bg-[#0E6B46] text-white text-sm font-semibold flex items-center justify-center hover:bg-[#0c5a3c] transition-colors"
                >
                  Choisir cette offre
                </Link>
              )}
            </div>
          ))}
        </div>
      </section>
    </>
  );
}
