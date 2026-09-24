'use client';

import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { CheckIcon } from '@/components/icons';
import { api } from '@/lib/api';
import { friendlyError } from '@/lib/error-messages';
import { getPushPermissionState, pushSupported, subscribeToPush, unsubscribeFromPush, type PushPermissionState } from '@/lib/push';
import { useToast } from '@/hooks/use-toast';

// NotificationSettings — activation volontaire des notifications push
// (jamais de demande de permission automatique au chargement, mauvaise
// pratique UX). Affiché sur /account, accessible à tout compte connecté.
export function NotificationSettings() {
  const { toast } = useToast();
  const [permission, setPermission] = useState<PushPermissionState>('default');
  const [subscribed, setSubscribed] = useState(false);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!pushSupported()) {
      setPermission('unsupported');
      return;
    }
    setPermission(getPushPermissionState());
    navigator.serviceWorker.getRegistration().then(async (reg) => {
      const sub = await reg?.pushManager.getSubscription();
      setSubscribed(Boolean(sub));
    });
  }, []);

  if (permission === 'unsupported') return null;

  const handleEnable = async () => {
    setLoading(true);
    try {
      const subscription = await subscribeToPush();
      if (!subscription) {
        setPermission(getPushPermissionState());
        toast({ variant: 'error', title: 'Notifications refusées', description: "Vous pouvez les autoriser depuis les réglages de votre navigateur." });
        return;
      }
      await api.subscribeToPushNotifications(subscription.toJSON() as any);
      setPermission('granted');
      setSubscribed(true);
      toast({ variant: 'success', title: 'Notifications activées' });
    } catch (err: any) {
      toast({ variant: 'error', title: 'Échec de l’activation', description: friendlyError(err) });
    } finally {
      setLoading(false);
    }
  };

  const handleDisable = async () => {
    setLoading(true);
    try {
      const reg = await navigator.serviceWorker.getRegistration();
      const sub = await reg?.pushManager.getSubscription();
      if (sub) {
        await api.unsubscribeFromPushNotifications(sub.endpoint).catch(() => {});
      }
      await unsubscribeFromPush();
      setSubscribed(false);
      toast({ variant: 'success', title: 'Notifications désactivées' });
    } catch (err: any) {
      toast({ variant: 'error', title: 'Échec de la désactivation', description: friendlyError(err) });
    } finally {
      setLoading(false);
    }
  };

  return (
    <Card className="border-green-900/5">
      <CardHeader className="pb-3">
        <CardTitle className="text-lg flex items-center gap-2">
          Notifications
          {subscribed && (
            <Badge className="bg-green-100 text-green-700 hover:bg-green-100 gap-1">
              <CheckIcon size={11} />
              Activées
            </Badge>
          )}
        </CardTitle>
        <CardDescription>
          Recevez une alerte sur cet appareil pour vos ventes, messages et versements, même
          quand DIARRA n&apos;est pas ouvert.
        </CardDescription>
      </CardHeader>
      <CardContent>
        {subscribed ? (
          <Button variant="outline" className="h-10" onClick={handleDisable} disabled={loading}>
            {loading ? 'Désactivation…' : 'Désactiver les notifications'}
          </Button>
        ) : permission === 'denied' ? (
          <p className="text-sm text-muted-foreground">
            Vous avez bloqué les notifications pour DIARRA. Autorisez-les depuis les réglages de
            votre navigateur pour les activer.
          </p>
        ) : (
          <Button className="h-10" onClick={handleEnable} disabled={loading}>
            {loading ? 'Activation…' : 'Activer les notifications'}
          </Button>
        )}
      </CardContent>
    </Card>
  );
}
