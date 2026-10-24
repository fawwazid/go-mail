package mail

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/fips140"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"
)

func TestRejectsMIMEHeaderInjection(t *testing.T) {
	for _, contentType := range []string{"text/plain\r\nX-Injected: yes", "text/plain;\r\n charset=utf-8"} {
		msg := validTestMessage()
		msg.ContentType = contentType
		if _, err := msg.Bytes(); err == nil {
			t.Fatal("accepted body header line break")
		}
		msg = validTestMessage().AddAttachmentData("file.txt", nil, contentType)
		if _, err := msg.Bytes(); err == nil {
			t.Fatal("accepted attachment header line break")
		}
	}
}

func testTLSHandshake(t *testing.T, version uint16, suite uint16, shouldSucceed bool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"},
		DNSNames: []string{"localhost"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(cert)
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	_ = serverConn.SetDeadline(time.Now().Add(5 * time.Second))
	_ = clientConn.SetDeadline(time.Now().Add(5 * time.Second))
	serverConfig := &tls.Config{
		MinVersion: version, MaxVersion: version,
		CipherSuites: []uint16{suite}, SessionTicketsDisabled: true,
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}},
	}
	serverDone := make(chan error, 1)
	go func() { serverDone <- tls.Server(serverConn, serverConfig).Handshake() }()
	config := NewClient("localhost", 465, "", "", EncryptionImplicitTLS).tlsConfig()
	config.RootCAs = roots
	err = tls.Client(clientConn, config).Handshake()
	serverErr := <-serverDone
	if shouldSucceed && (err != nil || serverErr != nil) {
		t.Fatalf("TLS %x handshake = %v, server = %v", version, err, serverErr)
	}
	if !shouldSucceed && err == nil {
		t.Fatal("accepted disallowed TLS configuration")
	}
}

func TestTLSPolicy(t *testing.T) {
	testTLSHandshake(t, tls.VersionTLS12, tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, true)
	testTLSHandshake(t, tls.VersionTLS13, 0, true)
	testTLSHandshake(t, tls.VersionTLS11, tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA, false)
	testTLSHandshake(t, tls.VersionTLS12, tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256, false)
}

func TestFIPSPolicy(t *testing.T) {
	if !fips140.Enabled() {
		t.Skip("requires FIPS mode")
	}
	testTLSHandshake(t, tls.VersionTLS12, tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, true)
	testTLSHandshake(t, tls.VersionTLS13, 0, true)
}
