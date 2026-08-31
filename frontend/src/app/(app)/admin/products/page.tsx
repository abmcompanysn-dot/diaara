'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { api } from '@/lib/api';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { PageHeader } from '@/components/page-header';
import { PageLoader } from '@/components/page-loader';
import { EmptyState } from '@/components/empty-state';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { Textarea } from '@/components/ui/textarea';
import { Input } from '@/components/ui/input';
import { CopyIcon, CheckIcon } from '@/components/icons';
import { PRODUCT_STATUS_BADGE, PRODUCT_STATUS_LABELS, CATEGORY_LABELS } from '@/lib/constants';
import { friendlyError } from '@/lib/error-messages';
import { cn } from '@/lib/utils';

interface Product {
  id: string;
  title: string;
  description: string;
  price_cfa: number;
  category: string;
  vendor_id: string;
  vendor_email?: string;
  moderation_status: string;
  moderation_note?: string | null;
  file_key?: string;
  image_prompt?: string | null;
  deletion_requested?: boolean;
  created_at: string;
}

type Tab = 'pending' | 'approved' | 'rejected' | 'all' | 'deletion';

const TABS: { key: Tab; label: string }[] = [
  { key: 'pending', label: 'En attente' },
  { key: 'approved', label: 'Approuvés' },
  { key: 'rejected', label: 'Refusés' },
  { key: 'deletion', label: 'Suppressions demandées' },
  { key: 'all', label: 'Tous' },
];

export default function AdminProductsPage() {
  const [tab, setTab] = useState<Tab>('pending');
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [toModerate, setToModerate] = useState<{ id: string; status: string } | null>(null);
  const [rejectNote, setRejectNote] = useState('');
  const [uploadingId, setUploadingId] = useState<string | null>(null);
  const [downloadingId, setDownloadingId] = useState<string | null>(null);
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [toDeletionAction, setToDeletionAction] = useState<{ id: string; action: 'confirm' | 'cancel' } | null>(null);

  useEffect(() => {
    loadProducts(tab);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab]);

  const loadProducts = async (t: Tab) => {
    setLoading(true);
    try {
      const result = await api.getAdminProducts(t === 'all' || t === 'deletion' ? undefined : t);
      setProducts(result.products);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setLoading(false);
    }
  };

  const handleDeletionAction = async () => {
    if (!toDeletionAction) return;
    try {
      if (toDeletionAction.action === 'confirm') {
        await api.confirmProductDeletion(toDeletionAction.id);
      } else {
        await api.cancelProductDeletion(toDeletionAction.id);
      }
      await loadProducts(tab);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setToDeletionAction(null);
    }
  };

  const handleModerate = async () => {
    if (!toModerate) return;
    try {
      await api.moderateProduct(toModerate.id, {
        status: toModerate.status,
        ...(toModerate.status === 'rejected' && rejectNote.trim() ? { note: rejectNote.trim() } : {}),
      });
      // Le produit change de statut : il ne fait peut-être plus partie de l'onglet actif.
      await loadProducts(tab);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setToModerate(null);
      setRejectNote('');
    }
  };

  const handleAttachFile = async (productId: string, file: File | undefined) => {
    if (!file) return;
    setUploadingId(productId);
    try {
      await api.attachProductFile(productId, file);
      await loadProducts(tab);
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setUploadingId(null);
    }
  };

  const handleDownloadFile = async (productId: string) => {
    setDownloadingId(productId);
    try {
      const { signed_url } = await api.getAdminProductDownloadUrl(productId);
      window.open(signed_url, '_blank', 'noopener');
    } catch (err: any) {
      setError(friendlyError(err));
    } finally {
      setDownloadingId(null);
    }
  };

  const handleCopyPrompt = async (productId: string, prompt: string) => {
    try {
      await navigator.clipboard.writeText(prompt);
      setCopiedId(productId);
      setTimeout(() => setCopiedId(null), 2000);
    } catch {
      // Presse-papiers indisponible : rien d'autre à faire.
    }
  };

  const displayedProducts = tab === 'deletion' ? products.filter((p) => p.deletion_requested) : products;

  return (
    <main>
      <PageHeader
        eyebrow="// administration"
        title="Modération"
        description="Produits soumis par les vendeurs"
        actions={
          <Button variant="outline" size="sm" render={<Link href="/admin" />}>
            ← Dashboard
          </Button>
        }
      />

      <section className="max-w-6xl mx-auto px-4 sm:px-6 py-10">
        <div className="flex gap-1.5 mb-6 border-b border-green-900/10 overflow-x-auto">
          {TABS.map((t) => (
            <button
              key={t.key}
              type="button"
              onClick={() => setTab(t.key)}
              className={cn(
                'px-4 py-2.5 text-sm font-medium whitespace-nowrap border-b-2 -mb-px transition-colors',
                tab === t.key
                  ? 'border-lime text-green-950'
                  : 'border-transparent text-green-900/50 hover:text-green-900'
              )}
            >
              {t.label}
            </button>
          ))}
        </div>

        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive rounded text-sm" role="alert">
            {error}
          </div>
        )}

        {loading ? (
          <PageLoader />
        ) : displayedProducts.length === 0 ? (
          <EmptyState
            title="Aucun produit"
            description={
              tab === 'pending'
                ? 'Toutes les soumissions ont été traitées.'
                : tab === 'deletion'
                  ? 'Aucune demande de suppression en attente.'
                  : 'Rien à afficher dans cet onglet.'
            }
          />
        ) : (
          <div className="space-y-4">
            {displayedProducts.map((product) => (
              <div key={product.id} className="p-6 border rounded-xl bg-white shadow-card border-green-900/10">
                <div className="flex justify-between items-start mb-2 gap-3">
                  <h2 className="font-display font-bold text-lg text-green-950">{product.title}</h2>
                  <div className="flex items-center gap-2 shrink-0">
                    <Badge className={PRODUCT_STATUS_BADGE[product.moderation_status]}>
                      {PRODUCT_STATUS_LABELS[product.moderation_status] || product.moderation_status}
                    </Badge>
                    <span className="font-mono font-bold text-green-600">
                      {product.price_cfa.toLocaleString('fr-FR')} FCFA
                    </span>
                  </div>
                </div>
                <p className="text-green-900/70 text-sm mb-4 leading-relaxed">
                  {product.description || 'Pas de description'}
                </p>
                {product.moderation_status === 'rejected' && product.moderation_note && (
                  <p className="text-sm text-red-600 mb-4">Raison du refus : {product.moderation_note}</p>
                )}
                {product.deletion_requested && (
                  <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg flex flex-wrap items-center justify-between gap-3">
                    <p className="text-sm text-red-800 font-medium">
                      Le vendeur a demandé la suppression de ce produit.
                    </p>
                    <div className="flex gap-2 shrink-0">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => setToDeletionAction({ id: product.id, action: 'cancel' })}
                      >
                        Refuser la demande
                      </Button>
                      <Button
                        size="sm"
                        variant="destructive"
                        onClick={() => setToDeletionAction({ id: product.id, action: 'confirm' })}
                      >
                        Confirmer la suppression
                      </Button>
                    </div>
                  </div>
                )}
                {!product.file_key && product.image_prompt && (
                  <div className="mb-4 p-3 bg-amber-50 border border-amber-200 rounded-lg">
                    <p className="text-xs font-semibold text-amber-800 mb-1">
                      Créé automatiquement — aucun fichier attaché. Prompt de génération d'image :
                    </p>
                    <div className="flex items-start gap-2">
                      <p className="text-sm text-amber-900/80 flex-1">{product.image_prompt}</p>
                      <Button
                        size="sm"
                        variant="outline"
                        className="h-8 shrink-0 gap-1.5"
                        onClick={() => handleCopyPrompt(product.id, product.image_prompt!)}
                      >
                        {copiedId === product.id ? <CheckIcon size={14} /> : <CopyIcon size={14} />}
                        {copiedId === product.id ? 'Copié' : 'Copier'}
                      </Button>
                    </div>
                    <div className="mt-3 flex items-center gap-2">
                      <Input
                        type="file"
                        accept="image/*"
                        disabled={uploadingId === product.id}
                        onChange={(e) => handleAttachFile(product.id, e.target.files?.[0])}
                        className="bg-white text-sm"
                      />
                      {uploadingId === product.id && <span className="text-xs text-amber-800 shrink-0">Envoi...</span>}
                    </div>
                  </div>
                )}
                <div className="flex flex-wrap items-center gap-4 text-xs text-muted-foreground mb-6">
                  <Badge variant="outline" className="bg-green-50 text-green-700 border-green-200">
                    {CATEGORY_LABELS[product.category] || product.category}
                  </Badge>
                  <span>Vendeur : {product.vendor_email || product.vendor_id.slice(0, 8) + '...'}</span>
                  <span>{new Date(product.created_at).toLocaleDateString('fr-FR')}</span>
                </div>
                <div className="flex flex-wrap gap-3">
                  {product.file_key && (
                    <Button
                      variant="outline"
                      onClick={() => handleDownloadFile(product.id)}
                      disabled={downloadingId === product.id}
                    >
                      {downloadingId === product.id ? 'Préparation...' : 'Télécharger le fichier'}
                    </Button>
                  )}
                  {product.moderation_status !== 'approved' && (
                    <Button
                      onClick={() => setToModerate({ id: product.id, status: 'approved' })}
                      disabled={!product.file_key}
                      title={!product.file_key ? 'Attachez un fichier avant d\'approuver' : undefined}
                      className="bg-green-600 text-white hover:bg-green-500"
                    >
                      Approuver
                    </Button>
                  )}
                  {product.moderation_status !== 'rejected' && (
                    <Button
                      variant="destructive"
                      onClick={() => setToModerate({ id: product.id, status: 'rejected' })}
                    >
                      Refuser
                    </Button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </section>

      <ConfirmDialog
        open={!!toModerate}
        title={toModerate?.status === 'approved' ? "Approuver ce produit ?" : "Refuser ce produit ?"}
        description={toModerate?.status === 'approved'
          ? "Le produit sera publié et deviendra visible pour tous les acheteurs."
          : "Le produit sera marqué comme refusé et le vendeur sera notifié."}
        confirmLabel={toModerate?.status === 'approved' ? "Approuver" : "Refuser"}
        cancelLabel="Annuler"
        danger={toModerate?.status === 'rejected'}
        onConfirm={handleModerate}
        onCancel={() => {
          setToModerate(null);
          setRejectNote('');
        }}
      >
        {toModerate?.status === 'rejected' && (
          <div className="space-y-1.5">
            <label htmlFor="reject-note" className="text-xs font-medium text-green-900/70">
              Raison du refus (envoyée au vendeur)
            </label>
            <Textarea
              id="reject-note"
              value={rejectNote}
              onChange={(e) => setRejectNote(e.target.value)}
              placeholder="Ex : image de couverture floue, description incomplète..."
              className="min-h-20 text-sm"
            />
          </div>
        )}
      </ConfirmDialog>

      <ConfirmDialog
        open={!!toDeletionAction}
        title={toDeletionAction?.action === 'confirm' ? 'Supprimer définitivement ce produit ?' : 'Refuser la demande de suppression ?'}
        description={
          toDeletionAction?.action === 'confirm'
            ? 'Le produit et son fichier livré seront supprimés définitivement. Cette action est irréversible.'
            : 'Le produit reste actif dans la boutique du vendeur.'
        }
        confirmLabel={toDeletionAction?.action === 'confirm' ? 'Supprimer définitivement' : 'Refuser la demande'}
        cancelLabel="Annuler"
        danger={toDeletionAction?.action === 'confirm'}
        onConfirm={handleDeletionAction}
        onCancel={() => setToDeletionAction(null)}
      />
    </main>
  );
}
