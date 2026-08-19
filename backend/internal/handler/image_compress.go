package handler

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
)

// coverMaxWidth — les cartes catalogue/fiche produit n'affichent jamais une
// couverture plus large que ça ; au-delà, ce ne sont que des octets envoyés
// pour rien sur un réseau mobile money en Afrique.
const coverMaxWidth = 1200

// compressCoverImage redimensionne (si plus large que coverMaxWidth) et
// recompresse une image de couverture avant stockage. Seuls JPEG et PNG sont
// traités : WebP (déjà compact, pas d'encodeur dans la stdlib Go) et GIF
// (perdrait son animation en repassant par image.Decode/Encode) sont
// renvoyés tels quels. En cas d'erreur de décodage, renvoie les octets
// d'origine plutôt que de faire échouer l'upload pour un souci de
// compression — mieux vaut une image non optimisée qu'un upload cassé.
func compressCoverImage(data []byte) []byte {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return data
	}
	if format != "jpeg" && format != "png" {
		return data
	}

	bounds := img.Bounds()
	if bounds.Dx() > coverMaxWidth {
		newHeight := bounds.Dy() * coverMaxWidth / bounds.Dx()
		resized := image.NewRGBA(image.Rect(0, 0, coverMaxWidth, newHeight))
		draw.BiLinear.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)
		img = resized
	}

	var buf bytes.Buffer
	switch format {
	case "jpeg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 82}); err != nil {
			return data
		}
	case "png":
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		if err := enc.Encode(&buf, img); err != nil {
			return data
		}
	}

	// Garde le résultat compressé seulement s'il est effectivement plus
	// petit — un PNG déjà bien optimisé ne doit pas repartir plus lourd.
	if buf.Len() >= len(data) {
		return data
	}
	return buf.Bytes()
}
