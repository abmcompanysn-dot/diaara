package vendormail

import (
	"fmt"
	"io"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

type ReaderConfig struct {
	Host     string // ex: "diarra.app:993"
	Username string
	Password string
	Mailbox  string // ex: "INBOX"
}

// IncomingMessage est un email reçu, déjà parsé, prêt à être rattaché à un
// fil (par l'adresse From) et enregistré par VendorMailRepo.CreateInbound.
type IncomingMessage struct {
	UID       uint32
	FromEmail string
	Subject   string
	Body      string
	MessageID string
	InReplyTo string
}

// FetchNewMessages se connecte en IMAPS, récupère tous les messages dont
// l'UID est strictement supérieur à afterUID, et renvoie leur UID le plus
// élevé (0 si aucun nouveau message) pour que l'appelant persiste ce
// curseur via VendorMailRepo.SetLastUID.
func FetchNewMessages(cfg ReaderConfig, afterUID uint32) ([]IncomingMessage, uint32, error) {
	c, err := client.DialTLS(cfg.Host, nil)
	if err != nil {
		return nil, afterUID, fmt.Errorf("vendormail: imap dial: %w", err)
	}
	defer c.Logout()

	if err := c.Login(cfg.Username, cfg.Password); err != nil {
		return nil, afterUID, fmt.Errorf("vendormail: imap login: %w", err)
	}

	mailbox := cfg.Mailbox
	if mailbox == "" {
		mailbox = "INBOX"
	}
	status, err := c.Select(mailbox, false)
	if err != nil {
		return nil, afterUID, fmt.Errorf("vendormail: imap select: %w", err)
	}
	if status.Messages == 0 {
		return nil, afterUID, nil
	}

	seqset := new(imap.SeqSet)
	seqset.AddRange(afterUID+1, 0) // 0 = pas de borne haute ("*")

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchUid, section.FetchItem()}

	messagesCh := make(chan *imap.Message, 16)
	fetchErrCh := make(chan error, 1)
	go func() {
		fetchErrCh <- c.UidFetch(seqset, items, messagesCh)
	}()

	var results []IncomingMessage
	maxUID := afterUID
	for msg := range messagesCh {
		if msg.Uid <= afterUID {
			continue // le serveur IMAP peut renvoyer des bornes incluses selon l'implémentation
		}
		if msg.Uid > maxUID {
			maxUID = msg.Uid
		}
		body := msg.GetBody(section)
		if body == nil {
			continue
		}
		parsed, err := parseIncoming(body)
		if err != nil {
			continue // email illisible (pub, format exotique) : ignoré plutôt que de bloquer toute la boucle
		}
		parsed.UID = msg.Uid
		results = append(results, parsed)
	}
	if err := <-fetchErrCh; err != nil {
		return nil, afterUID, fmt.Errorf("vendormail: imap fetch: %w", err)
	}

	return results, maxUID, nil
}

func parseIncoming(r io.Reader) (IncomingMessage, error) {
	mr, err := mail.CreateReader(r)
	if err != nil {
		return IncomingMessage{}, err
	}

	header := mr.Header
	fromList, _ := header.AddressList("From")
	fromEmail := ""
	if len(fromList) > 0 {
		fromEmail = strings.ToLower(strings.TrimSpace(fromList[0].Address))
	}
	subject, _ := header.Subject()
	messageID, _ := header.MessageID()
	inReplyTo := header.Get("In-Reply-To")

	var bodyText strings.Builder
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		switch h := part.Header.(type) {
		case *mail.InlineHeader:
			contentType, _, _ := h.ContentType()
			if strings.HasPrefix(contentType, "text/plain") {
				b, _ := io.ReadAll(part.Body)
				bodyText.Write(b)
			}
		}
	}

	if fromEmail == "" {
		return IncomingMessage{}, fmt.Errorf("vendormail: no From address")
	}

	return IncomingMessage{
		FromEmail: fromEmail,
		Subject:   subject,
		Body:      strings.TrimSpace(bodyText.String()),
		MessageID: messageID,
		InReplyTo: strings.TrimSpace(inReplyTo),
	}, nil
}
