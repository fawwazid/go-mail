package mail

import (
	"encoding/base64"
	"io"
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

func readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// CreateAttachmentFromFile creates an Attachment from a file path.
// It reads the file content and sets the filename.
func CreateAttachmentFromFile(path string, opts ...FileOption) (*Attachment, error) {
	content, err := readFile(path)
	if err != nil {
		return nil, err
	}

	filename := filepath.Base(path)
	// Simple content type detection based on extension could be added here or user provided
	// For now defaulting to application/octet-stream if unknown, but better handling usually needed.
	// We'll let the user/internals handle detailed MIME types later or keep it simple.

	a := &Attachment{
		Filename:    filename,
		Content:     content,
		ContentType: "application/octet-stream", // Default, arguably should be detected
	}

	for _, opt := range opts {
		opt(a)
	}

	return a, nil
}

// Base64 returns the base64 encoded content of the attachment
func (a *Attachment) Base64() string {
	return base64.StdEncoding.EncodeToString(a.Content)
}
