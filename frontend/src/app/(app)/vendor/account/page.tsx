'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Toaster } from '@/components/ui/toast';
import { PageHeader } from '@/components/page-header';
import { StoreIcon, UserIcon, WalletIcon, ChevronRightIcon, CheckIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';
import { findPayoutOperator, maskPhone } from '@/lib/operators';
import { useToast } from '@/hooks/use-toast';

interface PayoutMethod {
  phone: string | null;
  operator: string | null;
  operator_label: string;
  country: string | null;
}

export default function VendorAccountPage() {
  const { user, refresh, logout } = useAuth();
  const router = useRouter();
  const { toasts, toast, dismiss } = useToast();

  const [shopName, setShopName] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [saving, setSaving] = useState(false);
  const [payoutMethod, setPayoutMethod] = useState<PayoutMethod | null>(null);

  useEffect(() => {
    setDisplayName(user?.display_name || '');
    setShopName(user?.shop_name || '');
  }, [user]);

  useEffect(() => {
    api.getPayoutMethod().then((r) => setPayoutMethod(r.payout_method)).catch(() => {});
  }, []);

  const handleSaveShop = async () => {
    setSaving(true);
    try {
      await api.updateProfile({ display_name: displayName || undefined, shop_name: shopName || undefined });
      await refresh();
      toast({ variant: 'success', title: 'Boutique mise à jour' });
    } catch (err: any) {
      toast({ variant: 'error', title: 'Échec', description: friendlyError(err) });
    } finally {
      setSaving(false);
    }
  };

  const handleLogout = async () => {
    await logout();
    router.push('/');
  };

  const shopTitle = user?.shop_name || user?.display_name || 'Ma boutique';

  return (
    <main className="min-h-screen bg-[#EDF8F2] pb-10">
      <PageHeader back="/vendor" eyebrow="// espace vendeur" title="Compte" />

      <div className="max-w-lg mx-auto px-4 sm:px-6 -mt-10 relative z-10 text-center">
        <div
          className="w-[78px] h-[78px] rounded-[22px] mx-auto flex items-center justify-center shadow-lift border-4 border-white"
          style={{ background: 'linear-gradient(160deg,#12A06B,#0A4F35)' }}
        >
          <StoreIcon size={32} className="text-[#C9F22E]" />
        </div>
        <h3 className="mt-3 text-[16.5px] font-extrabold text-[#0B2318]">{shopTitle}</h3>
        {user && (
          <p className="font-mono text-xs text-[#4C6459] mt-1">/boutique?id={user.id}</p>
        )}
        <span className="inline-flex items-center gap-1.5 mt-2 bg-[#EDF8F2] text-[#0E6B46] text-[10.5px] font-extrabold px-2.5 py-1 rounded-full font-mono">
          <CheckIcon size={11} />
          Vendeur actif
        </span>
      </div>

      <div className="max-w-lg mx-auto px-4 sm:px-6 mt-6 space-y-5">
        <div>
          <p className="text-[11px] font-extrabold text-[#4C6459] uppercase tracking-wider mb-2">Boutique</p>
          <div className="bg-white rounded-2xl border border-green-900/10 p-4 space-y-3">
            <div className="space-y-1.5">
              <Label htmlFor="shop-name">Nom de boutique</Label>
              <Input id="shop-name" value={shopName} onChange={(e) => setShopName(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="display-name">Nom affiché</Label>
              <Input id="display-name" value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
            </div>
            <Button size="sm" onClick={handleSaveShop} disabled={saving} className="w-full">
              {saving ? 'Enregistrement…' : 'Enregistrer'}
            </Button>
          </div>
        </div>

        <div>
          <p className="text-[11px] font-extrabold text-[#4C6459] uppercase tracking-wider mb-2">Paiement</p>
          <Link
            href="/vendor/earnings"
            className="bg-white rounded-2xl border border-green-900/10 p-3.5 flex items-center gap-3"
          >
            <span className="w-9 h-9 rounded-xl bg-[#EDF8F2] flex items-center justify-center shrink-0 text-[#0A4F35]">
              <WalletIcon size={16} />
            </span>
            <div className="min-w-0 flex-1">
              <p className="text-[13px] font-semibold text-[#0B2318]">Moyen de versement</p>
              <p className="text-xs text-[#4C6459] mt-0.5">
                {payoutMethod?.operator
                  ? `${payoutMethod.operator_label} · ${maskPhone(payoutMethod.phone, findPayoutOperator(payoutMethod.operator)?.dialCode) || payoutMethod.phone}`
                  : 'Non configuré'}
              </p>
            </div>
            <ChevronRightIcon size={16} className="text-[#4C6459] shrink-0" />
          </Link>
        </div>

        <div>
          <p className="text-[11px] font-extrabold text-[#4C6459] uppercase tracking-wider mb-2">Espace client</p>
          <Link href="/dashboard" className="bg-white rounded-2xl border border-green-900/10 p-3.5 flex items-center gap-3">
            <span className="w-9 h-9 rounded-xl bg-[#EDF8F2] flex items-center justify-center shrink-0 text-[#0A4F35]">
              <UserIcon size={16} />
            </span>
            <div className="min-w-0 flex-1">
              <p className="text-[13px] font-semibold text-[#0B2318]">Mode vendeur</p>
              <p className="text-xs text-[#4C6459] mt-0.5">Basculer vers l&apos;espace client</p>
            </div>
            <ChevronRightIcon size={16} className="text-[#4C6459] shrink-0" />
          </Link>
        </div>

        <button
          type="button"
          onClick={handleLogout}
          className="block w-full text-center mt-2 text-[#E4573D] font-extrabold text-xs py-2"
        >
          Se déconnecter
        </button>
      </div>

      <Toaster toasts={toasts} onDismiss={dismiss} />
    </main>
  );
}
