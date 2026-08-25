'use client';

import { useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { PageHeader } from '@/components/page-header';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { ArrowLeftIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';

export default function AdminBroadcastPage() {
  const [subject, setSubject] = useState('');
  const [htmlBody, setHtmlBody] = useState('');
  const [sendingTest, setSendingTest] = useState(false);
  const [sendingAll, setSendingAll] = useState(false);
  const [confirmSendAll, setConfirmSendAll] = useState(false);
  const [message, setMessage] = useState('');
  const [error, setError] = useState('');

  const canSend = subject.trim().length > 0 && htmlBody.trim().length > 0;

  const handleSendTest = async () => {
    setError('');
    setMessage('');
    setSendingTest(true);
    try {
      const result = await api.sendBroadcast({ subject, html: htmlBody, test_only: true });
      setMessage(`Email de test envoyé à ${result.to}.`);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSendingTest(false);
    }
  };

  const handleSendAll = async () => {
    setConfirmSendAll(false);
    setError('');
    setMessage('');
    setSendingAll(true);
    try {
      const result = await api.sendBroadcast({ subject, html: htmlBody });
      setMessage(`Envoi lancé vers ${result.recipients} destinataire(s). La livraison se poursuit en arrière-plan.`);
      setSubject('');
      setHtmlBody('');
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setSendingAll(false);
    }
  };

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Diffusion email"
        description="Composez un email HTML et envoyez-le à tous les comptes DIARRA"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/admin" />}>
            <ArrowLeftIcon size={16} className="mr-2" />
            Tableau de bord
          </Button>
        }
      />

      <section className="max-w-4xl mx-auto px-4 sm:px-6 py-10 space-y-6">
        {error && (
          <div className="p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}
        {message && <div className="p-3 bg-green-50 text-green-700 rounded text-sm">{message}</div>}

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Nouvel email</CardTitle>
            <CardDescription>
              Envoyé tel quel (pas d&apos;échappement) : composez le HTML complet vous-même.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="broadcast-subject">Sujet</Label>
              <Input
                id="broadcast-subject"
                value={subject}
                onChange={(e) => setSubject(e.target.value)}
                placeholder="Ex: Nouveautés DIARRA ce mois-ci"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="broadcast-html">Contenu HTML</Label>
              <textarea
                id="broadcast-html"
                value={htmlBody}
                onChange={(e) => setHtmlBody(e.target.value)}
                rows={14}
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono"
                placeholder="<p>Bonjour,</p><p>...</p>"
              />
            </div>

            <div className="flex flex-col sm:flex-row gap-3">
              <Button variant="outline" onClick={handleSendTest} disabled={!canSend || sendingTest}>
                {sendingTest ? 'Envoi…' : "M'envoyer un test"}
              </Button>
              <Button
                onClick={() => setConfirmSendAll(true)}
                disabled={!canSend || sendingAll}
                className="sm:ml-auto"
              >
                {sendingAll ? 'Envoi…' : 'Envoyer à tous les utilisateurs'}
              </Button>
            </div>
          </CardContent>
        </Card>

        {htmlBody && (
          <Card className="shadow-card border-green-900/5">
            <CardHeader>
              <CardTitle className="text-sm">Aperçu</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="border rounded-md p-4 bg-white" dangerouslySetInnerHTML={{ __html: htmlBody }} />
            </CardContent>
          </Card>
        )}
      </section>

      <ConfirmDialog
        open={confirmSendAll}
        title="Envoyer à tous les utilisateurs ?"
        description="Cet email partira vers tous les comptes DIARRA. Cette action ne peut pas être annulée une fois lancée — pensez à envoyer un test d'abord."
        confirmLabel="Envoyer à tous"
        cancelLabel="Annuler"
        danger
        onConfirm={handleSendAll}
        onCancel={() => setConfirmSendAll(false)}
      />
    </main>
  );
}
