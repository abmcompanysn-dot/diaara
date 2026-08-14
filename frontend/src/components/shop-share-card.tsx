'use client';

import { useEffect, useState } from 'react';
import QRCode from 'qrcode';
import { Button } from '@/components/ui/button';
import { CopyIcon, CheckIcon, ShareIcon } from '@/components/icons';

interface ShopShareCardProps {
  vendorId: string;
  shopTitle: string;
}

// Carte "boutique" affichée dans l'espace vendeur : QR code + lien à
// partager vers la boutique publique (voir frontend/src/app/boutique).
export function ShopShareCard({ vendorId, shopTitle }: ShopShareCardProps) {
  const [qrDataUrl, setQrDataUrl] = useState('');
  const [copied, setCopied] = useState(false);
  const shopUrl = typeof window !== 'undefined' ? `${window.location.origin}/boutique?id=${vendorId}` : '';

  useEffect(() => {
    if (!shopUrl) return;
    QRCode.toDataURL(shopUrl, { width: 160, margin: 1, color: { dark: '#0a3d2b', light: '#ffffff' } })
      .then(setQrDataUrl)
      .catch(() => {});
  }, [shopUrl]);

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(shopUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      // Presse-papiers indisponible (permissions navigateur) : rien d'autre à faire.
    }
  };

  const handleShare = async () => {
    if (navigator.share) {
      try {
        await navigator.share({ title: shopTitle, text: `Découvrez ma boutique sur DIARRA : ${shopTitle}`, url: shopUrl });
        return;
      } catch {
        // Partage annulé ou indisponible : on retombe sur la copie du lien.
      }
    }
    handleCopy();
  };

  return (
    <div className="rounded-2xl bg-white border border-green-900/10 shadow-card p-5 sm:p-6">
      <div className="flex flex-col sm:flex-row items-center sm:items-start gap-5">
        <div className="shrink-0 w-[168px] h-[168px] rounded-xl bg-green-50 flex items-center justify-center overflow-hidden">
          {qrDataUrl ? (
            <img src={qrDataUrl} alt="QR code de la boutique" className="w-40 h-40" />
          ) : (
            <div className="w-40 h-40 animate-pulse bg-green-100 rounded-lg" />
          )}
        </div>
        <div className="flex-1 min-w-0 text-center sm:text-left">
          <p className="font-mono text-xs uppercase tracking-widest text-green-700">Ma boutique</p>
          <h3 className="font-display font-bold text-lg text-green-950 mt-1">{shopTitle}</h3>
          <p className="text-sm text-green-900/60 mt-1">
            Partagez ce lien ou ce QR code pour que vos clients découvrent tous vos produits sur DIARRA.
          </p>
          <p className="mt-3 font-mono text-xs text-green-900/50 break-all">{shopUrl}</p>
          <div className="flex flex-wrap justify-center sm:justify-start gap-2 mt-3">
            <Button onClick={handleShare} className="h-9 gap-1.5 bg-green-950 text-white hover:bg-green-900">
              <ShareIcon size={14} />
              Partager
            </Button>
            <Button variant="outline" onClick={handleCopy} className="h-9 gap-1.5">
              {copied ? <CheckIcon size={14} /> : <CopyIcon size={14} />}
              {copied ? 'Copié' : 'Copier le lien'}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
