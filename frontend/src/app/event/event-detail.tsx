'use client';

import { useEffect, useState } from 'react';
import { useSearchParams } from 'next/navigation';
import Link from 'next/link';
import { api, apiOrigin } from '@/lib/api';
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
  logo_key?: string | null;
  accent_color?: string | null;
  venue_name?: string | null;
  venue_address?: string | null;
  venue_map_url?: string | null;
  about_organizer?: string | null;
}

interface Offer {
  id: string;
  title: string;
  is_free: boolean;
  product_id?: string | null;
  product_slug?: string | null;
  product_price_cfa?: number | null;
}

interface GalleryImage {
  id: string;
}

interface ScheduleItem {
  id: string;
  time_label: string;
  title: string;
  description?: string | null;
}

interface HeroImage {
  id: string;
}

interface OtherEvent {
  id: string;
  slug: string;
  title: string;
  cover_image_key?: string | null;
}

interface OtherProduct {
  id: string;
  slug?: string | null;
  title: string;
  price_cfa: number;
  cover_image_key?: string | null;
}

// Carrousel du bandeau hero : défilement automatique (5 s) entre les images
// dédiées uploadées par le vendeur (distinctes de la galerie photo).
function HeroCarousel({ images }: { images: HeroImage[] }) {
  const [active, setActive] = useState(0);

  useEffect(() => {
    if (images.length <= 1) return;
    const timer = setInterval(() => setActive((i) => (i + 1) % images.length), 5000);
    return () => clearInterval(timer);
  }, [images.length]);

  if (images.length === 0) return null;

  return (
    <div className="absolute inset-0">
      {images.map((img, i) => (
        <img
          key={img.id}
          src={`${apiOrigin}/api/events/hero/${img.id}/file`}
          alt=""
          aria-hidden
          className="absolute inset-0 w-full h-full object-cover opacity-30 transition-opacity duration-700"
          style={{ opacity: i === active ? 0.3 : 0 }}
        />
      ))}
      {images.length > 1 && (
        <div className="absolute bottom-3 left-1/2 -translate-x-1/2 flex gap-1.5 z-10">
          {images.map((img, i) => (
            <button
              key={img.id}
              type="button"
              aria-label={`Image ${i + 1}`}
              onClick={() => setActive(i)}
              className={`w-1.5 h-1.5 rounded-full transition-colors ${i === active ? 'bg-white' : 'bg-white/40'}`}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function FreeOfferForm({ offerId, accent }: { offerId: string; accent: string }) {
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
        style={{ backgroundColor: accent }}
        className="w-full h-10 rounded-md text-white text-sm font-semibold transition-opacity hover:opacity-90 disabled:opacity-60"
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
  const [gallery, setGallery] = useState<GalleryImage[]>([]);
  const [schedule, setSchedule] = useState<ScheduleItem[]>([]);
  const [heroImages, setHeroImages] = useState<HeroImage[]>([]);
  const [organizerName, setOrganizerName] = useState('');
  const [otherEvents, setOtherEvents] = useState<OtherEvent[]>([]);
  const [otherProducts, setOtherProducts] = useState<OtherProduct[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!id) return;
    api
      .getEvent(id)
      .then((res) => {
        setEvent(res.event);
        setOffers(res.offers || []);
        setGallery(res.gallery || []);
        setSchedule(res.schedule || []);
        setHeroImages(res.hero_images || []);
        setOrganizerName(res.organizer_name || '');
        setOtherEvents(res.other_events || []);
        setOtherProducts(res.other_products || []);
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

  const accent = event.accent_color || '#0E6B46';
  const hasVenue = event.venue_name || event.venue_address;

  return (
    <>
      <section
        className="text-white relative overflow-hidden"
        style={event.accent_color ? { background: `linear-gradient(135deg, ${accent}, #052018)` } : undefined}
      >
        <div
          className={event.accent_color ? undefined : 'gradient-green absolute inset-0'}
          aria-hidden
        />
        <div className="wax-pattern absolute inset-0" aria-hidden />
        {heroImages.length > 0 ? (
          <HeroCarousel images={heroImages} />
        ) : (
          event.cover_image_key && (
            <img
              src={`${apiOrigin}/api/events/${event.id}/cover`}
              alt=""
              aria-hidden
              className="absolute inset-0 w-full h-full object-cover opacity-30"
            />
          )
        )}
        <div className="relative max-w-3xl mx-auto px-4 py-16 text-center">
          {event.logo_key && (
            <img
              src={`${apiOrigin}/api/events/${event.id}/logo`}
              alt={event.title}
              className="h-14 sm:h-16 object-contain mx-auto mb-6"
            />
          )}
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
          {hasVenue && (
            <p className="mt-4 text-sm text-white/80">
              📍 {event.venue_name}
              {event.venue_name && event.venue_address ? ' — ' : ''}
              {event.venue_address}
              {event.venue_map_url && (
                <>
                  {' '}
                  <a href={event.venue_map_url} target="_blank" rel="noopener noreferrer" className="underline">
                    Voir sur la carte
                  </a>
                </>
              )}
            </p>
          )}
        </div>
      </section>

      {schedule.length > 0 && (
        <section className="py-16 max-w-3xl mx-auto px-4">
          <h2 className="font-display text-2xl font-bold text-green-950 text-center mb-8">Programme</h2>
          <div className="space-y-4">
            {schedule.map((item) => (
              <div key={item.id} className="flex gap-4 rounded-xl border border-green-900/10 bg-white shadow-card p-4">
                <div className="font-mono text-sm font-bold shrink-0 w-20" style={{ color: accent }}>
                  {item.time_label}
                </div>
                <div className="min-w-0">
                  <p className="font-semibold text-green-950">{item.title}</p>
                  {item.description && <p className="text-sm text-green-900/70 mt-1">{item.description}</p>}
                </div>
              </div>
            ))}
          </div>
        </section>
      )}

      {gallery.length > 0 && (
        <section className="pb-16 max-w-4xl mx-auto px-4">
          <h2 className="font-display text-2xl font-bold text-green-950 text-center mb-8">Galerie</h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
            {gallery.map((img) => (
              <img
                key={img.id}
                src={`${apiOrigin}/api/events/gallery/${img.id}/file`}
                alt=""
                className="w-full h-32 sm:h-40 object-cover rounded-xl border border-green-900/10"
              />
            ))}
          </div>
        </section>
      )}

      <section className="py-16 max-w-4xl mx-auto px-4">
        <h2 className="font-display text-2xl font-bold text-green-950 text-center mb-8">Offres</h2>
        <div className="grid sm:grid-cols-2 gap-6" style={{ gridTemplateColumns: offers.length === 1 ? '1fr' : undefined }}>
          {offers.map((offer) => (
            <div key={offer.id} className="rounded-2xl border border-green-900/10 bg-white shadow-lift p-6">
              <h3 className="font-display text-lg font-bold text-green-950">{offer.title}</h3>
              <p className="mt-1 text-2xl font-bold" style={{ color: accent }}>
                {offer.is_free ? 'Gratuit' : formatPrice(offer.product_price_cfa || 0)}
              </p>
              {offer.is_free ? (
                <FreeOfferForm offerId={offer.id} accent={accent} />
              ) : (
                <Link
                  href={`/checkout?product=${offer.product_slug || offer.product_id}`}
                  style={{ backgroundColor: accent }}
                  className="mt-4 h-11 rounded-md text-white text-sm font-semibold flex items-center justify-center transition-opacity hover:opacity-90"
                >
                  Choisir cette offre
                </Link>
              )}
            </div>
          ))}
        </div>
      </section>

      {event.about_organizer && (
        <section className="pb-16 max-w-2xl mx-auto px-4 text-center">
          <h2 className="font-display text-2xl font-bold text-green-950 mb-4">À propos de l&rsquo;organisateur</h2>
          {organizerName && <p className="font-semibold text-green-950 mb-2">{organizerName}</p>}
          <p className="text-green-900/70 whitespace-pre-line">{event.about_organizer}</p>
        </section>
      )}

      {(otherEvents.length > 0 || otherProducts.length > 0) && (
        <section className="pb-16 max-w-4xl mx-auto px-4">
          <h2 className="font-display text-2xl font-bold text-green-950 text-center mb-8">
            Autres offres {organizerName ? `de ${organizerName}` : "de l'organisateur"}
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
            {otherEvents.map((e) => (
              <Link
                key={e.id}
                href={`/event?id=${e.slug}`}
                className="rounded-xl border border-green-900/10 bg-white shadow-card overflow-hidden hover:shadow-lift transition-shadow"
              >
                {e.cover_image_key && (
                  <img
                    src={`${apiOrigin}/api/events/${e.id}/cover`}
                    alt=""
                    className="w-full h-24 sm:h-32 object-cover"
                  />
                )}
                <p className="p-3 text-sm font-medium text-green-950 truncate">{e.title}</p>
              </Link>
            ))}
            {otherProducts.map((p) => (
              <Link
                key={p.id}
                href={`/product?id=${p.slug || p.id}`}
                className="rounded-xl border border-green-900/10 bg-white shadow-card overflow-hidden hover:shadow-lift transition-shadow"
              >
                {p.cover_image_key && (
                  <img
                    src={`${apiOrigin}/api/products/${p.id}/cover`}
                    alt=""
                    className="w-full h-24 sm:h-32 object-cover"
                  />
                )}
                <div className="p-3">
                  <p className="text-sm font-medium text-green-950 truncate">{p.title}</p>
                  <p className="text-xs text-green-900/60 font-mono mt-0.5">{formatPrice(p.price_cfa)}</p>
                </div>
              </Link>
            ))}
          </div>
        </section>
      )}
    </>
  );
}
