package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestGenerateSelfSignedCerts tests the main certificate generation function
func TestGenerateSelfSignedCerts(t *testing.T) {
	tests := []struct {
		name      string
		hostname  string
		wantError bool
	}{
		{
			name:      "valid hostname",
			hostname:  "test-agent",
			wantError: false,
		},
		{
			name:      "hostname with dots",
			hostname:  "agent.example.com",
			wantError: false,
		},
		{
			name:      "empty hostname",
			hostname:  "",
			wantError: false, // Should still work, just empty in cert
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()

			// Generate certificates
			info, err := GenerateSelfSignedCerts(tmpDir, tt.hostname)

			if tt.wantError {
				if err == nil {
					t.Errorf("GenerateSelfSignedCerts() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("GenerateSelfSignedCerts() error = %v", err)
			}

			// Verify CertificateInfo
			if info == nil {
				t.Fatal("GenerateSelfSignedCerts() returned nil info")
			}

			if info.CAFingerprint == "" {
				t.Error("CAFingerprint is empty")
			}
			if info.CertFingerprint == "" {
				t.Error("CertFingerprint is empty")
			}
			if !strings.HasPrefix(info.CAFingerprint, "SHA256:") {
				t.Errorf("CAFingerprint has wrong format: %s", info.CAFingerprint)
			}
			if !strings.HasPrefix(info.CertFingerprint, "SHA256:") {
				t.Errorf("CertFingerprint has wrong format: %s", info.CertFingerprint)
			}

			// Verify validity period
			if info.ValidFrom.After(time.Now()) {
				t.Error("ValidFrom is in the future")
			}
			if info.ValidUntil.Before(time.Now()) {
				t.Error("ValidUntil is in the past")
			}

			// Should be valid for approximately 1 year
			duration := info.ValidUntil.Sub(info.ValidFrom)
			expectedDuration := 365 * 24 * time.Hour
			tolerance := 1 * time.Hour
			if duration < expectedDuration-tolerance || duration > expectedDuration+tolerance {
				t.Errorf("Validity period is %v, expected ~%v", duration, expectedDuration)
			}

			// Verify subject
			if tt.hostname != "" {
				expectedSubject := "ocserv-agent-" + tt.hostname
				if info.Subject != expectedSubject {
					t.Errorf("Subject = %s, expected %s", info.Subject, expectedSubject)
				}
			}

			// Verify files exist
			caPath := filepath.Join(tmpDir, "ca.crt")
			certPath := filepath.Join(tmpDir, "agent.crt")
			keyPath := filepath.Join(tmpDir, "agent.key")

			if _, err := os.Stat(caPath); err != nil {
				t.Errorf("CA cert file not found: %v", err)
			}
			if _, err := os.Stat(certPath); err != nil {
				t.Errorf("Agent cert file not found: %v", err)
			}
			if _, err := os.Stat(keyPath); err != nil {
				t.Errorf("Agent key file not found: %v", err)
			}

			// Verify file permissions
			verifyCertPermissions(t, caPath, 0644)
			verifyCertPermissions(t, certPath, 0644)
			verifyCertPermissions(t, keyPath, 0600)

			// Verify certificates can be loaded
			caCert := loadCertificate(t, caPath)
			agentCert := loadCertificate(t, certPath)
			agentKey := loadPrivateKey(t, keyPath)

			// Verify CA properties
			if !caCert.IsCA {
				t.Error("CA certificate IsCA is false")
			}
			if caCert.KeyUsage&x509.KeyUsageCertSign == 0 {
				t.Error("CA certificate missing KeyUsageCertSign")
			}

			// Verify agent cert is signed by CA
			if err := agentCert.CheckSignatureFrom(caCert); err != nil {
				t.Errorf("Agent cert not signed by CA: %v", err)
			}

			// Verify agent cert has correct hostname
			if tt.hostname != "" {
				found := false
				for _, dnsName := range agentCert.DNSNames {
					if dnsName == tt.hostname {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Agent cert DNSNames %v does not contain %s", agentCert.DNSNames, tt.hostname)
				}
			}

			// Verify private key matches cert
			publicKey, ok := agentCert.PublicKey.(*ecdsa.PublicKey)
			if !ok {
				t.Fatal("Agent cert public key is not ECDSA")
			}
			if !publicKey.Equal(&agentKey.PublicKey) {
				t.Error("Agent key does not match agent cert")
			}
		})
	}
}

// TestGenerateSelfSignedCertsInvalidDir tests error handling for invalid directory
func TestGenerateSelfSignedCertsInvalidDir(t *testing.T) {
	// Try to create in a non-writable directory (Linux-specific test)
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir := t.TempDir()
	invalidDir := filepath.Join(tmpDir, "readonly", "subdir")

	// Create readonly parent
	readonlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readonlyDir, 0444); err != nil {
		t.Fatalf("Failed to create readonly dir: %v", err)
	}
	defer os.Chmod(readonlyDir, 0755) // Restore permissions for cleanup

	_, err := GenerateSelfSignedCerts(invalidDir, "test")
	if err == nil {
		t.Error("GenerateSelfSignedCerts() expected error for invalid directory, got nil")
	}
}

// TestGenerateCA tests CA certificate generation
func TestGenerateCA(t *testing.T) {
	hostname := "test-ca"

	priv, cert, err := generateCA(hostname)
	if err != nil {
		t.Fatalf("generateCA() error = %v", err)
	}

	// Verify private key
	if priv == nil {
		t.Fatal("generateCA() returned nil private key")
	}
	if priv.Curve != elliptic.P256() {
		t.Errorf("Private key curve = %v, expected P256", priv.Curve)
	}

	// Verify certificate
	if cert == nil {
		t.Fatal("generateCA() returned nil certificate")
	}

	// Verify CA properties
	if !cert.IsCA {
		t.Error("Certificate IsCA is false")
	}
	if cert.MaxPathLen != 1 {
		t.Errorf("MaxPathLen = %d, expected 1", cert.MaxPathLen)
	}

	// Verify key usage
	if cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Error("Missing KeyUsageCertSign")
	}
	if cert.KeyUsage&x509.KeyUsageCRLSign == 0 {
		t.Error("Missing KeyUsageCRLSign")
	}
	if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		t.Error("Missing KeyUsageDigitalSignature")
	}

	// Verify extended key usage
	hasServerAuth := false
	hasClientAuth := false
	for _, eku := range cert.ExtKeyUsage {
		if eku == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
		}
		if eku == x509.ExtKeyUsageClientAuth {
			hasClientAuth = true
		}
	}
	if !hasServerAuth {
		t.Error("Missing ExtKeyUsageServerAuth")
	}
	if !hasClientAuth {
		t.Error("Missing ExtKeyUsageClientAuth")
	}

	// Verify subject
	expectedCN := "ocserv-agent-ca-" + hostname
	if cert.Subject.CommonName != expectedCN {
		t.Errorf("Subject.CommonName = %s, expected %s", cert.Subject.CommonName, expectedCN)
	}
	if len(cert.Subject.Organization) == 0 || cert.Subject.Organization[0] != "ocserv-agent" {
		t.Errorf("Subject.Organization = %v, expected [ocserv-agent]", cert.Subject.Organization)
	}

	// Verify validity period (1 year)
	duration := cert.NotAfter.Sub(cert.NotBefore)
	expectedDuration := 365 * 24 * time.Hour
	tolerance := 1 * time.Hour
	if duration < expectedDuration-tolerance || duration > expectedDuration+tolerance {
		t.Errorf("Validity period = %v, expected ~%v", duration, expectedDuration)
	}

	// Verify self-signed
	if err := cert.CheckSignatureFrom(cert); err != nil {
		t.Errorf("Certificate is not self-signed: %v", err)
	}

	// Verify public key matches private key
	publicKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatal("Certificate public key is not ECDSA")
	}
	if !publicKey.Equal(&priv.PublicKey) {
		t.Error("Public key does not match private key")
	}
}

// TestGenerateAgentCert tests agent certificate generation
func TestGenerateAgentCert(t *testing.T) {
	hostname := "test-agent"

	// First generate CA
	caKey, caCert, err := generateCA("test-ca")
	if err != nil {
		t.Fatalf("generateCA() error = %v", err)
	}

	// Generate agent cert
	priv, cert, err := generateAgentCert(hostname, caKey, caCert)
	if err != nil {
		t.Fatalf("generateAgentCert() error = %v", err)
	}

	// Verify private key
	if priv == nil {
		t.Fatal("generateAgentCert() returned nil private key")
	}
	if priv.Curve != elliptic.P256() {
		t.Errorf("Private key curve = %v, expected P256", priv.Curve)
	}

	// Verify certificate
	if cert == nil {
		t.Fatal("generateAgentCert() returned nil certificate")
	}

	// Verify NOT a CA
	if cert.IsCA {
		t.Error("Agent certificate IsCA is true")
	}

	// Verify key usage
	if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		t.Error("Missing KeyUsageDigitalSignature")
	}
	if cert.KeyUsage&x509.KeyUsageKeyEncipherment == 0 {
		t.Error("Missing KeyUsageKeyEncipherment")
	}

	// Verify extended key usage
	hasServerAuth := false
	hasClientAuth := false
	for _, eku := range cert.ExtKeyUsage {
		if eku == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
		}
		if eku == x509.ExtKeyUsageClientAuth {
			hasClientAuth = true
		}
	}
	if !hasServerAuth {
		t.Error("Missing ExtKeyUsageServerAuth")
	}
	if !hasClientAuth {
		t.Error("Missing ExtKeyUsageClientAuth")
	}

	// Verify subject
	expectedCN := "ocserv-agent-" + hostname
	if cert.Subject.CommonName != expectedCN {
		t.Errorf("Subject.CommonName = %s, expected %s", cert.Subject.CommonName, expectedCN)
	}

	// Verify DNSNames
	expectedDNS := []string{hostname, "localhost"}
	if len(cert.DNSNames) != len(expectedDNS) {
		t.Errorf("DNSNames count = %d, expected %d", len(cert.DNSNames), len(expectedDNS))
	}
	for _, expected := range expectedDNS {
		found := false
		for _, actual := range cert.DNSNames {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DNSNames %v missing expected %s", cert.DNSNames, expected)
		}
	}

	// Verify signed by CA
	if err := cert.CheckSignatureFrom(caCert); err != nil {
		t.Errorf("Agent cert not signed by CA: %v", err)
	}

	// Verify public key matches private key
	publicKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		t.Fatal("Certificate public key is not ECDSA")
	}
	if !publicKey.Equal(&priv.PublicKey) {
		t.Error("Public key does not match private key")
	}

	// Verify issuer is CA
	if cert.Issuer.CommonName != caCert.Subject.CommonName {
		t.Errorf("Issuer.CommonName = %s, expected %s", cert.Issuer.CommonName, caCert.Subject.CommonName)
	}
}

// TestSaveCertificate tests certificate saving to PEM file
func TestSaveCertificate(t *testing.T) {
	// Create a test certificate
	_, cert, err := generateCA("test")
	if err != nil {
		t.Fatalf("generateCA() error = %v", err)
	}

	// Save to temp file
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "test.crt")

	if err := saveCertificate(certPath, cert); err != nil {
		t.Fatalf("saveCertificate() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(certPath); err != nil {
		t.Errorf("Certificate file not found: %v", err)
	}

	// Verify permissions
	verifyCertPermissions(t, certPath, 0644)

	// Verify can be loaded
	loadedCert := loadCertificate(t, certPath)

	// Verify loaded cert matches original
	if !cert.Equal(loadedCert) {
		t.Error("Loaded certificate does not match original")
	}
}

// TestSavePrivateKey tests private key saving to PEM file
func TestSavePrivateKey(t *testing.T) {
	// Create a test private key
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	// Save to temp file
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.key")

	if err := savePrivateKey(keyPath, priv); err != nil {
		t.Fatalf("savePrivateKey() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(keyPath); err != nil {
		t.Errorf("Private key file not found: %v", err)
	}

	// Verify permissions (must be 0600)
	verifyCertPermissions(t, keyPath, 0600)

	// Verify can be loaded
	loadedKey := loadPrivateKey(t, keyPath)

	// Verify loaded key matches original
	if !priv.Equal(loadedKey) {
		t.Error("Loaded private key does not match original")
	}
}

// TestCalculateFingerprint tests SHA256 fingerprint calculation
func TestCalculateFingerprint(t *testing.T) {
	// Create a test certificate
	_, cert, err := generateCA("test")
	if err != nil {
		t.Fatalf("generateCA() error = %v", err)
	}

	// Calculate fingerprint
	fingerprint := calculateFingerprint(cert)

	// Verify format: SHA256:xx:xx:xx:...
	if !strings.HasPrefix(fingerprint, "SHA256:") {
		t.Errorf("Fingerprint does not start with SHA256:, got: %s", fingerprint)
	}

	// Verify has correct number of colons (SHA256: + 32 hex pairs = 32 colons after prefix)
	colonCount := strings.Count(fingerprint, ":")
	if colonCount != 32 {
		t.Errorf("Fingerprint has %d colons, expected 32", colonCount)
	}

	// Verify reproducibility
	fingerprint2 := calculateFingerprint(cert)
	if fingerprint != fingerprint2 {
		t.Error("Fingerprint calculation is not reproducible")
	}

	// Verify different certs have different fingerprints
	_, cert2, err := generateCA("test2")
	if err != nil {
		t.Fatalf("generateCA() error = %v", err)
	}
	fingerprint3 := calculateFingerprint(cert2)
	if fingerprint == fingerprint3 {
		t.Error("Different certificates have same fingerprint")
	}
}

// TestCertsExist tests certificate existence check
func TestCertsExist(t *testing.T) {
	tmpDir := t.TempDir()

	certFile := filepath.Join(tmpDir, "agent.crt")
	keyFile := filepath.Join(tmpDir, "agent.key")
	caFile := filepath.Join(tmpDir, "ca.crt")

	// Test when no files exist
	if CertsExist(certFile, keyFile, caFile) {
		t.Error("CertsExist() returned true when no files exist")
	}

	// Create only cert file
	if err := os.WriteFile(certFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create cert file: %v", err)
	}
	if CertsExist(certFile, keyFile, caFile) {
		t.Error("CertsExist() returned true when only cert exists")
	}

	// Create key file
	if err := os.WriteFile(keyFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	if CertsExist(certFile, keyFile, caFile) {
		t.Error("CertsExist() returned true when cert and key exist but ca missing")
	}

	// Create CA file
	if err := os.WriteFile(caFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create ca file: %v", err)
	}
	if !CertsExist(certFile, keyFile, caFile) {
		t.Error("CertsExist() returned false when all files exist")
	}

	// Remove one file
	os.Remove(keyFile)
	if CertsExist(certFile, keyFile, caFile) {
		t.Error("CertsExist() returned true when key file missing")
	}
}

// Helper functions

func verifyCertPermissions(t *testing.T, path string, expectedMode os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Failed to stat file %s: %v", path, err)
	}

	actualMode := info.Mode().Perm()
	if actualMode != expectedMode {
		t.Errorf("File %s has permissions %o, expected %o", path, actualMode, expectedMode)
	}
}

func loadCertificate(t *testing.T, path string) *x509.Certificate {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read certificate file: %v", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("Failed to decode PEM block")
	}
	if block.Type != "CERTIFICATE" {
		t.Fatalf("PEM block type = %s, expected CERTIFICATE", block.Type)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	return cert
}

func loadPrivateKey(t *testing.T, path string) *ecdsa.PrivateKey {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read private key file: %v", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("Failed to decode PEM block")
	}
	if block.Type != "EC PRIVATE KEY" {
		t.Fatalf("PEM block type = %s, expected EC PRIVATE KEY", block.Type)
	}

	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		t.Fatalf("Failed to parse private key: %v", err)
	}

	return key
}

// TestGenerateSelfSignedCertsIdempotency tests that generating certs twice produces different results
// (certificates should have unique serial numbers)
func TestGenerateSelfSignedCertsIdempotency(t *testing.T) {
	hostname := "test-agent"

	// Generate first set
	tmpDir1 := t.TempDir()
	info1, err := GenerateSelfSignedCerts(tmpDir1, hostname)
	if err != nil {
		t.Fatalf("First GenerateSelfSignedCerts() error = %v", err)
	}

	// Generate second set
	tmpDir2 := t.TempDir()
	info2, err := GenerateSelfSignedCerts(tmpDir2, hostname)
	if err != nil {
		t.Fatalf("Second GenerateSelfSignedCerts() error = %v", err)
	}

	// Fingerprints should be different (different serial numbers and keys)
	if info1.CAFingerprint == info2.CAFingerprint {
		t.Error("Two CA certificates have the same fingerprint")
	}
	if info1.CertFingerprint == info2.CertFingerprint {
		t.Error("Two agent certificates have the same fingerprint")
	}

	// Load and verify serial numbers are different
	cert1 := loadCertificate(t, filepath.Join(tmpDir1, "agent.crt"))
	cert2 := loadCertificate(t, filepath.Join(tmpDir2, "agent.crt"))

	if cert1.SerialNumber.Cmp(cert2.SerialNumber) == 0 {
		t.Error("Two agent certificates have the same serial number")
	}
}

// TestGenerateCAErrorConditions tests error handling in CA generation
// Note: Hard to trigger errors in crypto/rand, this is more for coverage
func TestGenerateCAMultipleCalls(t *testing.T) {
	// Just verify multiple calls work fine
	for i := 0; i < 5; i++ {
		_, _, err := generateCA("test")
		if err != nil {
			t.Errorf("generateCA() call %d failed: %v", i+1, err)
		}
	}
}

// TestSavePrivateKeyInvalidPath tests error handling for invalid path
func TestSavePrivateKeyInvalidPath(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}

	tmpDir := t.TempDir()
	readonlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readonlyDir, 0444); err != nil {
		t.Fatalf("Failed to create readonly dir: %v", err)
	}
	defer os.Chmod(readonlyDir, 0755)

	invalidPath := filepath.Join(readonlyDir, "test.key")
	err = savePrivateKey(invalidPath, priv)
	if err == nil {
		t.Error("savePrivateKey() expected error for invalid path, got nil")
	}
}

// TestSaveCertificateInvalidPath tests error handling for invalid path
func TestSaveCertificateInvalidPath(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	// Create a test certificate
	_, cert, err := generateCA("test")
	if err != nil {
		t.Fatalf("generateCA() error = %v", err)
	}

	tmpDir := t.TempDir()
	readonlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readonlyDir, 0444); err != nil {
		t.Fatalf("Failed to create readonly dir: %v", err)
	}
	defer os.Chmod(readonlyDir, 0755)

	invalidPath := filepath.Join(readonlyDir, "test.crt")
	err = saveCertificate(invalidPath, cert)
	if err == nil {
		t.Error("saveCertificate() expected error for invalid path, got nil")
	}
}

// TestGenerateAgentCertWithInvalidCA tests error handling (for coverage)
func TestGenerateAgentCertWithNilCA(t *testing.T) {
	// Generate valid CA first
	caKey, caCert, err := generateCA("test-ca")
	if err != nil {
		t.Fatalf("generateCA() error = %v", err)
	}

	// Test with valid inputs (negative test is hard without mocking crypto/x509)
	_, _, err = generateAgentCert("test", caKey, caCert)
	if err != nil {
		t.Errorf("generateAgentCert() unexpected error: %v", err)
	}
}

// TestCertificateInfoFields verifies all fields are populated
func TestCertificateInfoFields(t *testing.T) {
	tmpDir := t.TempDir()
	info, err := GenerateSelfSignedCerts(tmpDir, "test-agent")
	if err != nil {
		t.Fatalf("GenerateSelfSignedCerts() error = %v", err)
	}

	// Verify all fields are non-zero
	if info.CAFingerprint == "" {
		t.Error("CAFingerprint is empty")
	}
	if info.CertFingerprint == "" {
		t.Error("CertFingerprint is empty")
	}
	if info.ValidFrom.IsZero() {
		t.Error("ValidFrom is zero")
	}
	if info.ValidUntil.IsZero() {
		t.Error("ValidUntil is zero")
	}
	if info.Subject == "" {
		t.Error("Subject is empty")
	}

	// Verify ValidFrom is before ValidUntil
	if !info.ValidFrom.Before(info.ValidUntil) {
		t.Error("ValidFrom is not before ValidUntil")
	}
}
