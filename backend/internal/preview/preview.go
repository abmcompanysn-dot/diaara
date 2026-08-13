// Package preview génère des aperçus filigranés d'un fichier produit avant
// achat : 3 premières pages pour un PDF, image réduite pour une image, et
// un court extrait pour l'audio/la vidéo. Utilise des outils externes
// (poppler-utils, ImageMagick, ffmpeg) plutôt que des bibliothèques Go —
// plus fiable pour le rendu PDF et le transcodage média.
package preview

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Storage est l'interface minimale requise pour lire le fichier source et
// écrire les aperçus générés.
type Storage interface {
	Download(ctx context.Context, key string) ([]byte, error)
	Upload(ctx context.Context, key string, data []byte) error
}

const watermarkText = "DIARRA — APERÇU"

var (
	pdfExts   = map[string]bool{".pdf": true}
	imageExts = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
	audioExts = map[string]bool{".mp3": true, ".wav": true, ".m4a": true, ".ogg": true, ".flac": true}
	videoExts = map[string]bool{".mp4": true, ".mov": true, ".webm": true, ".mkv": true, ".avi": true}
)

// Generate produit les aperçus pour le fichier désigné par fileKey et les
// téléverse sous previews/{vendorID}/{productID}/. Retourne une liste vide
// (sans erreur) si le type de fichier n'a pas d'aperçu géré.
func Generate(ctx context.Context, st Storage, vendorID, productID, fileKey string) ([]string, error) {
	ext := strings.ToLower(filepath.Ext(fileKey))

	data, err := st.Download(ctx, fileKey)
	if err != nil {
		return nil, fmt.Errorf("download source: %w", err)
	}

	dir, err := os.MkdirTemp("", "diarra-preview-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	src := filepath.Join(dir, "source"+ext)
	if err := os.WriteFile(src, data, 0o600); err != nil {
		return nil, err
	}

	switch {
	case pdfExts[ext]:
		return generatePDF(ctx, st, dir, src, vendorID, productID)
	case imageExts[ext]:
		return generateImage(ctx, st, dir, src, vendorID, productID)
	case audioExts[ext]:
		return generateAudio(ctx, st, dir, src, vendorID, productID)
	case videoExts[ext]:
		return generateVideo(ctx, st, dir, src, vendorID, productID)
	default:
		return nil, nil
	}
}

func run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

const watermarkFont = "/usr/share/fonts/dejavu/DejaVuSans-Bold.ttf"

func watermarkImage(ctx context.Context, srcPath, dstPath string) error {
	return run(ctx, "convert", srcPath,
		"-resize", "1400x1400>",
		"-gravity", "South",
		"-font", watermarkFont,
		"-pointsize", "34",
		"-fill", "rgba(255,255,255,0.65)",
		"-stroke", "rgba(0,0,0,0.4)", "-strokewidth", "1",
		"-annotate", "+0+24", watermarkText,
		dstPath,
	)
}

// generatePDF rend les 3 premières pages en image et les filigrane.
func generatePDF(ctx context.Context, st Storage, dir, src, vendorID, productID string) ([]string, error) {
	outPrefix := filepath.Join(dir, "page")
	if err := run(ctx, "pdftoppm", "-png", "-f", "1", "-l", "3", "-r", "120", src, outPrefix); err != nil {
		return nil, err
	}

	matches, _ := filepath.Glob(outPrefix + "-*.png")
	sort.Strings(matches)

	var keys []string
	for i, path := range matches {
		wm := path + ".wm.png"
		if err := watermarkImage(ctx, path, wm); err != nil {
			return nil, err
		}
		data, err := os.ReadFile(wm)
		if err != nil {
			return nil, err
		}
		key := fmt.Sprintf("previews/%s/%s/page-%d.png", vendorID, productID, i+1)
		if err := st.Upload(ctx, key, data); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// generateImage produit une version réduite et filigranée de l'image.
func generateImage(ctx context.Context, st Storage, dir, src, vendorID, productID string) ([]string, error) {
	dst := filepath.Join(dir, "preview.jpg")
	if err := watermarkImage(ctx, src, dst); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("previews/%s/%s/preview.jpg", vendorID, productID)
	if err := st.Upload(ctx, key, data); err != nil {
		return nil, err
	}
	return []string{key}, nil
}

// generateAudio extrait les 25 premières secondes (pas de filigrane visuel
// possible sur de l'audio ; la durée limitée suffit comme aperçu).
func generateAudio(ctx context.Context, st Storage, dir, src, vendorID, productID string) ([]string, error) {
	dst := filepath.Join(dir, "preview.mp3")
	if err := run(ctx, "ffmpeg", "-y", "-i", src, "-t", "25", "-vn", "-acodec", "libmp3lame", "-b:a", "128k", dst); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("previews/%s/%s/preview.mp3", vendorID, productID)
	if err := st.Upload(ctx, key, data); err != nil {
		return nil, err
	}
	return []string{key}, nil
}

// generateVideo extrait les 25 premières secondes avec le filigrane
// incrusté en overlay (drawtext, police DejaVu via fontconfig).
func generateVideo(ctx context.Context, st Storage, dir, src, vendorID, productID string) ([]string, error) {
	dst := filepath.Join(dir, "preview.mp4")
	drawtext := "drawtext=fontfile=" + watermarkFont + ":text='" + watermarkText + "':fontcolor=white@0.7:fontsize=28:x=(w-text_w)/2:y=h-th-24:box=1:boxcolor=black@0.35:boxborderw=8"
	if err := run(ctx, "ffmpeg", "-y", "-i", src, "-t", "25",
		"-vf", drawtext,
		"-c:v", "libx264", "-preset", "veryfast", "-crf", "28",
		"-c:a", "aac", "-b:a", "96k",
		dst,
	); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		return nil, err
	}
	key := fmt.Sprintf("previews/%s/%s/preview.mp4", vendorID, productID)
	if err := st.Upload(ctx, key, data); err != nil {
		return nil, err
	}
	return []string{key}, nil
}
