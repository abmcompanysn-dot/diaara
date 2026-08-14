'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { formatPrice } from '@/lib/constants';
import { PageLoader } from '@/components/page-loader';
import { ShopShareCard } from '@/components/shop-share-card';
import {
  StoreIcon,
  PackageIcon,
  WalletIcon,
  UserIcon,
  MessageIcon,
  LinkIcon,
  BellIcon,
  GridIcon,
  CheckIcon,
} from '@/components/icons';

interface Notification {
  id: string;
  type: string;
  title: string;
  body?: string;
  link?: string;
  read_at: string | null;
  created_at: string;
}

const SHORTCUTS = [
  { href: '/vendor/products', label: 'Produits', Icon: StoreIcon, bg: '#DFF3E7', fg: '#0A4F35' },
  { href: '/vendor/products/bundles', label: 'Packs', Icon: PackageIcon, bg: '#FFF3D6', fg: '#B8860B' },
  { href: '/vendor/earnings', label: 'Revenus', Icon: WalletIcon, bg: '#E4F0FF', fg: '#1F5FBF' },
  { href: '/vendor/sales', label: 'Clients', Icon: UserIcon, bg: '#FBE4EE', fg: '#B23B72' },
  { href: '/vendor/messages', label: 'Messages', Icon: MessageIcon, bg: '#E9E4FB', fg: '#5A3FBF' },
  { href: '/vendor/affiliation', label: 'Affiliation', Icon: LinkIcon, bg: '#DDF7EF', fg: '#0E9E6D' },
  { href: '/vendor/account', label: 'Compte', Icon: UserIcon, bg: '#FBEADD', fg: '#C0641B' },
  { href: '/vendor/notifications', label: 'Notifications', Icon: BellIcon, bg: '#EDEFF1', fg: '#4C6459' },
];

export default function VendorHomePage() {
  const { user } = useAuth();
  const [available, setAvailable] = useState(0);
  const [hideBalance, setHideBalance] = useState(false);
  const [activity, setActivity] = useState<Notification[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    (async () => {
      try {
        const [earnings, notifs] = await Promise.all([api.getVendorEarnings(), api.getNotifications()]);
        setAvailable(earnings.available);
        setActivity(notifs.notifications.slice(0, 5));
      } catch {
        // Le tableau de bord reste utilisable même si une des deux requêtes échoue.
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  if (loading || !user) return <PageLoader />;

  const shopTitle = user.shop_name || user.display_name || 'Ma boutique';

  return (
    <main className="min-h-screen bg-[#EDF8F2]">
      <div className="gradient-green text-white relative overflow-hidden pb-9 px-4 sm:px-6 pt-6">
        <div className="wax-pattern absolute inset-0 opacity-70" aria-hidden />
        <div className="relative max-w-2xl mx-auto">
          <div className="flex items-center justify-between">
            <span className="text-sm font-semibold opacity-0" aria-hidden>·</span>
            <Link
              href="/vendor/notifications"
              aria-label="Notifications"
              className="relative w-9 h-9 rounded-xl bg-white/15 border border-white/20 flex items-center justify-center hover:bg-white/25 transition-colors"
            >
              <BellIcon size={17} />
              {activity.some((n) => !n.read_at) && (
                <span className="absolute top-1.5 right-1.5 w-2 h-2 rounded-full bg-[#C9F22E] border border-[#0A4F35]" />
              )}
            </Link>
          </div>

          <div className="text-center mt-2">
            <p className="text-white font-bold text-[15px]">{shopTitle}</p>
          </div>

          <div className="text-center mt-4">
            <p className="text-white/70 text-xs font-semibold">Solde disponible</p>
            <div className="flex items-baseline justify-center gap-2 mt-1">
              <p className="font-mono text-4xl font-extrabold tracking-tight">
                {hideBalance ? '••••' : available.toLocaleString('fr-FR')}{' '}
                <span className="text-base font-bold text-[#C9F22E]">F</span>
              </p>
              <button
                type="button"
                onClick={() => setHideBalance((v) => !v)}
                aria-label={hideBalance ? 'Afficher le solde' : 'Masquer le solde'}
                className="opacity-75 hover:opacity-100 transition-opacity"
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
                  <path d="M1 12s4-7 11-7 11 7 11 7-4 7-11 7-11-7-11-7z" stroke="#fff" strokeWidth="1.8" />
                  <circle cx="12" cy="12" r="3" stroke="#fff" strokeWidth="1.8" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-2xl mx-auto px-4 sm:px-6 -mt-5 relative z-10">
        <ShopShareCard vendorId={user.id} shopTitle={shopTitle} />
      </div>

      <div className="max-w-2xl mx-auto px-4 sm:px-6">
        <div className="grid grid-cols-4 gap-y-4 gap-x-1.5 py-6">
          {SHORTCUTS.map(({ href, label, Icon, bg, fg }) => (
            <Link key={href} href={href} className="flex flex-col items-center gap-1.5 text-center">
              <span
                className="w-13 h-13 rounded-2xl flex items-center justify-center"
                style={{ width: 52, height: 52, background: bg, color: fg }}
              >
                <Icon size={22} />
              </span>
              <span className="text-[11px] font-semibold text-[#0B2318] leading-tight">{label}</span>
            </Link>
          ))}
        </div>
      </div>

      <div className="bg-white rounded-t-[20px] pt-5 pb-10 px-4 sm:px-6">
        <div className="max-w-2xl mx-auto">
          <div className="flex items-center justify-between mb-3">
            <h2 className="font-bold text-[15px] text-[#0B2318]">Activité récente</h2>
            <Link href="/vendor/notifications" className="font-mono text-xs font-bold text-[#0E6B46]">
              Tout voir
            </Link>
          </div>

          {activity.length === 0 ? (
            <p className="text-sm text-[#4C6459] py-6 text-center flex flex-col items-center gap-2">
              <GridIcon size={20} className="opacity-40" />
              Aucune activité pour le moment.
            </p>
          ) : (
            activity.map((n) => (
              <div key={n.id} className="flex items-center gap-3 py-2.5 border-b border-[#DCEAE2] last:border-none">
                <span className="w-10 h-10 rounded-xl bg-[#EDF8F2] flex items-center justify-center shrink-0 text-[#0A4F35]">
                  <CheckIcon size={17} />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="text-[13.5px] font-semibold text-[#0B2318] truncate">{n.title}</p>
                  <p className="text-[11.5px] text-[#4C6459] mt-0.5">
                    {new Date(n.created_at).toLocaleString('fr-FR', { dateStyle: 'short', timeStyle: 'short' })}
                  </p>
                </div>
                {!n.read_at && (
                  <span className="text-[10px] font-bold text-[#8A6300] bg-[#FFF3D6] px-2 py-0.5 rounded-full shrink-0">
                    Nouveau
                  </span>
                )}
              </div>
            ))
          )}
        </div>
      </div>
    </main>
  );
}
