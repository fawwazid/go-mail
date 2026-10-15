package mail

import (
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

// Attachment represents an email attachment.
type Attachment struct {
	Filename    string
	ContentType string
	Content     []byte
	Inline      bool
}

// FileOption defines options for attaching files.
type FileOption func(*Attachment)

// WithFileName overrides the filename of the attachment.
func WithFileName(name string) FileOption {
	return func(a *Attachment) {
		a.Filename = name
	}
}

// WithInline sets the attachment as inline.
func WithInline(inline bool) FileOption {
	return func(a *Attachment) {
		a.Inline = inline
	}
}

// WithContentType overrides the detected MIME type of the attachment.
func WithContentType(contentType string) FileOption {
	return func(a *Attachment) {
		a.ContentType = contentType
	}
}

func readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	content, err := io.ReadAll(io.LimitReader(f, MaxMessageSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > MaxMessageSize {
		return nil, fmt.Errorf("%w: attachment exceeds %d bytes", ErrMessageTooLarge, MaxMessageSize)
	}
	return content, nil
}

// CreateAttachmentFromFile creates an Attachment from a file path.
// It reads the file content and sets the filename.
func CreateAttachmentFromFile(path string, opts ...FileOption) (*Attachment, error) {
	content, err := readFile(path)
	if err != nil {
		return nil, err
	}

	filename := filepath.Base(path)
	contentType := mime.TypeByExtension(filepath.Ext(filename))
	if contentType == "" {
		contentType = http.DetectContentType(content)
	}

	a := &Attachment{
		Filename:    filename,
		Content:     content,
		ContentType: contentType,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(a)
		}
	}

	return a, nil
}

// Base64 returns the base64 encoded content of the attachment.
func (a *Attachment) Base64() string {
	return base64.StdEncoding.EncodeToString(a.Content)
}
