package mail

// EncryptionType defines the type of encryption to use for the SMTP connection.
type EncryptionType string

const (
	// EncryptionNone uses no encryption.
	EncryptionNone EncryptionType = "NONE"
	// EncryptionSSL uses implicit SSL/TLS (usually port 465).
	EncryptionSSL EncryptionType = "SSL"
	// EncryptionTLS uses explicit TLS (STARTTLS, usually port 587).
	EncryptionTLS EncryptionType = "TLS"
)
