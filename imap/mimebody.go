package imap

import (
	"bytes"
	"io"
	"strings"

	"github.com/emersion/go-message"
	"mail0/store"
)

// parseRFC822 walks a MIME message and extracts plain text, HTML, and simple attachment metadata.
func parseRFC822(raw []byte) (plain, html string, attachments []store.AttachmentInfo) {
	if len(raw) == 0 {
		return "", "", nil
	}
	e, err := message.Read(bytes.NewReader(raw))
	if err != nil {
		// Unparseable — show raw body as plain text so the UI does not inject MIME as HTML.
		return string(raw), "", nil
	}

	var plainParts, htmlParts []string
	_ = e.Walk(func(path []int, entity *message.Entity, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entity.MultipartReader() != nil {
			return nil
		}

		mt, typeParams, err := entity.Header.ContentType()
		if err != nil {
			mt = ""
		}
		mtLower := strings.ToLower(strings.TrimSpace(strings.Split(mt, ";")[0]))

		body, err := io.ReadAll(entity.Body)
		if err != nil {
			return nil
		}

		disp, dispParams, _ := entity.Header.ContentDisposition()
		fn := dispParams["filename"]
		if fn == "" {
			fn = typeParams["name"]
		}

		isAttachment := disp == "attachment" ||
			(fn != "" && mtLower != "text/plain" && mtLower != "text/html")

		switch mtLower {
		case "text/plain":
			if !isAttachment {
				plainParts = append(plainParts, string(body))
			}
		case "text/html":
			if !isAttachment {
				htmlParts = append(htmlParts, string(body))
			}
		default:
			if isAttachment || (fn != "" && len(body) > 0) {
				attachments = append(attachments, store.AttachmentInfo{
					Filename: fn,
					Size:     uint32(len(body)),
					MimeType: mtLower,
				})
			}
		}
		return nil
	})

	return lastNonEmpty(plainParts), lastNonEmpty(htmlParts), attachments
}

func lastNonEmpty(parts []string) string {
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.TrimSpace(parts[i]) != "" {
			return parts[i]
		}
	}
	return ""
}
