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
import { ArrowLeftIcon, TrashIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';

interface SupportAgent {
  id: string;
  name: string;
  email: string;
  phone?: string;
  callmebot_apikey?: string;
  active: boolean;
}

interface SupportContactRequest {
  id: string;
  name: string;
  contact_method: string;
  contact_value: string;
  message: string;
  created_at: string;
}

export default function AdminSupportAgentsPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const [agents, setAgents] = useState<SupportAgent[]>([]);
  const [requests, setRequests] = useState<SupportContactRequest[]>([]);

  const [addingAgent, setAddingAgent] = useState(false);
  const [savingAgent, setSavingAgent] = useState(false);
  const [agentName, setAgentName] = useState('');
  const [agentEmail, setAgentEmail] = useState('');
  const [agentPhone, setAgentPhone] = useState('');
  const [agentApiKey, setAgentApiKey] = useState('');
  const [agentError, setAgentError] = useState('');
  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null);

  const load = () => {
    Promise.all([api.getSupportAgents(), api.getSupportContacts()])
      .then(([a, r]) => {
        setAgents(a.agents);
        setRequests(r.requests);
      })
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  };

  useEffect(load, []);

  const handleAddAgent = async () => {
    if (!agentName.trim() || !agentEmail.trim()) {
      setAgentError('Le nom et l’email sont requis.');
      return;
    }
    setAgentError('');
    setSavingAgent(true);
    try {
      await api.createSupportAgent({
        name: agentName.trim(),
        email: agentEmail.trim(),
        phone: agentPhone.trim() || undefined,
        callmebot_apikey: agentApiKey.trim() || undefined,
      });
      setAgentName('');
      setAgentEmail('');
      setAgentPhone('');
      setAgentApiKey('');
      setAddingAgent(false);
      load();
    } catch (err: any) {
      setAgentError(friendlyError(err));
    } finally {
      setSavingAgent(false);
    }
  };

  const handleToggleActive = async (agent: SupportAgent) => {
    try {
      await api.updateSupportAgent(agent.id, { active: !agent.active });
      load();
    } catch (err: any) {
      setError(friendlyError(err));
    }
  };

  const handleDelete = async () => {
    if (!confirmDeleteId) return;
    try {
      await api.deleteSupportAgent(confirmDeleteId);
      setConfirmDeleteId(null);
      load();
    } catch (err: any) {
      setError(friendlyError(err));
      setConfirmDeleteId(null);
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader eyebrow="// administration" title="Contact support" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Contact support"
        description="Agents notifiés par email à chaque message envoyé depuis le bouton de contact du site"
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

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Agents support</CardTitle>
            <CardDescription>
              Chaque agent actif reçoit un email avec un lien de réponse directe (email ou WhatsApp) à chaque
              nouveau contact.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {agents.map((agent) => (
              <div
                key={agent.id}
                className="flex items-center justify-between p-3 rounded-lg border border-border gap-3"
              >
                <div className="min-w-0">
                  <p className="text-sm font-semibold truncate">{agent.name}</p>
                  <p className="text-xs text-muted-foreground truncate">{agent.email}</p>
                  {agent.phone && agent.callmebot_apikey && (
                    <Badge className="bg-green-100 text-green-700 mt-1">WhatsApp actif</Badge>
                  )}
                </div>
                <div className="flex items-center gap-3 shrink-0">
                  <label className="flex items-center gap-1.5 text-xs cursor-pointer">
                    <Checkbox checked={agent.active} onCheckedChange={() => handleToggleActive(agent)} />
                    Actif
                  </label>
                  <button
                    type="button"
                    onClick={() => setConfirmDeleteId(agent.id)}
                    className="text-red-600 hover:text-red-700"
                    aria-label={`Supprimer ${agent.name}`}
                  >
                    <TrashIcon size={16} />
                  </button>
                </div>
              </div>
            ))}
            {agents.length === 0 && !addingAgent && (
              <p className="text-sm text-muted-foreground italic">
                Aucun agent enregistré — les messages de contact ne seront pas notifiés par email.
              </p>
            )}

            {addingAgent ? (
              <div className="pt-3 border-t border-border space-y-3">
                <div className="space-y-2">
                  <Label htmlFor="agent-name">Nom</Label>
                  <Input id="agent-name" value={agentName} onChange={(e) => setAgentName(e.target.value)} placeholder="Ex: Awa" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="agent-email">Email</Label>
                  <Input
                    id="agent-email"
                    type="email"
                    value={agentEmail}
                    onChange={(e) => setAgentEmail(e.target.value)}
                    placeholder="awa@diarra.com"
                  />
                </div>
                <div className="pt-2 border-t border-border space-y-3">
                  <p className="text-xs text-muted-foreground">
                    Optionnel : notification WhatsApp automatique via{' '}
                    <a
                      href="https://www.callmebot.com/blog/free-api-whatsapp-messages/"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="underline"
                    >
                      CallMeBot
                    </a>
                    . Chaque agent envoie « I allow callmebot to send me messages » au numéro CallMeBot depuis son
                    propre WhatsApp pour recevoir sa clé API.
                  </p>
                  <div className="space-y-2">
                    <Label htmlFor="agent-phone">Numéro WhatsApp (avec indicatif)</Label>
                    <Input
                      id="agent-phone"
                      value={agentPhone}
                      onChange={(e) => setAgentPhone(e.target.value)}
                      placeholder="Ex: +221771234567"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="agent-apikey">Clé API CallMeBot</Label>
                    <Input
                      id="agent-apikey"
                      value={agentApiKey}
                      onChange={(e) => setAgentApiKey(e.target.value)}
                      placeholder="Ex: 123456"
                    />
                  </div>
                </div>
                {agentError && <p className="text-xs text-red-600">{agentError}</p>}
                <div className="flex gap-2">
                  <Button onClick={handleAddAgent} disabled={savingAgent} size="sm" className="font-semibold">
                    {savingAgent ? 'Enregistrement...' : 'Ajouter'}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      setAddingAgent(false);
                      setAgentError('');
                    }}
                  >
                    Annuler
                  </Button>
                </div>
              </div>
            ) : (
              <Button variant="outline" size="sm" onClick={() => setAddingAgent(true)}>
                + Ajouter un agent
              </Button>
            )}
          </CardContent>
        </Card>

        <Card className="shadow-card border-green-900/5">
          <CardHeader>
            <CardTitle>Derniers messages reçus</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {requests.length === 0 && (
              <p className="text-sm text-muted-foreground italic">Aucun message pour le moment.</p>
            )}
            {requests.map((req) => (
              <div key={req.id} className="p-3 rounded-lg border border-border space-y-1">
                <div className="flex items-center justify-between gap-3">
                  <p className="text-sm font-semibold truncate">{req.name}</p>
                  <Badge className="bg-green-100 text-green-700 shrink-0">
                    {req.contact_method === 'whatsapp' ? 'WhatsApp' : 'Email'}
                  </Badge>
                </div>
                <p className="text-xs text-muted-foreground font-mono">{req.contact_value}</p>
                <p className="text-sm">{req.message}</p>
                <p className="text-xs text-muted-foreground">{new Date(req.created_at).toLocaleString('fr-FR')}</p>
              </div>
            ))}
          </CardContent>
        </Card>
      </section>

      <ConfirmDialog
        open={!!confirmDeleteId}
        title="Supprimer cet agent ?"
        description="Il ne recevra plus les notifications de contact support."
        confirmLabel="Supprimer"
        cancelLabel="Annuler"
        danger
        onConfirm={handleDelete}
        onCancel={() => setConfirmDeleteId(null)}
      />
    </main>
  );
}
