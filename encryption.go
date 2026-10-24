package mail

// EncryptionType defines the type of encryption to use for the SMTP connection.
type EncryptionType string

const (
	// EncryptionImplicitTLS establishes TLS before SMTP (usually port 465).
	EncryptionImplicitTLS EncryptionType = "IMPLICIT_TLS"
	// EncryptionSTARTTLS uses explicit TLS (STARTTLS, usually port 587).
	EncryptionSTARTTLS EncryptionType = "STARTTLS"
)
