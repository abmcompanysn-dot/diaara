'use client';

import { useEffect, useState } from 'react';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { CopyIcon, CheckIcon } from '@/components/icons';
import { friendlyError } from '@/lib/error-messages';

function CodeBlock({ children }: { children: string }) {
  return (
    <pre className="bg-green-950 text-green-100 text-xs rounded-lg p-4 overflow-x-auto whitespace-pre-wrap break-all">
      {children}
    </pre>
  );
}

export default function AdminAutomationPage() {
  const [key, setKey] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [regenerating, setRegenerating] = useState(false);
  const [copied, setCopied] = useState(false);
  const [confirmRegenerate, setConfirmRegenerate] = useState(false);

  useEffect(() => {
    api
      .getAutomationKey()
      .then((r) => setKey(r.key))
      .catch((err: any) => setError(friendlyError(err)))
      .finally(() => setLoading(false));
  }, []);

  const handleRegenerate = async () => {
    setConfirmRegenerate(false);
    setRegenerating(true);
    setError('');
    try {
      const result = await api.regenerateAutomationKey();
      setKey(result.key);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setRegenerating(false);
    }
  };

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(key);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Presse-papiers indisponible : rien d'autre à faire.
    }
  };

  if (loading)
    return (
      <main>
        <PageHeader back="/admin" eyebrow="// administration" title="Automatisation" />
        <PageLoader />
      </main>
    );

  return (
    <main>
      <PageHeader
        back="/admin"
        eyebrow="// administration"
        title="Automatisation"
        description="Créer des produits DIARRA depuis un script ou une IA externe"
      />

      <section className="max-w-3xl mx-auto px-4 sm:px-6 py-10 space-y-6">
        {error && (
          <div className="p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        <div className="p-6 border rounded-xl bg-white shadow-card border-green-900/10">
          <h2 className="font-display font-bold text-lg text-green-950 mb-1">Clé d'automatisation</h2>
          <p className="text-sm text-green-900/60 mb-4">
            À garder secrète — elle donne le droit de créer des produits en votre nom. La régénérer
            invalide immédiatement l'ancienne clé.
          </p>

          {key ? (
            <div className="flex items-center gap-2">
              <code className="flex-1 bg-secondary/60 rounded-lg px-3 py-2.5 text-xs font-mono break-all">
                {key}
              </code>
              <Button variant="outline" className="h-10 shrink-0 gap-1.5" onClick={handleCopy}>
                {copied ? <CheckIcon size={14} /> : <CopyIcon size={14} />}
                {copied ? 'Copié' : 'Copier'}
              </Button>
            </div>
          ) : (
            <p className="text-sm text-green-900/50 italic mb-2">Aucune clé générée pour le moment.</p>
          )}

          <Button
            variant="outline"
            className="h-10 mt-4"
            disabled={regenerating}
            onClick={() => setConfirmRegenerate(true)}
          >
            {regenerating ? 'Génération...' : key ? 'Régénérer la clé' : 'Générer une clé'}
          </Button>
        </div>

        <div className="p-6 border rounded-xl bg-white shadow-card border-green-900/10 space-y-4">
          <h2 className="font-display font-bold text-lg text-green-950">Comment ça marche</h2>

          <div>
            <p className="text-sm font-semibold text-green-900 mb-1">1. Créer un produit</p>
            <p className="text-sm text-green-900/70 mb-2">
              Envoyez une requête <code>multipart/form-data</code> avec le header{' '}
              <code>X-Automation-Key</code>. Deux cas possibles :
            </p>
            <ul className="text-sm text-green-900/70 list-disc pl-5 space-y-1 mb-3">
              <li>
                <strong>Fichier prêt</strong> (image, PDF, ZIP...) : envoyez-le dans le champ{' '}
                <code>file</code> — il sert de couverture et de fichier livré à l'acheteur.
              </li>
              <li>
                <strong>Pas encore de fichier</strong> : envoyez <code>image_prompt</code> à la
                place. Le produit part en modération avec ce prompt affiché ; un fichier est
                attaché plus tard (étape 2) avant approbation.
              </li>
            </ul>
            <CodeBlock>{`curl -X POST https://origin.abmcy.com/api/automation/products \\
  -H "X-Automation-Key: ${key || '<votre_cle>'}" \\
  -F "title=Mon super template" \\
  -F "description=Un template prêt à l'emploi" \\
  -F "price_cfa=5000" \\
  -F "category=other" \\
  -F "image_prompt=Illustration flat design d'un template de site web, palette verte"`}</CodeBlock>
            <p className="text-xs text-green-900/50 mt-2">
              Champs : <code>title</code>, <code>price_cfa</code>, <code>category</code> requis ;{' '}
              <code>description</code>, <code>vendor_id</code> (compte cible, sinon le vôtre) optionnels.
            </p>
          </div>

          <div>
            <p className="text-sm font-semibold text-green-900 mb-1">2. Attacher le fichier (si envoyé plus tard)</p>
            <CodeBlock>{`curl -X PUT https://origin.abmcy.com/api/automation/products/{id}/file \\
  -H "X-Automation-Key: ${key || '<votre_cle>'}" \\
  -F "file=@/chemin/vers/image.png"`}</CodeBlock>
          </div>

          <div>
            <p className="text-sm font-semibold text-green-900 mb-1">3. Modifier un produit existant</p>
            <p className="text-sm text-green-900/70 mb-2">
              Titre, description, prix, catégorie... — un produit déjà approuvé repasse
              automatiquement "en attente" (le contenu a changé, il doit être revalidé).
            </p>
            <CodeBlock>{`curl -X PUT https://origin.abmcy.com/api/automation/products/{id} \\
  -H "X-Automation-Key: ${key || '<votre_cle>'}" \\
  -H "Content-Type: application/json" \\
  -d '{"title": "Nouveau titre", "price_cfa": 6000}'`}</CodeBlock>
            <p className="text-sm text-green-900/70 mt-3 mb-2">
              Pour tout changer en un seul appel — champs texte, fichier livré et couverture —
              utilisez plutôt cette même route en <code>multipart/form-data</code> :
            </p>
            <CodeBlock>{`curl -X PUT https://origin.abmcy.com/api/automation/products/{id} \\
  -H "X-Automation-Key: ${key || '<votre_cle>'}" \\
  -F "title=Nouveau titre" \\
  -F "price_cfa=6000" \\
  -F "file=@/chemin/vers/nouveau-fichier.pdf" \\
  -F "cover=@/chemin/vers/nouvelle-cover.png"`}</CodeBlock>
            <p className="text-xs text-green-900/50 mt-2">
              Tous les champs sont optionnels et indépendants : n&apos;envoyez que ceux à changer.
            </p>
          </div>

          <div>
            <p className="text-sm font-semibold text-green-900 mb-1">
              4. Changer juste l'image de couverture
            </p>
            <p className="text-sm text-green-900/70 mb-2">
              Indépendant du fichier livré à l'acheteur (étape 2) — pour remplacer le visuel
              marketing sans toucher au fichier vendu.
            </p>
            <CodeBlock>{`curl -X PUT https://origin.abmcy.com/api/automation/products/{id}/cover \\
  -H "X-Automation-Key: ${key || '<votre_cle>'}" \\
  -F "file=@/chemin/vers/nouvelle-cover.png"`}</CodeBlock>
          </div>

          <div>
            <p className="text-sm font-semibold text-green-900 mb-1">5. Modération</p>
            <p className="text-sm text-green-900/70">
              Le produit reste "en attente" comme n'importe quel produit créé depuis le site.
              Retrouvez-le dans{' '}
              <a href="/admin/products" className="text-green-700 underline">
                Modération
              </a>{' '}
              pour l'approuver — le bouton "Approuver" reste désactivé tant qu'aucun fichier n'est
              attaché.
            </p>
          </div>
        </div>
      </section>

      <ConfirmDialog
        open={confirmRegenerate}
        title={key ? 'Régénérer la clé ?' : 'Générer une clé ?'}
        description={
          key
            ? "L'ancienne clé cessera de fonctionner immédiatement — tout script qui l'utilise encore échouera."
            : undefined
        }
        confirmLabel={key ? 'Régénérer' : 'Générer'}
        cancelLabel="Annuler"
        danger={!!key}
        onConfirm={handleRegenerate}
        onCancel={() => setConfirmRegenerate(false)}
      />
    </main>
  );
}
