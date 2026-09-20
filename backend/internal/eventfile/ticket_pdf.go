package eventfile

import (
	"bytes"
	"fmt"

	"github.com/go-pdf/fpdf"
	qrcode "github.com/skip2/go-qrcode"
)

// Ticket décrit un billet PDF individuel, généré à la demande par vente
// (pas par produit — voir EventTicketHandler) : chaque acheteur a son propre
// QR code unique, indispensable pour distinguer les billets à l'entrée.
type Ticket struct {
	EventTitle string
	OfferTitle string
	BuyerName  string
	PriceCFA   int
	// CheckInURL : encodée dans le QR code. Pointe vers la page de scan
	// admin/vendeur (voir EventTicketRepo.check_in_token) — jamais l'ID de
	// vente en clair, pour ne pas rendre un billet devinable/forgeable.
	CheckInURL string
}

// BuildPDF génère le billet PDF (titre, acheteur, prix, QR de vérification).
// Pure Go (fpdf + go-qrcode), aucune dépendance système (pas de binaire
// externe à installer dans l'image Docker) — voir Dockerfile.
func BuildPDF(t Ticket) ([]byte, error) {
	qrPNG, err := qrcode.Encode(t.CheckInURL, qrcode.Medium, 300)
	if err != nil {
		return nil, fmt.Errorf("génération QR: %w", err)
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 20)
	pdf.SetTextColor(5, 32, 24) // #052018
	pdf.CellFormat(0, 14, t.EventTitle, "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 13)
	pdf.SetTextColor(15, 122, 80) // #0f7a50
	pdf.CellFormat(0, 8, t.OfferTitle, "", 1, "C", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont("Helvetica", "", 11)
	pdf.SetTextColor(10, 50, 37) // #0a3225
	pdf.CellFormat(0, 7, fmt.Sprintf("Billet nominatif : %s", t.BuyerName), "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 7, fmt.Sprintf("%d FCFA", t.PriceCFA), "", 1, "C", false, 0, "")
	pdf.Ln(8)

	// QR centré. fpdf.RegisterImageOptionsReader lit depuis un io.Reader — on
	// enregistre les octets PNG sous un nom d'image virtuel (pas de fichier
	// temporaire sur disque, le pod tourne en lecture seule).
	imgOpt := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	pdf.RegisterImageOptionsReader("qr", imgOpt, bytes.NewReader(qrPNG))
	qrSize := 70.0
	pageW, _ := pdf.GetPageSize()
	x := (pageW - qrSize) / 2
	pdf.ImageOptions("qr", x, pdf.GetY(), qrSize, qrSize, false, imgOpt, 0, "")
	pdf.SetY(pdf.GetY() + qrSize + 6)

	pdf.SetFont("Helvetica", "I", 9)
	pdf.SetTextColor(107, 124, 116) // #6b7c74
	pdf.CellFormat(0, 6, "Présentez ce QR code à l'entrée de l'événement.", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("rendu PDF: %w", err)
	}
	return buf.Bytes(), nil
}
