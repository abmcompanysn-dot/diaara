'use client';

import { useEffect, useState } from 'react';
import { DownloadIcon } from '@/components/icons';

// InstallPromptBanner — propose d'installer DIARRA comme application
// (PWA). Comportement volontairement différent de "ne plus jamais
// afficher" : un clic sur "Plus tard" masque le bandeau pour CETTE session
// (sessionStorage, pas localStorage) — il réapparaîtra à la prochaine visite
// si toujours pas installé, cohérent avec le fait que l'utilisateur pourrait
// changer d'avis plus tard sans avoir à chercher une option cachée.
//
// iOS n'émet JAMAIS `beforeinstallprompt` (aucune API pour déclencher
// l'installation par script) — on détecte l'UA et on affiche des
// instructions manuelles à la place (Partager -> Sur l'écran d'accueil).

const DISMISS_KEY = 'diarra_install_banner_dismissed';

function isIOS(): boolean {
  if (typeof navigator === 'undefined') return false;
  return /iphone|ipad|ipod/i.test(navigator.userAgent);
}

function isStandalone(): boolean {
  if (typeof window === 'undefined') return false;
  return (
    window.matchMedia?.('(display-mode: standalone)').matches ||
    // iOS Safari : propriété non standard, absente du typage DOM officiel.
    (navigator as unknown as { standalone?: boolean }).standalone === true
  );
}

export function InstallPromptBanner() {
  const [deferredPrompt, setDeferredPrompt] = useState<Event | null>(null);
  const [showIOSInstructions, setShowIOSInstructions] = useState(false);
  const [dismissed, setDismissed] = useState(true); // true par défaut : évite un flash avant le check useEffect

  useEffect(() => {
    if (isStandalone()) return; // déjà installé, rien à proposer

    try {
      if (sessionStorage.getItem(DISMISS_KEY) === 'true') return;
    } catch {
      // sessionStorage indisponible (navigation privée stricte) : on
      // affiche quand même, tant pis pour la mémorisation du dismiss.
    }
    setDismissed(false);

    if (isIOS()) {
      setShowIOSInstructions(true);
      return;
    }

    const onBeforeInstallPrompt = (e: Event) => {
      e.preventDefault();
      setDeferredPrompt(e);
    };
    window.addEventListener('beforeinstallprompt', onBeforeInstallPrompt);
    return () => window.removeEventListener('beforeinstallprompt', onBeforeInstallPrompt);
  }, []);

  const handleDismiss = () => {
    setDismissed(true);
    try {
      sessionStorage.setItem(DISMISS_KEY, 'true');
    } catch {
      // Pas bloquant.
    }
  };

  const handleInstall = async () => {
    if (!deferredPrompt) return;
    // prompt()/userChoice ne sont pas dans le typage DOM standard (API encore
    // expérimentale) — cast local plutôt que d'élargir un type global.
    const promptEvent = deferredPrompt as Event & {
      prompt: () => Promise<void>;
      userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
    };
    await promptEvent.prompt();
    await promptEvent.userChoice;
    setDeferredPrompt(null);
    handleDismiss();
  };

  if (dismissed) return null;
  if (!showIOSInstructions && !deferredPrompt) return null; // rien à proposer tant que l'event n'est pas arrivé

  return (
    <div
      role="region"
      aria-label="Installer DIARRA"
      className="fixed bottom-0 inset-x-0 z-50 bg-green-950 text-white px-4 py-3 shadow-lift flex items-center gap-3 pb-[calc(env(safe-area-inset-bottom)+0.75rem)]"
    >
      <span className="w-9 h-9 rounded-lg bg-white/10 flex items-center justify-center shrink-0">
        <DownloadIcon size={18} />
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-sm font-semibold">Installer DIARRA</p>
        <p className="text-xs text-white/70 mt-0.5">
          {showIOSInstructions
            ? 'Appuyez sur Partager puis « Sur l\'écran d\'accueil ».'
            : 'Accédez plus vite au catalogue et à vos commandes.'}
        </p>
      </div>
      {!showIOSInstructions && (
        <button
          type="button"
          onClick={handleInstall}
          className="shrink-0 px-3 py-1.5 rounded-lg bg-lime text-green-950 text-sm font-semibold hover:bg-green-300"
        >
          Installer
        </button>
      )}
      <button
        type="button"
        onClick={handleDismiss}
        aria-label="Fermer"
        className="shrink-0 text-white/60 hover:text-white text-sm px-2"
      >
        ✕
      </button>
    </div>
  );
}
