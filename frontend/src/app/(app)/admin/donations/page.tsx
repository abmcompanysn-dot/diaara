'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Checkbox } from '@/components/ui/checkbox';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { PayoutMethodForm } from '@/components/payout-method-form';
import { ArrowLeftIcon, CheckIcon, TrashIcon } from '@/components/icons';
import { formatPrice } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';

interface DonationRecipient {
  id: string;
  name: string;
  phone_number: string;
  operator: string;
  country: string;
  active: boolean;
}

interface DonationPayout {
  id: string;
  recipient_name: string;
  recipient_phone: string;
  amount_cfa: number;
  status: string;
  failure_reason?: string;
  requested_at: string;
}

const STATUS_LABELS: Record<string, string> = {
  requested: 'En attente',
  processing: 'En cours',
  paid: 'Versé',
  failed: 'Échoué',
};

const STATUS_COLOR: Record<string, string> = {
  requested: 'bg-yellow-100 text-yellow-700',
  processing: 'bg-blue-100 text-blue-700',
  paid: 'bg-green-100 text-green-700',
  failed: 'bg-red-100 text-red-700',
};

export default function AdminDonationsPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [saved, setSaved] = useState(false);

  const [poolBalance, setPoolBalance] = useState(0);
  const [recipients, setRecipients] = useState<DonationRecipient[]>([]);
  const [payouts, setPayouts] = useState<DonationPayout[]>([]);

  const [sharePct, setSharePct] = useState('80');
  const [thresholdCfa, setThresholdCfa] = useState('100000');
  const [enabled, setEnabled] = useState(true);
  const [savingSettings, setSavingSettings] = useState(false);

  const [addingRecipient, setAddingRecipient] = useState(false);
  const [savingRecipient, setSavingRecipient] = useState(false);
  const [recipientName, setRecipientName] = useState('');
  const [recipientError, setRecipientError] = useState('');
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);
  const [retryingId, setRetryingId] = useState<string | null>(null);

  const load = () => {
    api
      .getDonations()
      .then((r) => {
        setPoolBalance(r.pool.balance_cfa);
        setRecipients(r.recipients);
        setPayouts(r.payouts);
        setSharePct(String(r.settings.share_pct));
        setThresholdCfa(String(r.settings.threshold_cfa));
        setEnabled(r.settings.enabled);
      })
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const handleSaveSettings = async () => {
    setError('');
    setSaved(false);
    const pct = parseFloat(sharePct);
    const threshold = parseFloat(thresholdCfa);
    if (isNaN(pct) || pct < 0 || pct > 100) {
      setError('Le pourcentage doit être entre 0 et 100.');
      return;
    }
    if (isNaN(threshold) || threshold < 0) {
      setError('Le seuil doit être un nombre positif.');
      return;
    }
    setSavingSettings(true);
    try {
      await api.updateAdminSettings({
        donation_share_pct: String(pct),
        donation_threshold_cfa: String(threshold),
        donation_program_enabled: String(enabled),
      });
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSavingSettings(false);
    }
  };

  const handleAddRecipient = async (data: { phone: string; operator: string; country: string }) => {
    if (!recipientName.trim()) {
      setRecipientError('Le nom du destinataire est requis.');
      return;
    }
    setRecipientError('');
    setSavingRecipient(true);
    try {
      await api.createDonationRecipient({ name: recipientName.trim(), ...data });
      setRecipientName('');
      setAddingRecipient(false);
      load();
    } catch (err: any) {
      setRecipientError(friendlyError(err));
    } finally {
      setSavingRecipient(false);
    }
  };

  const handleToggleActive = async (recipient: DonationRecipient) => {
    try {
      await api.updateDonationRecipient(recipient.id, { active: !recipient.active });
      load();
    } catch (err: any) {
      setError(friendlyError(err));
    }
  };

  const handleDelete = async () => {
    if (!confirmDeleteId) return;
    try {
      await api.deleteDonationRecipient(confirmDeleteId);
      setConfirmDeleteId(null);
      load();
    } catch (err: any) {
      setError(friendlyError(err));
      setConfirmDeleteId(null);
    }
  };

  const handleRetry = async (payoutId: string) => {
    setRetryingId(payoutId);
    setError('');
    try {
      await api.retryDonationPayout(payoutId);
      load();
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setRetryingId(null);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// administration" title="Fidélisation" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Fidélisation"
        description="Reversement automatique d'une part de la commission à des associations/personnes"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/admin" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Tableau de bord
          </Button>
        }
      />

      <section className="max-w-2xl mx-auto px-4 sm:px-6 py-10 space-y-6">
        {error && (
          <div className="p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}
        {saved && (
          <div className="p-3 bg-green-50 text-green-700 rounded text-sm flex items-center gap-2" role="status">
            <CheckIcon size={16} /> Réglages enregistrés.
          </div>
        )}

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Cagnotte actuelle</CardTitle>
            <CardDescription>
              Alimentée automatiquement à chaque vente payée. Distribuée dès qu'elle atteint le seuil ci-dessous.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <p className="font-display text-3xl font-bold text-green-700">{formatPrice(poolBalance)}</p>
          </CardContent>
        </Card>

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Réglages</CardTitle>
            <CardDescription>
              Le programme reste actif même désactivé temporairement ci-dessous : la cagnotte continue de
              s'alimenter, seule la distribution automatique s'arrête.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <label className="flex items-center justify-between p-3 rounded-lg border border-border cursor-pointer">
              <span className="text-sm font-medium">Programme actif</span>
              <Checkbox checked={enabled} onCheckedChange={(c) => setEnabled(c === true)} />
            </label>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-2">
                <Label htmlFor="share-pct">Part reversée (%)</Label>
                <Input
                  id="share-pct"
                  type="number"
                  min={0}
                  max={100}
                  step="0.1"
                  value={sharePct}
                  onChange={(e) => setSharePct(e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="threshold">Seuil de déclenchement (FCFA)</Label>
                <Input
                  id="threshold"
                  type="number"
                  min={0}
                  step="1000"
                  value={thresholdCfa}
                  onChange={(e) => setThresholdCfa(e.target.value)}
                />
              </div>
            </div>
            <Button onClick={handleSaveSettings} disabled={savingSettings} className="w-full font-semibold">
              {savingSettings ? 'Enregistrement...' : 'Enregistrer les réglages'}
            </Button>
          </CardContent>
        </Card>

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Destinataires</CardTitle>
            <CardDescription>
              Répartition à parts égales entre tous les destinataires actifs à chaque distribution.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {recipients.map((rec) => (
              <div
                key={rec.id}
                className="flex items-center justify-between p-3 rounded-lg border border-border gap-3"
              >
                <div className="min-w-0">
                  <p className="text-sm font-semibold truncate">{rec.name}</p>
                  <p className="text-xs text-muted-foreground font-mono">{rec.phone_number}</p>
                </div>
                <div className="flex items-center gap-3 shrink-0">
                  <label className="flex items-center gap-1.5 text-xs cursor-pointer">
                    <Checkbox checked={rec.active} onCheckedChange={() => handleToggleActive(rec)} />
                    Actif
                  </label>
                  <button
                    type="button"
                    onClick={() => setConfirmDeleteId(rec.id)}
                    className="text-red-600 hover:text-red-700"
                    aria-label={`Supprimer ${rec.name}`}
                  >
                    <TrashIcon size={16} />
                  </button>
                </div>
              </div>
            ))}
            {recipients.length === 0 && !addingRecipient && (
              <p className="text-sm text-muted-foreground italic">Aucun destinataire — la cagnotte s'accumule sans être distribuée.</p>
            )}

            {addingRecipient ? (
              <div className="pt-3 border-t border-border space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="recipient-name">Nom (association, personne...)</Label>
                  <Input
                    id="recipient-name"
                    value={recipientName}
                    onChange={(e) => setRecipientName(e.target.value)}
                    placeholder="Ex: Association Espoir"
                  />
                </div>
                <PayoutMethodForm
                  onSave={handleAddRecipient}
                  onCancel={() => {
                    setAddingRecipient(false);
                    setRecipientError('');
                  }}
                  saving={savingRecipient}
                  error={recipientError}
                />
              </div>
            ) : (
              <Button variant="outline" size="sm" onClick={() => setAddingRecipient(true)}>
                + Ajouter un destinataire
              </Button>
            )}
          </CardContent>
        </Card>

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Historique des versements</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {payouts.length === 0 && (
              <p className="text-sm text-muted-foreground italic">Aucun versement pour le moment.</p>
            )}
            {payouts.map((p) => (
              <div key={p.id} className="flex items-center justify-between p-3 rounded-lg border border-border gap-3">
                <div className="min-w-0">
                  <p className="text-sm font-medium truncate">{p.recipient_name}</p>
                  <p className="text-xs text-muted-foreground">
                    {formatPrice(p.amount_cfa)} · {new Date(p.requested_at).toLocaleDateString('fr-FR')}
                  </p>
                  {p.status === 'failed' && p.failure_reason && (
                    <p className="text-xs text-red-600 mt-0.5">{p.failure_reason}</p>
                  )}
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <Badge className={STATUS_COLOR[p.status] || 'bg-green-100'}>
                    {STATUS_LABELS[p.status] || p.status}
                  </Badge>
                  {p.status === 'failed' && (
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={retryingId === p.id}
                      onClick={() => handleRetry(p.id)}
                    >
                      {retryingId === p.id ? '...' : 'Réessayer'}
                    </Button>
                  )}
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      </section>

      <ConfirmDialog
        open={!!confirmDeleteId}
        title="Supprimer ce destinataire ?"
        description="Il ne recevra plus de versement lors des prochaines distributions."
        confirmLabel="Supprimer"
        cancelLabel="Annuler"
        danger
        onConfirm={handleDelete}
        onCancel={() => setConfirmDeleteId(null)}
      />
    </main>
  );
}
