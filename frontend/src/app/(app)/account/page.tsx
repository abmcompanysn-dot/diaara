'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { api } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { RequireAuth } from '@/lib/guards';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Toaster } from '@/components/ui/toast';
import { PageHeader } from '@/components/page-header';
import { PayoutMethodForm } from '@/components/payout-method-form';
import { PhoneVerifyForm } from '@/components/phone-verify-form';
import { StoreIcon, CheckIcon, WalletIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';
import { findPayoutOperator, maskPhone } from '@/lib/operators';
import { useToast } from '@/hooks/use-toast';

interface PayoutMethod {
  active_channel?: 'mobile_money' | 'paypal';
  phone: string | null;
  operator: string | null;
  operator_label: string;
  country: string | null;
  paypal_email?: string | null;
}

export default function AccountPage() {
  const { user, refresh } = useAuth();
  const router = useRouter();
  const { toasts, toast, dismiss } = useToast();

  const isVendeur = user?.roles?.includes('vendeur') ?? false;

  const [shopName, setShopName] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [becomingVendor, setBecomingVendor] = useState(false);

  // Numéro du compte : certains comptes (inscription email seule) n'en ont
  // aucun et n'avaient jusqu'ici aucun moyen d'en ajouter un — ce qui bloquait
  // la vérification exigée avant tout versement.
  const [phone, setPhone] = useState('');
  const [phoneSaving, setPhoneSaving] = useState(false);
  const [phoneError, setPhoneError] = useState('');
  const [showPhoneVerify, setShowPhoneVerify] = useState(false);
  const phoneVerified = Boolean(user?.phone_verified_at);

  const [payoutMethod, setPayoutMethod] = useState<PayoutMethod | null>(null);
  const [editingMethod, setEditingMethod] = useState(false);
  const [methodSubmitting, setMethodSubmitting] = useState(false);
  const [methodError, setMethodError] = useState('');

  useEffect(() => {
    setDisplayName(user?.display_name || '');
    setShopName(user?.shop_name || '');
    setPhone(user?.phone || '');
  }, [user]);

  const handleSavePhone = async () => {
    const trimmed = phone.trim();
    setPhoneError('');
    if (!trimmed) {
      setPhoneError('Saisissez votre numéro de téléphone.');
      return;
    }
    setPhoneSaving(true);
    try {
      await api.updateProfile({ phone: trimmed });
      await refresh();
      // Changer de numéro remet phone_verified_at à NULL côté backend : on
      // enchaîne directement sur la vérification.
      setShowPhoneVerify(true);
      toast({ variant: 'success', title: 'Numéro enregistré', description: 'Vérifiez-le maintenant pour activer les versements.' });
    } catch (err: any) {
      // Message affiché en dur sous le champ (pas seulement en toast, qui
      // disparaît tout seul) : une erreur bloquante comme « numéro déjà pris
      // par un autre compte » doit rester visible tant qu'elle n'est pas
      // corrigée, sinon on obtient exactement le symptôme "je ne comprends
      // pas, rien ne se passe" alors qu'un message était bien envoyé.
      setPhoneError(friendlyError(err));
    } finally {
      setPhoneSaving(false);
    }
  };

  useEffect(() => {
    api
      .getAccountPayoutMethod()
      .then((r) => setPayoutMethod(r.payout_method))
      .catch(() => {});
  }, []);

  const hasPayoutMethod = Boolean(payoutMethod?.operator);

  const handleBecomeVendor = async () => {
    setBecomingVendor(true);
    try {
      await api.addRole('vendeur');
      if (displayName || shopName) {
        await api.updateProfile({ display_name: displayName || undefined, shop_name: shopName || undefined });
      }
      await refresh();
      toast({ variant: 'success', title: 'Bienvenue vendeur !', description: 'Votre espace vendeur est maintenant actif.' });
      router.push('/vendor/products');
    } catch (err: any) {
      toast({ variant: 'error', title: 'Impossible de devenir vendeur', description: friendlyError(err) });
    } finally {
      setBecomingVendor(false);
    }
  };

  const handleSaveMethod = async (data: { phone: string; operator: string; country: string }) => {
    setMethodError('');
    setMethodSubmitting(true);
    try {
      await api.setAccountPayoutMethod({ channel: 'mobile_money', ...data });
      const fresh = await api.getAccountPayoutMethod();
      setPayoutMethod(fresh.payout_method);
      setEditingMethod(false);
      toast({ variant: 'success', title: 'Moyen de retrait enregistré' });
    } catch (err: any) {
      setMethodError(friendlyError(err));
    } finally {
      setMethodSubmitting(false);
    }
  };

  return (
    <RequireAuth>
      <main>
        <PageHeader eyebrow="// mon compte" title="Mon compte" description="Profil, boutique et moyen de retrait" />

        <section className="max-w-3xl mx-auto px-4 sm:px-6 py-10 space-y-6">
          {!isVendeur && (
            <Card className="border-green-900/5">
              <CardHeader className="pb-3">
                <CardTitle className="text-lg flex items-center gap-2">
                  <StoreIcon size={18} className="text-green-700" />
                  Devenir vendeur
                </CardTitle>
                <CardDescription>Ouvrez votre boutique et commencez à vendre vos produits dès aujourd&apos;hui.</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="display-name">Votre nom</Label>
                  <Input
                    id="display-name"
                    value={displayName}
                    onChange={(e) => setDisplayName(e.target.value)}
                    placeholder="Ex : Awa Diarra"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="shop-name">Nom de la boutique</Label>
                  <Input
                    id="shop-name"
                    value={shopName}
                    onChange={(e) => setShopName(e.target.value)}
                    placeholder="Ex : Awa Digital Store"
                  />
                </div>
                <Button onClick={handleBecomeVendor} disabled={becomingVendor}>
                  {becomingVendor ? 'Activation…' : 'Devenir vendeur'}
                </Button>
              </CardContent>
            </Card>
          )}

          {isVendeur && (
            <Card className="border-green-900/5">
              <CardHeader className="pb-3 flex flex-row items-center justify-between">
                <div>
                  <CardTitle className="text-lg">Ma boutique</CardTitle>
                  <CardDescription>Le nom affiché sur vos fiches produit</CardDescription>
                </div>
                <Button
                  variant="outline"
                  className="h-9"
                  onClick={async () => {
                    try {
                      await api.updateProfile({ display_name: displayName || undefined, shop_name: shopName || undefined });
                      await refresh();
                      toast({ variant: 'success', title: 'Profil mis à jour' });
                    } catch (err: any) {
                      toast({ variant: 'error', title: 'Échec', description: friendlyError(err) });
                    }
                  }}
                >
                  Enregistrer
                </Button>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="display-name-2">Votre nom</Label>
                  <Input id="display-name-2" value={displayName} onChange={(e) => setDisplayName(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="shop-name-2">Nom de la boutique</Label>
                  <Input id="shop-name-2" value={shopName} onChange={(e) => setShopName(e.target.value)} />
                </div>
              </CardContent>
            </Card>
          )}

          <Card className="border-green-900/5">
            <CardHeader className="pb-3">
              <CardTitle className="text-lg flex items-center gap-2">
                Numéro de téléphone
                {user?.phone && phoneVerified && (
                  <Badge className="bg-green-100 text-green-700 hover:bg-green-100 gap-1">
                    <CheckIcon size={11} />
                    Vérifié
                  </Badge>
                )}
                {user?.phone && !phoneVerified && (
                  <Badge className="bg-yellow-100 text-yellow-800 hover:bg-yellow-100">Non vérifié</Badge>
                )}
              </CardTitle>
              <CardDescription>
                Requis et vérifié avant de pouvoir demander un versement. Un code de confirmation
                vous sera envoyé.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex flex-col sm:flex-row gap-2 sm:items-end">
                <div className="flex-1 space-y-2">
                  <Label htmlFor="account-phone">Numéro (avec l&apos;indicatif pays)</Label>
                  <Input
                    id="account-phone"
                    type="tel"
                    inputMode="tel"
                    value={phone}
                    onChange={(e) => {
                      setPhone(e.target.value);
                      if (phoneError) setPhoneError('');
                    }}
                    placeholder="+221 77 123 45 67"
                  />
                </div>
                <Button
                  variant="outline"
                  className="h-10"
                  onClick={handleSavePhone}
                  disabled={phoneSaving || phone.trim() === (user?.phone || '')}
                >
                  {phoneSaving ? 'Enregistrement…' : 'Enregistrer'}
                </Button>
              </div>

              {phoneError && (
                <div className="p-3 bg-destructive/10 text-destructive rounded-lg text-sm" role="alert">
                  {phoneError}
                </div>
              )}

              {user?.phone && !phoneVerified && !showPhoneVerify && (
                <Button className="h-10" onClick={() => setShowPhoneVerify(true)}>
                  Vérifier ce numéro
                </Button>
              )}

              {showPhoneVerify && user?.phone && !phoneVerified && (
                <div className="rounded-lg border border-green-900/10 p-4 bg-green-50/40">
                  <PhoneVerifyForm
                    phone={user.phone}
                    onVerified={async () => {
                      await refresh();
                      setShowPhoneVerify(false);
                      toast({ variant: 'success', title: 'Numéro vérifié', description: 'Vous pouvez maintenant demander un versement.' });
                    }}
                    onSkip={() => setShowPhoneVerify(false)}
                    skipLabel="Fermer"
                  />
                </div>
              )}
            </CardContent>
          </Card>

          <Card className="border-green-900/5">
            <CardHeader className="pb-3 flex flex-row items-center justify-between">
              <div>
                <CardTitle className="text-lg">Moyen de retrait</CardTitle>
                <CardDescription>Le compte mobile money utilisé pour vous envoyer de l&apos;argent (ex : un remboursement)</CardDescription>
              </div>
              {!editingMethod && (
                <Button variant="outline" className="h-9" onClick={() => { setMethodError(''); setEditingMethod(true); }}>
                  {hasPayoutMethod ? 'Modifier' : 'Ajouter'}
                </Button>
              )}
            </CardHeader>
            <CardContent>
              {!editingMethod ? (
                hasPayoutMethod ? (
                  <div className="flex items-center gap-3 flex-wrap">
                    <Badge className="bg-green-100 text-green-700 hover:bg-green-100">{payoutMethod!.operator_label}</Badge>
                    <span className="font-mono text-sm text-green-900/70">
                      {maskPhone(payoutMethod!.phone, findPayoutOperator(payoutMethod!.operator)?.dialCode) || payoutMethod!.phone}
                    </span>
                    <Badge className="bg-green-100 text-green-700 hover:bg-green-100 gap-1">
                      <CheckIcon size={11} />
                      Vérifié
                    </Badge>
                  </div>
                ) : (
                  <div className="flex flex-col items-start gap-3 py-2">
                    <div className="w-10 h-10 rounded-full bg-yellow-100 flex items-center justify-center">
                      <WalletIcon size={18} className="text-yellow-700" />
                    </div>
                    <p className="text-sm text-muted-foreground">
                      Aucun moyen de retrait enregistré.
                    </p>
                  </div>
                )
              ) : (
                <PayoutMethodForm
                  initialCountry={payoutMethod?.country}
                  initialOperator={payoutMethod?.operator}
                  initialPhone={payoutMethod?.phone}
                  onSave={handleSaveMethod}
                  onCancel={() => setEditingMethod(false)}
                  saving={methodSubmitting}
                  error={methodError}
                />
              )}
            </CardContent>
          </Card>

          <Button variant="outline" className="h-10" render={<Link href="/dashboard" />}>
            ← Mon espace
          </Button>
        </section>

        <Toaster toasts={toasts} onDismiss={dismiss} />
      </main>
    </RequireAuth>
  );
}
