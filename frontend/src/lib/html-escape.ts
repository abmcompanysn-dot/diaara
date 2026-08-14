// escapeHtml — échappe les caractères spéciaux HTML avant interpolation dans
// une chaîne de markup construite à la main (ex: document.write). Nécessaire
// partout où du texte utilisateur (nom, titre de produit...) est inséré hors
// du rendu React/JSX habituel, qui échappe automatiquement.
export function escapeHtml(value: string | null | undefined): string {
  if (!value) return '';
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}
