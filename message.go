package data

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"strings"
	"time"
)

// MessageID represents the ID of an SMTP message including the hostname part
type MessageID string

// NewMessageID generates a new message ID
func NewMessageID(hostname string) (MessageID, error) {
	size := 32

	rb := make([]byte, size)
	_, err := rand.Read(rb)

	if err != nil {
		return MessageID(""), err
	}

	rs := base64.URLEncoding.EncodeToString(rb)

	return MessageID(rs + "@" + hostname), nil
}

// Messages represents an array of Messages
// - TODO is this even required?
type Messages []Message

// Message represents a parsed SMTP message
type Message struct {
	ID      MessageID
	From    *Path
	To      []*Path
	Content *Content
	Created time.Time
	MIME    *MIMEBody // FIXME refactor to use Content.MIME
	Raw     *SMTPMessage
}

// Path represents an SMTP forward-path or return-path
type Path struct {
	Relays  []string
	Mailbox string
	Domain  string
	Params  string
}

// Content represents the body content of an SMTP message
type Content struct {
	Headers map[string][]string
	Body    string
	Size    int
	MIME    *MIMEBody
}

// SMTPMessage represents a raw SMTP message
type SMTPMessage struct {
	From string
	To   []string
	Data string
	Helo string
}

// MIMEBody represents a collection of MIME parts
type MIMEBody struct {
	Parts []*Content
}

// Parse converts a raw SMTP message to a parsed MIME message
func (m *SMTPMessage) Parse(hostname string) *Message {
	var arr []*Path
	for _, path := range m.To {
		arr = append(arr, PathFromString(path))
	}

	id, _ := NewMessageID(hostname)
	msg := &Message{
		ID:      id,
		From:    PathFromString(m.From),
		To:      arr,
		Content: ContentFromString(m.Data),
		Created: time.Now(),
		Raw:     m,
	}

	if msg.Content.IsMIME() {
		log.Printf("Parsing MIME body")
		msg.MIME = msg.Content.ParseMIMEBody()
	}

	// FIXME shouldn't be setting Message-ID, its a client thing
	msg.Content.Headers["Message-ID"] = []string{string(id)}
	msg.Content.Headers["Received"] = []string{"from " + m.Helo + " by " + hostname + " (Go-MailHog)\r\n          id " + string(id) + "; " + time.Now().Format(time.RFC1123Z)}
	msg.Content.Headers["Return-Path"] = []string{"<" + m.From + ">"}
	return msg
}

// IsMIME detects a valid MIME header
func (content *Content) IsMIME() bool {
	header, ok := content.Headers["Content-Type"]
	if !ok {
		return false
	}
	return strings.HasPrefix(header[0], "multipart/")
}

// ParseMIMEBody parses SMTP message content into multiple MIME parts
func (content *Content) ParseMIMEBody() *MIMEBody {
	var parts []*Content

	if hdr, ok := content.Headers["Content-Type"]; ok {
		if len(hdr) > 0 {
			boundary := extractBoundary(hdr[0])
			var p []string
			if len(boundary) > 0 {
				p = strings.Split(content.Body, "--"+boundary)
				log.Printf("Got boundary: %s", boundary)
			} else {
				log.Printf("Boundary not found: %s", hdr[0])
			}

			for _, s := range p {
				if len(s) > 0 {
					part := ContentFromString(strings.Trim(s, "\r\n"))
					if part.IsMIME() {
						log.Printf("Parsing inner MIME body")
						part.MIME = part.ParseMIMEBody()
					}
					parts = append(parts, part)
				}
			}
		}
	}

	return &MIMEBody{
		Parts: parts,
	}
}

// PathFromString parses a forward-path or reverse-path into its parts
func PathFromString(path string) *Path {
	var relays []string
	email := path
	if strings.Contains(path, ":") {
		x := strings.SplitN(path, ":", 2)
		r, e := x[0], x[1]
		email = e
		relays = strings.Split(r, ",")
	}
	mailbox, domain := "", ""
	if strings.Contains(email, "@") {
		x := strings.SplitN(email, "@", 2)
		mailbox, domain = x[0], x[1]
	} else {
		mailbox = email
	}

	return &Path{
		Relays:  relays,
		Mailbox: mailbox,
		Domain:  domain,
		Params:  "", // FIXME?
	}
}

// ContentFromString parses SMTP content into separate headers and body
func ContentFromString(data string) *Content {
	log.Printf("Parsing Content from string: '%s'", data)
	x := strings.SplitN(data, "\r\n\r\n", 2)
	h := make(map[string][]string, 0)

	if len(x) == 2 {
		headers, body := x[0], x[1]
		hdrs := strings.Split(headers, "\r\n")
		var lastHdr = ""
		for _, hdr := range hdrs {
			if lastHdr != "" && (strings.HasPrefix(hdr, " ") || strings.HasPrefix(hdr, "\t")) {
				h[lastHdr][len(h[lastHdr])-1] = h[lastHdr][len(h[lastHdr])-1] + hdr
			} else if strings.Contains(hdr, ": ") {
				y := strings.SplitN(hdr, ": ", 2)
				key, value := y[0], y[1]
				// TODO multiple header fields
				h[key] = []string{value}
				lastHdr = key
			} else {
				log.Printf("Found invalid header: '%s'", hdr)
			}
		}
		return &Content{
			Size:    len(data),
			Headers: h,
			Body:    body,
		}
	}
	return &Content{
		Size:    len(data),
		Headers: h,
		Body:    x[0],
	}
}

// extractBoundary extract boundary string in contentType.
// It returns empty string if no valid boundary found
func extractBoundary(contentType string) string {
	var boundary string
	// first searching for the 'boundary=' token
	boundaryIdx := strings.Index(contentType, "boundary=")
	if boundaryIdx > -1 && len(contentType) > boundaryIdx+9 {
		// then id check if what is the next char after '='
		firstCharIdx := boundaryIdx + 9
		if contentType[firstCharIdx] == '"' {
			// ok, searching for the close quote
			closeQuoteIdx := strings.Index(contentType[firstCharIdx+1:], "\"")
			if closeQuoteIdx > -1 {
				boundary = contentType[firstCharIdx+1 : firstCharIdx+closeQuoteIdx+1]
			}
		} else {
			// that mean the boundary not quoted, check for ';' or 'space'
			// or any kind of newline
			terminateIdx := strings.IndexAny(contentType[firstCharIdx:], "; \r\n")
			if terminateIdx > -1 {
				boundary = contentType[firstCharIdx : firstCharIdx+terminateIdx]
			} else {
				// the boundary is the last param of contentType
				boundary = contentType[firstCharIdx:]
			}
		}
	}

	return boundary
}
