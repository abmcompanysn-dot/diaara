import { ImageResponse } from 'next/og';

// Export statique (output: 'export') : doit être généré au build, pas à
// la demande — même contrainte que robots.ts.
export const dynamic = 'force-static';

// Généré une seule fois au build (export statique) — image de partage par
// défaut pour les pages qui n'en ont pas de spécifique (accueil, catalogue,
// etc.). La fiche produit a sa propre image dynamique côté backend
// (ProductHandler.Share, voir /p/{id}).
export const size = { width: 1200, height: 630 };
export const contentType = 'image/png';

export default function Image() {
  return new ImageResponse(
    (
      <div
        style={{
          width: '100%',
          height: '100%',
          display: 'flex',
          flexDirection: 'column',
          justifyContent: 'center',
          padding: '80px',
          background: 'linear-gradient(135deg, #0A2A1C 0%, #0F7A4C 130%)',
          color: '#EAF3EC',
          fontFamily: 'sans-serif',
        }}
      >
        <div
          style={{
            display: 'flex',
            fontSize: 30,
            letterSpacing: 4,
            textTransform: 'uppercase',
            color: '#B4E62B',
            marginBottom: 28,
          }}
        >
          Le marché des biens numériques
        </div>
        <div style={{ display: 'flex', fontSize: 108, fontWeight: 800, lineHeight: 1.02 }}>
          DIARRA
        </div>
        <div style={{ display: 'flex', fontSize: 34, marginTop: 32, color: 'rgba(234,243,236,0.82)', maxWidth: 880 }}>
          Clés d&apos;abonnement, comptes, ebooks, PDF — payé en mobile money, reçu instantanément.
        </div>
      </div>
    ),
    { ...size }
  );
}
