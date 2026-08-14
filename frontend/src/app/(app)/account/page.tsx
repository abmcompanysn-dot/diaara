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
import { StoreIcon, CheckIcon, WalletIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';
import { findPayoutOperator, maskPhone } from '@/lib/operators';
import { useToast } from '@/hooks/use-toast';

interface PayoutMethod {
  phone: string | null;
  operator: string | null;
  operator_label: string;
  country: string | null;
}

export default function AccountPage() {
  const { user, refresh } = useAuth();
  const router = useRouter();
  const { toasts, toast, dismiss } = useToast();

  const isVendeur = user?.roles?.includes('vendeur') ?? false;

  const [shopName, setShopName] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [becomingVendor, setBecomingVendor] = useState(false);

  const [payoutMethod, setPayoutMethod] = useState<PayoutMethod | null>(null);
  const [editingMethod, setEditingMethod] = useState(false);
  const [methodSubmitting, setMethodSubmitting] = useState(false);
  const [methodError, setMethodError] = useState('');

  useEffect(() => {
    setDisplayName(user?.display_name || '');
    setShopName(user?.shop_name || '');
  }, [user]);

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
      const result = await api.setAccountPayoutMethod(data);
      setPayoutMethod(result.payout_method);
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
                  size="sm"
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
            <CardHeader className="pb-3 flex flex-row items-center justify-between">
              <div>
                <CardTitle className="text-lg">Moyen de retrait</CardTitle>
                <CardDescription>Le compte mobile money utilisé pour vous envoyer de l&apos;argent (ex : un remboursement)</CardDescription>
              </div>
              {!editingMethod && (
                <Button variant="outline" size="sm" onClick={() => { setMethodError(''); setEditingMethod(true); }}>
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

          <Button variant="outline" size="sm" render={<Link href="/dashboard" />}>
            ← Mon espace
          </Button>
        </section>

        <Toaster toasts={toasts} onDismiss={dismiss} />
      </main>
    </RequireAuth>
  );
}
