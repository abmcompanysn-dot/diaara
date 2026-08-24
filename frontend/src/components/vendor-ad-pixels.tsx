'use client';

import Script from 'next/script';

interface VendorAdPixelsProps {
  facebookPixelId?: string | null;
  googleTagId?: string | null;
}

// Injecte le Facebook Pixel / Google Tag propres à un vendeur (publicité
// qu'il gère lui-même) sur sa boutique et ses pages produit. N'a aucun effet
// tant que le vendeur n'a rien renseigné dans son espace vendeur.
export function VendorAdPixels({ facebookPixelId, googleTagId }: VendorAdPixelsProps) {
  return (
    <>
      {facebookPixelId && (
        <Script id={`fb-pixel-${facebookPixelId}`} strategy="afterInteractive">
          {`
            !function(f,b,e,v,n,t,s)
            {if(f.fbq)return;n=f.fbq=function(){n.callMethod?
            n.callMethod.apply(n,arguments):n.queue.push(arguments)};
            if(!f._fbq)f._fbq=n;n.push=n;n.loaded=!0;n.version='2.0';
            n.queue=[];t=b.createElement(e);t.async=!0;
            t.src=v;s=b.getElementsByTagName(e)[0];
            s.parentNode.insertBefore(t,s)}(window, document,'script',
            'https://connect.facebook.net/en_US/fbevents.js');
            fbq('init', '${facebookPixelId}');
            fbq('track', 'PageView');
          `}
        </Script>
      )}
      {googleTagId && (
        <>
          <Script src={`https://www.googletagmanager.com/gtag/js?id=${googleTagId}`} strategy="afterInteractive" />
          <Script id={`gtag-${googleTagId}`} strategy="afterInteractive">
            {`
              window.dataLayer = window.dataLayer || [];
              function gtag(){dataLayer.push(arguments);}
              gtag('js', new Date());
              gtag('config', '${googleTagId}');
            `}
          </Script>
        </>
      )}
    </>
  );
}
