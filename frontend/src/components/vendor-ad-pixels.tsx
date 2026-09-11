'use client';

import Script from 'next/script';

interface VendorAdPixelsProps {
  facebookPixelId?: string | null;
  googleTagId?: string | null;
}

// Formats stricts attendus (miroir de la validation backend,
// UpdateAdTracking) — filet de sécurité côté client : même si une valeur
// invalide passait malgré tout (API modifiée, vieille donnée...), on ne
// l'interpole jamais dans le <script> inline ci-dessous. Voir audit
// sécurité 2026-09-11 (XSS stocké via un pixel publicitaire vendeur).
const FACEBOOK_PIXEL_ID_RE = /^\d{6,20}$/;
const GOOGLE_TAG_ID_RE = /^(G|GT|AW|DC)-[A-Z0-9]{4,20}$/;

// Injecte le Facebook Pixel / Google Tag propres à un vendeur (publicité
// qu'il gère lui-même) sur sa boutique et ses pages produit. N'a aucun effet
// tant que le vendeur n'a rien renseigné dans son espace vendeur.
export function VendorAdPixels({ facebookPixelId, googleTagId }: VendorAdPixelsProps) {
  const validPixelId = facebookPixelId && FACEBOOK_PIXEL_ID_RE.test(facebookPixelId) ? facebookPixelId : null;
  const validTagId = googleTagId && GOOGLE_TAG_ID_RE.test(googleTagId) ? googleTagId : null;

  return (
    <>
      {validPixelId && (
        // L'ID n'est jamais concaténé dans le texte du script : il est lu
        // depuis un data-attribute au runtime, donc aucune valeur ne peut
        // jamais se faire passer pour du JavaScript.
        <Script id={`fb-pixel-${validPixelId}`} strategy="afterInteractive" data-pixel-id={validPixelId}>
          {`
            !function(f,b,e,v,n,t,s)
            {if(f.fbq)return;n=f.fbq=function(){n.callMethod?
            n.callMethod.apply(n,arguments):n.queue.push(arguments)};
            if(!f._fbq)f._fbq=n;n.push=n;n.loaded=!0;n.version='2.0';
            n.queue=[];t=b.createElement(e);t.async=!0;
            t.src=v;s=b.getElementsByTagName(e)[0];
            s.parentNode.insertBefore(t,s)}(window, document,'script',
            'https://connect.facebook.net/en_US/fbevents.js');
            var pid = document.currentScript.getAttribute('data-pixel-id');
            fbq('init', pid);
            fbq('track', 'PageView');
          `}
        </Script>
      )}
      {validTagId && (
        <>
          <Script src={`https://www.googletagmanager.com/gtag/js?id=${encodeURIComponent(validTagId)}`} strategy="afterInteractive" />
          <Script id={`gtag-${validTagId}`} strategy="afterInteractive" data-tag-id={validTagId}>
            {`
              window.dataLayer = window.dataLayer || [];
              function gtag(){dataLayer.push(arguments);}
              gtag('js', new Date());
              gtag('config', document.currentScript.getAttribute('data-tag-id'));
            `}
          </Script>
        </>
      )}
    </>
  );
}
