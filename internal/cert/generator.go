package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// CertificateInfo holds information about generated certificates
type CertificateInfo struct {
	CAFingerprint   string
	CertFingerprint string
	ValidFrom       time.Time
	ValidUntil      time.Time
	Subject         string
}

// GenerateSelfSignedCerts generates a self-signed CA and agent certificate
// This is used for bootstrap mode when agent runs autonomously
func GenerateSelfSignedCerts(outputDir, hostname string) (*CertificateInfo, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate CA
	caKey, caCert, err := generateCA(hostname)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA: %w", err)
	}

	// Generate agent certificate signed by CA
	agentKey, agentCert, err := generateAgentCert(hostname, caKey, caCert)
	if err != nil {
		return nil, fmt.Errorf("failed to generate agent cert: %w", err)
	}

	// Save CA cert
	caPath := filepath.Join(outputDir, "ca.crt")
	if err := saveCertificate(caPath, caCert); err != nil {
		return nil, fmt.Errorf("failed to save CA cert: %w", err)
	}

	// Save agent cert
	certPath := filepath.Join(outputDir, "agent.crt")
	if err := saveCertificate(certPath, agentCert); err != nil {
		return nil, fmt.Errorf("failed to save agent cert: %w", err)
	}

	// Save agent key
	keyPath := filepath.Join(outputDir, "agent.key")
	if err := savePrivateKey(keyPath, agentKey); err != nil {
		return nil, fmt.Errorf("failed to save agent key: %w", err)
	}

	// Calculate fingerprints
	caFingerprint := calculateFingerprint(caCert)
	certFingerprint := calculateFingerprint(agentCert)

	info := &CertificateInfo{
		CAFingerprint:   caFingerprint,
		CertFingerprint: certFingerprint,
		ValidFrom:       agentCert.NotBefore,
		ValidUntil:      agentCert.NotAfter,
		Subject:         agentCert.Subject.CommonName,
	}

	return info, nil
}

// generateCA generates a self-signed CA certificate
func generateCA(hostname string) (*ecdsa.PrivateKey, *x509.Certificate, error) {
	// Generate private key (ECDSA P-256)
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Generate serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Create CA certificate template
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // Valid for 1 year

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:  []string{"ocserv-agent"},
			CommonName:    fmt.Sprintf("ocserv-agent-ca-%s", hostname),
			Country:       []string{"US"},
			Locality:      []string{"Bootstrap"},
			Province:      []string{"Self-Signed"},
			StreetAddress: []string{},
			PostalCode:    []string{},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	// Self-sign the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return priv, cert, nil
}

// generateAgentCert generates an agent certificate signed by CA
func generateAgentCert(hostname string, caKey *ecdsa.PrivateKey, caCert *x509.Certificate) (*ecdsa.PrivateKey, *x509.Certificate, error) {
	// Generate private key
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Generate serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Create agent certificate template
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour) // Valid for 1 year

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization:  []string{"ocserv-agent"},
			CommonName:    fmt.Sprintf("ocserv-agent-%s", hostname),
			Country:       []string{"US"},
			Locality:      []string{"Bootstrap"},
			Province:      []string{"Self-Signed"},
			StreetAddress: []string{},
			PostalCode:    []string{},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              []string{hostname, "localhost"},
		IPAddresses:           nil, // Add IPs if needed
	}

	// Sign with CA
	certDER, err := x509.CreateCertificate(rand.Reader, template, caCert, &priv.PublicKey, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return priv, cert, nil
}

// saveCertificate saves a certificate to a PEM file
func saveCertificate(path string, cert *x509.Certificate) error {
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	})

	if err := os.WriteFile(path, certPEM, 0644); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	return nil
}

// savePrivateKey saves a private key to a PEM file
func savePrivateKey(path string, key *ecdsa.PrivateKey) error {
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	if err := os.WriteFile(path, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}

// calculateFingerprint calculates SHA256 fingerprint of a certificate
func calculateFingerprint(cert *x509.Certificate) string {
	hash := sha256.Sum256(cert.Raw)
	fingerprint := hex.EncodeToString(hash[:])

	// Format as SHA256:xx:xx:xx:...
	formatted := "SHA256:"
	for i := 0; i < len(fingerprint); i += 2 {
		if i > 0 {
			formatted += ":"
		}
		formatted += fingerprint[i : i+2]
	}

	return formatted
}

// CertsExist checks if certificate files exist
func CertsExist(certFile, keyFile, caFile string) bool {
	_, certErr := os.Stat(certFile)
	_, keyErr := os.Stat(keyFile)
	_, caErr := os.Stat(caFile)

	return certErr == nil && keyErr == nil && caErr == nil
}
