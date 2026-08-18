import { apiOrigin } from './api';
import { CATEGORY_LABELS, formatPrice } from './constants';

interface PosterProduct {
  id: string;
  title: string;
  category: string;
  price_cfa: number;
  cover_image_key?: string | null;
}

const WIDTH = 1080;
const HEIGHT = 1350;
const GREEN_950 = '#052018';
const GREEN_900 = '#0a3225';
const LIME = '#c9f22e';

function loadImage(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.crossOrigin = 'anonymous';
    img.onload = () => resolve(img);
    img.onerror = reject;
    img.src = src;
  });
}

// Dessine `img` en mode "cover" (rempli + recadré) dans le rectangle donné,
// même logique que `object-fit: cover` en CSS.
function drawCover(ctx: CanvasRenderingContext2D, img: HTMLImageElement, x: number, y: number, w: number, h: number) {
  const scale = Math.max(w / img.width, h / img.height);
  const sw = w / scale;
  const sh = h / scale;
  const sx = (img.width - sw) / 2;
  const sy = (img.height - sh) / 2;
  ctx.drawImage(img, sx, sy, sw, sh, x, y, w, h);
}

// Retourne les lignes d'un texte une fois réparties pour tenir dans
// `maxWidth`, jusqu'à `maxLines` (la dernière ligne est tronquée avec "…"
// si le texte ne tient toujours pas).
function wrapText(ctx: CanvasRenderingContext2D, text: string, maxWidth: number, maxLines: number): string[] {
  const words = text.split(' ');
  const lines: string[] = [];
  let current = '';

  for (const word of words) {
    const attempt = current ? `${current} ${word}` : word;
    if (ctx.measureText(attempt).width > maxWidth && current) {
      lines.push(current);
      current = word;
      if (lines.length === maxLines) break;
    } else {
      current = attempt;
    }
  }
  if (lines.length < maxLines && current) lines.push(current);

  if (lines.length === maxLines) {
    let last = lines[maxLines - 1];
    while (ctx.measureText(`${last}…`).width > maxWidth && last.length > 1) {
      last = last.slice(0, -1);
    }
    lines[maxLines - 1] = `${last}…`;
  }
  return lines;
}

// Compose une affiche produit (image + titre + prix + branding DIARRA +
// QR code) et la renvoie en PNG — tout se fait dans le navigateur, aucun
// appel serveur dédié (le QR code est déjà généré ailleurs et passé ici).
export async function generateProductPoster(product: PosterProduct, qrDataUrl: string): Promise<Blob> {
  const canvas = document.createElement('canvas');
  canvas.width = WIDTH;
  canvas.height = HEIGHT;
  const ctx = canvas.getContext('2d');
  if (!ctx) throw new Error('canvas_unsupported');

  await document.fonts.ready;

  // Fond
  ctx.fillStyle = '#ffffff';
  ctx.fillRect(0, 0, WIDTH, HEIGHT);

  // Bandeau haut (marque)
  const headerHeight = 140;
  ctx.fillStyle = GREEN_950;
  ctx.fillRect(0, 0, WIDTH, headerHeight);
  try {
    const logo = await loadImage('/brand/diarra-icon.png');
    const logoSize = 72;
    ctx.drawImage(logo, 48, (headerHeight - logoSize) / 2, logoSize, logoSize);
    ctx.fillStyle = '#ffffff';
    ctx.font = '700 34px Manrope, sans-serif';
    ctx.textBaseline = 'middle';
    ctx.fillText('DIARRA', 48 + logoSize + 20, headerHeight / 2);
  } catch {
    // Logo introuvable : le bandeau reste vert, sans bloquer la génération.
  }

  // Image produit (carrée, sous le bandeau)
  const imgY = headerHeight + 48;
  const imgSize = WIDTH - 96;
  if (product.cover_image_key) {
    try {
      const cover = await loadImage(`${apiOrigin}/api/products/${product.id}/cover`);
      ctx.save();
      const radius = 24;
      ctx.beginPath();
      ctx.moveTo(48 + radius, imgY);
      ctx.arcTo(48 + imgSize, imgY, 48 + imgSize, imgY + imgSize, radius);
      ctx.arcTo(48 + imgSize, imgY + imgSize, 48, imgY + imgSize, radius);
      ctx.arcTo(48, imgY + imgSize, 48, imgY, radius);
      ctx.arcTo(48, imgY, 48 + imgSize, imgY, radius);
      ctx.closePath();
      ctx.clip();
      drawCover(ctx, cover, 48, imgY, imgSize, imgSize);
      ctx.restore();
    } catch {
      drawImageFallback(ctx, product, imgY, imgSize);
    }
  } else {
    drawImageFallback(ctx, product, imgY, imgSize);
  }

  // Titre
  const textY = imgY + imgSize + 70;
  ctx.fillStyle = GREEN_950;
  ctx.font = '700 44px Manrope, sans-serif';
  ctx.textBaseline = 'alphabetic';
  const titleLines = wrapText(ctx, product.title, WIDTH - 96, 2);
  titleLines.forEach((line, i) => ctx.fillText(line, 48, textY + i * 54));

  // Prix
  ctx.fillStyle = GREEN_900;
  ctx.font = '600 36px Manrope, sans-serif';
  const priceY = textY + titleLines.length * 54 + 50;
  ctx.fillText(formatPrice(product.price_cfa), 48, priceY);

  // Bandeau bas : QR code + call-to-action
  const footerHeight = 220;
  const footerY = HEIGHT - footerHeight;
  ctx.fillStyle = '#f4f7f5';
  ctx.fillRect(0, footerY, WIDTH, footerHeight);
  const qr = await loadImage(qrDataUrl);
  const qrSize = 160;
  ctx.drawImage(qr, 48, footerY + (footerHeight - qrSize) / 2, qrSize, qrSize);

  ctx.fillStyle = GREEN_950;
  ctx.font = '700 32px Manrope, sans-serif';
  ctx.fillText('Scannez pour acheter', 48 + qrSize + 32, footerY + footerHeight / 2 - 8);
  ctx.fillStyle = GREEN_900;
  ctx.font = '500 26px Manrope, sans-serif';
  ctx.fillText('diarra.app', 48 + qrSize + 32, footerY + footerHeight / 2 + 32);

  return new Promise((resolve, reject) => {
    canvas.toBlob((blob) => (blob ? resolve(blob) : reject(new Error('poster_export_failed'))), 'image/png');
  });
}

// Repli visuel quand le produit n'a pas d'image de couverture — dégradé +
// initiale + catégorie, même esprit que ProductImage (product-image.tsx).
function drawImageFallback(ctx: CanvasRenderingContext2D, product: PosterProduct, y: number, size: number) {
  const gradient = ctx.createLinearGradient(48, y, 48 + size, y + size);
  gradient.addColorStop(0, '#c9f22e33');
  gradient.addColorStop(1, '#115e4033');
  ctx.fillStyle = gradient;
  ctx.fillRect(48, y, size, size);

  ctx.fillStyle = GREEN_900;
  ctx.font = '700 120px Manrope, sans-serif';
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  const initial = (product.title.trim().charAt(0) || 'P').toUpperCase();
  ctx.fillText(initial, 48 + size / 2, y + size / 2 - 30);

  ctx.font = '600 32px Manrope, sans-serif';
  ctx.fillText(CATEGORY_LABELS[product.category] || product.category, 48 + size / 2, y + size / 2 + 60);
  ctx.textAlign = 'left';
}
