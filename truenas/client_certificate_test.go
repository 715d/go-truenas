package truenas

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCertificateClient(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	client := server.CreateTestClient(t)
	defer client.Close()

	certClient := NewCertificateClient(client)
	require.NotNil(t, certClient)
	assert.Equal(t, client, certClient.client)
}

func TestCertificateClient_List(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockCertificates := []Certificate{
		{
			ID:                 1,
			Name:               "test-cert-1",
			Type:               1,
			Certificate:        "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
			Privatekey:         "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----",
			CA:                 false,
			Country:            "US",
			State:              "CA",
			City:               "San Francisco",
			Organization:       "Test Org",
			OrganizationalUnit: "IT",
			Common:             "test.example.com",
			SAN:                []string{"test.example.com", "www.test.example.com"},
			Email:              "admin@test.example.com",
			KeyLength:          2048,
			KeyType:            "RSA",
			Digest:             "SHA256",
			CertType:           "CERTIFICATE",
			CertTypeFull:       "Certificate",
			Internal:           "NO",
			Parsed:             true,
		},
		{
			ID:       2,
			Name:     "ca-cert",
			Type:     2,
			CA:       true,
			Common:   "Test CA",
			KeyType:  "RSA",
			Internal: "YES",
			Parsed:   true,
		},
	}
	server.SetResponse("certificate.query", mockCertificates)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	certificates, err := client.Certificate.List(ctx)
	require.NoError(t, err)
	assert.Len(t, certificates, 2)
	assert.Equal(t, "test-cert-1", certificates[0].Name)
	assert.Equal(t, "ca-cert", certificates[1].Name)
	assert.False(t, certificates[0].CA)
	assert.True(t, certificates[1].CA)
}

func TestCertificateClient_List_Empty(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("certificate.query", []Certificate{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	certificates, err := client.Certificate.List(ctx)
	require.NoError(t, err)
	assert.Len(t, certificates, 0)
}

func TestCertificateClient_Get(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockCertificate := Certificate{
		ID:          1,
		Name:        "test-cert",
		Type:        1,
		Certificate: "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
		Privatekey:  "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----",
		CA:          false,
		Common:      "test.example.com",
		KeyLength:   2048,
		KeyType:     "RSA",
		Digest:      "SHA256",
		Serial:      12345,
		Fingerprint: "AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD",
		NotBefore:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Subject:     map[string]any{"CN": "test.example.com"},
		Issuer:      map[string]any{"CN": "test.example.com"},
		Extensions:  map[string]any{},
		Internal:    "NO",
		Parsed:      true,
		Lifetime:    365,
	}
	server.SetResponse("certificate.query", []Certificate{mockCertificate})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	certificate, err := client.Certificate.Get(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, certificate)
	assert.Equal(t, 1, certificate.ID)
	assert.Equal(t, "test-cert", certificate.Name)
	assert.Equal(t, "test.example.com", certificate.Common)
	assert.Equal(t, 2048, certificate.KeyLength)
	assert.Equal(t, "RSA", certificate.KeyType)
	assert.Equal(t, 12345, certificate.Serial)
}

func TestCertificateClient_Get_NotFound(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetResponse("certificate.query", []Certificate{})

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	certificate, err := client.Certificate.Get(ctx, 999)
	assert.Error(t, err)
	assert.Nil(t, certificate)
	var notFoundErr *NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func TestCertificateClient_Create_Internal(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockCertificate := Certificate{
		ID:           1,
		Name:         "internal-cert",
		Type:         1,
		CA:           false,
		Common:       "internal.example.com",
		KeyLength:    2048,
		KeyType:      "RSA",
		Digest:       "SHA256",
		Country:      "US",
		State:        "CA",
		City:         "San Francisco",
		Organization: "Test Organization",
		Email:        "admin@example.com",
		Internal:     "YES",
		Parsed:       true,
		Lifetime:     365,
	}
	server.SetJobResponse("certificate.create", mockCertificate)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &CertificateCreateRequest{
		Name:               "internal-cert",
		CreateType:         string(CertificateCreateInternal),
		KeyLength:          2048,
		KeyType:            string(CertificateKeyTypeRSA),
		DigestAlgorithm:    string(CertificateDigestSHA256),
		Lifetime:           365,
		Country:            "US",
		State:              "CA",
		City:               "San Francisco",
		Organization:       "Test Organization",
		OrganizationalUnit: "IT Department",
		Email:              "admin@example.com",
		Common:             "internal.example.com",
		SAN:                []string{"internal.example.com", "www.internal.example.com"},
		CertExtensions: &CertificateExtensions{
			KeyUsage: &KeyUsage{
				Enabled:           true,
				DigitalSignature:  true,
				KeyEncipherment:   true,
				ExtensionCritical: false,
			},
			ExtendedKeyUsage: &ExtendedKeyUsage{
				Enabled:           true,
				Usages:            []string{"SERVER_AUTH", "CLIENT_AUTH"},
				ExtensionCritical: false,
			},
		},
	}

	ctx := NewTestContext(t)
	certificate, err := client.Certificate.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, certificate)
	assert.Equal(t, "internal-cert", certificate.Name)
	assert.Equal(t, "internal.example.com", certificate.Common)
	assert.Equal(t, 365, certificate.Lifetime)
}

func TestCertificateClient_Create_Imported(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockCertificate := Certificate{
		ID:          1,
		Name:        "imported-cert",
		Type:        1,
		Certificate: "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
		Privatekey:  "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----",
		CA:          false,
		Common:      "imported.example.com",
		Internal:    "NO",
		Parsed:      true,
	}
	server.SetJobResponse("certificate.create", mockCertificate)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &CertificateCreateRequest{
		Name:        "imported-cert",
		CreateType:  string(CertificateCreateImported),
		Certificate: "-----BEGIN CERTIFICATE-----\nMIIC...\n-----END CERTIFICATE-----",
		Privatekey:  "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----",
		Passphrase:  "secret",
	}

	ctx := NewTestContext(t)
	certificate, err := client.Certificate.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, certificate)
	assert.Equal(t, "imported-cert", certificate.Name)
}

func TestCertificateClient_Create_CSR(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockCertificate := Certificate{
		ID:       1,
		Name:     "csr-cert",
		Type:     1,
		CSR:      "-----BEGIN CERTIFICATE REQUEST-----\nMIIC...\n-----END CERTIFICATE REQUEST-----",
		CA:       false,
		Common:   "csr.example.com",
		Internal: "NO",
		Parsed:   true,
	}
	server.SetJobResponse("certificate.create", mockCertificate)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &CertificateCreateRequest{
		Name:               "csr-cert",
		CreateType:         string(CertificateCreateCSR),
		KeyLength:          2048,
		KeyType:            string(CertificateKeyTypeRSA),
		DigestAlgorithm:    string(CertificateDigestSHA256),
		Country:            "US",
		State:              "CA",
		City:               "San Francisco",
		Organization:       "Test Organization",
		OrganizationalUnit: "IT Department",
		Email:              "admin@example.com",
		Common:             "csr.example.com",
		SAN:                []string{"csr.example.com"},
	}

	ctx := NewTestContext(t)
	certificate, err := client.Certificate.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, certificate)
	assert.Equal(t, "csr-cert", certificate.Name)
}

func TestCertificateClient_Create_ACME(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockCertificate := Certificate{
		ID:       1,
		Name:     "acme-cert",
		Type:     1,
		CA:       false,
		Common:   "acme.example.com",
		Internal: "NO",
		Parsed:   true,
	}
	server.SetJobResponse("certificate.create", mockCertificate)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &CertificateCreateRequest{
		Name:             "acme-cert",
		CreateType:       string(CertificateCreateACME),
		TOS:              true,
		CSRID:            1,
		AcmeDirectoryURI: "https://acme-v02.api.letsencrypt.org/directory",
		DNSMapping: map[string]string{
			"acme.example.com": "dns_cloudflare",
		},
		RenewDays: 30,
	}

	ctx := NewTestContext(t)
	certificate, err := client.Certificate.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, certificate)
	assert.Equal(t, "acme-cert", certificate.Name)
}

func TestCertificateClient_Create_WithECKey(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockCertificate := Certificate{
		ID:       1,
		Name:     "ec-cert",
		Type:     1,
		CA:       false,
		Common:   "ec.example.com",
		KeyType:  "EC",
		Internal: "YES",
		Parsed:   true,
	}
	server.SetJobResponse("certificate.create", mockCertificate)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &CertificateCreateRequest{
		Name:            "ec-cert",
		CreateType:      string(CertificateCreateInternal),
		KeyType:         string(CertificateKeyTypeEC),
		ECCurve:         string(CertificateECCurveSECP384R1),
		DigestAlgorithm: string(CertificateDigestSHA384),
		Lifetime:        365,
		Common:          "ec.example.com",
	}

	ctx := NewTestContext(t)
	certificate, err := client.Certificate.Create(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, certificate)
	assert.Equal(t, "ec-cert", certificate.Name)
	assert.Equal(t, "EC", certificate.KeyType)
}

func TestCertificateClient_Update(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockCertificate := Certificate{
		ID:       1,
		Name:     "updated-cert",
		Type:     1,
		Revoked:  true,
		CA:       false,
		Common:   "test.example.com",
		Internal: "NO",
		Parsed:   true,
	}
	server.SetJobResponse("certificate.update", mockCertificate)

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &CertificateUpdateRequest{
		Name:    "updated-cert",
		Revoked: true,
	}

	ctx := NewTestContext(t)
	certificate, err := client.Certificate.Update(ctx, 1, req)
	require.NoError(t, err)
	require.NotNil(t, certificate)
	assert.Equal(t, "updated-cert", certificate.Name)
	assert.True(t, certificate.Revoked)
}

func TestCertificateClient_Delete(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("certificate.delete", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Certificate.Delete(ctx, 1, false)
	assert.NoError(t, err)
}

func TestCertificateClient_Delete_Force(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobResponse("certificate.delete", nil)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Certificate.Delete(ctx, 1, true)
	assert.NoError(t, err)
}

func TestCertificateClient_GetCountryChoices(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockChoices := map[string]string{
		"US": "United States",
		"CA": "Canada",
		"GB": "United Kingdom",
		"DE": "Germany",
		"FR": "France",
	}
	server.SetResponse("certificate.country_choices", mockChoices)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	choices, err := client.Certificate.GetCountryChoices(ctx)
	require.NoError(t, err)
	assert.Len(t, choices, 5)
	assert.Equal(t, "United States", choices["US"])
	assert.Equal(t, "Canada", choices["CA"])
}

func TestCertificateClient_GetKeyTypeChoices(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockChoices := map[string]string{
		"RSA": "RSA",
		"EC":  "Elliptic Curve",
	}
	server.SetResponse("certificate.key_type_choices", mockChoices)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	choices, err := client.Certificate.GetKeyTypeChoices(ctx)
	require.NoError(t, err)
	assert.Len(t, choices, 2)
	assert.Equal(t, "RSA", choices["RSA"])
	assert.Equal(t, "Elliptic Curve", choices["EC"])
}

func TestCertificateClient_GetECCurveChoices(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockChoices := map[string]string{
		"BrainpoolP256R1": "BrainpoolP256R1",
		"BrainpoolP384R1": "BrainpoolP384R1",
		"BrainpoolP512R1": "BrainpoolP512R1",
		"SECP256K1":       "SECP256K1",
		"SECP384R1":       "SECP384R1",
		"SECP521R1":       "SECP521R1",
		"ed25519":         "Ed25519",
	}
	server.SetResponse("certificate.ec_curve_choices", mockChoices)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	choices, err := client.Certificate.GetECCurveChoices(ctx)
	require.NoError(t, err)
	assert.Len(t, choices, 7)
	assert.Equal(t, "BrainpoolP256R1", choices["BrainpoolP256R1"])
	assert.Equal(t, "Ed25519", choices["ed25519"])
}

func TestCertificateClient_GetExtendedKeyUsageChoices(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockChoices := map[string]string{
		"SERVER_AUTH":      "TLS Web Server Authentication",
		"CLIENT_AUTH":      "TLS Web Client Authentication",
		"CODE_SIGNING":     "Code Signing",
		"EMAIL_PROTECTION": "E-mail Protection",
		"TIME_STAMPING":    "Time Stamping",
		"OCSP_SIGNING":     "OCSP Signing",
	}
	server.SetResponse("certificate.extended_key_usage_choices", mockChoices)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	choices, err := client.Certificate.GetExtendedKeyUsageChoices(ctx)
	require.NoError(t, err)
	assert.Len(t, choices, 6)
	assert.Equal(t, "TLS Web Server Authentication", choices["SERVER_AUTH"])
	assert.Equal(t, "Code Signing", choices["CODE_SIGNING"])
}

func TestCertificateClient_GetProfiles(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockProfiles := map[string]any{
		"https": map[string]any{
			"name":        "HTTPS Certificate",
			"description": "Certificate for HTTPS web servers",
			"key_usage": map[string]any{
				"digital_signature": true,
				"key_encipherment":  true,
			},
			"extended_key_usage": []string{"SERVER_AUTH"},
		},
		"ca": map[string]any{
			"name":        "Certificate Authority",
			"description": "Certificate Authority for signing other certificates",
			"key_usage": map[string]any{
				"key_cert_sign": true,
				"crl_sign":      true,
			},
			"basic_constraints": map[string]any{
				"ca": true,
			},
		},
	}
	server.SetResponse("certificate.profiles", mockProfiles)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	profiles, err := client.Certificate.GetProfiles(ctx)
	require.NoError(t, err)
	assert.Len(t, profiles, 2)
	assert.Contains(t, profiles, "https")
	assert.Contains(t, profiles, "ca")
}

func TestCertificateClient_GetACMEServerChoices(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	mockChoices := map[string]string{
		"https://acme-v02.api.letsencrypt.org/directory":         "Let's Encrypt (Production)",
		"https://acme-staging-v02.api.letsencrypt.org/directory": "Let's Encrypt (Staging)",
		"https://acme.zerossl.com/v2/DV90":                       "ZeroSSL (Production)",
	}
	server.SetResponse("certificate.acme_server_choices", mockChoices)

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	choices, err := client.Certificate.GetACMEServerChoices(ctx)
	require.NoError(t, err)
	assert.Len(t, choices, 3)
	assert.Equal(t, "Let's Encrypt (Production)", choices["https://acme-v02.api.letsencrypt.org/directory"])
	assert.Equal(t, "ZeroSSL (Production)", choices["https://acme.zerossl.com/v2/DV90"])
}

// Error handling tests
func TestCertificateClient_List_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("certificate.query", 500, "Certificate service unavailable")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Certificate.List(ctx)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.Data.Error)
	assert.Equal(t, "Certificate service unavailable", apiErr.Message)
}

func TestCertificateClient_Get_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("certificate.query", 404, "Certificate not found")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Certificate.Get(ctx, 999)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 404, apiErr.Data.Error)
	assert.Equal(t, "Certificate not found", apiErr.Message)
}

func TestCertificateClient_Create_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobError("certificate.create", "Invalid certificate parameters")

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &CertificateCreateRequest{
		Name:       "invalid-cert",
		CreateType: "INVALID_TYPE",
	}

	ctx := NewTestContext(t)
	_, err := client.Certificate.Create(ctx, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Invalid certificate parameters")
}

func TestCertificateClient_Update_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobError("certificate.update", "Certificate is in use and cannot be modified")

	client := server.CreateTestClient(t)
	defer client.Close()

	req := &CertificateUpdateRequest{
		Name: "cert-in-use",
	}

	ctx := NewTestContext(t)
	_, err := client.Certificate.Update(ctx, 1, req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Certificate is in use and cannot be modified")
}

func TestCertificateClient_Delete_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetJobError("certificate.delete", "Certificate is in use and cannot be deleted")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	err := client.Certificate.Delete(ctx, 1, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Certificate is in use and cannot be deleted")
}

func TestCertificateClient_GetChoices_Error(t *testing.T) {
	t.Parallel()
	server := NewTestServer(t)
	defer server.Close()

	server.SetError("certificate.country_choices", 503, "Service temporarily unavailable")

	client := server.CreateTestClient(t)
	defer client.Close()

	ctx := NewTestContext(t)
	_, err := client.Certificate.GetCountryChoices(ctx)
	require.Error(t, err)

	var apiErr *RPCError
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 503, apiErr.Data.Error)
	assert.Equal(t, "Service temporarily unavailable", apiErr.Message)
}

// Test certificate extensions
func TestCertificateExtensions_BasicConstraints(t *testing.T) {
	t.Parallel()
	pathLength := 5
	bc := &BasicConstraints{
		CA:                true,
		Enabled:           true,
		PathLength:        &pathLength,
		ExtensionCritical: true,
	}

	assert.True(t, bc.CA)
	assert.True(t, bc.Enabled)
	assert.NotNil(t, bc.PathLength)
	assert.Equal(t, 5, *bc.PathLength)
	assert.True(t, bc.ExtensionCritical)
}

func TestCertificateExtensions_KeyUsage(t *testing.T) {
	t.Parallel()
	ku := &KeyUsage{
		Enabled:          true,
		DigitalSignature: true,
		KeyEncipherment:  true,
		KeyCertSign:      true,
		CRLSign:          true,
	}

	assert.True(t, ku.Enabled)
	assert.True(t, ku.DigitalSignature)
	assert.False(t, ku.ContentCommitment)
	assert.True(t, ku.KeyEncipherment)
	assert.True(t, ku.KeyCertSign)
	assert.True(t, ku.CRLSign)
	assert.False(t, ku.ExtensionCritical)
}

func TestCertificateExtensions_ExtendedKeyUsage(t *testing.T) {
	t.Parallel()
	eku := &ExtendedKeyUsage{
		Usages:            []string{"SERVER_AUTH", "CLIENT_AUTH", "CODE_SIGNING"},
		Enabled:           true,
		ExtensionCritical: false,
	}

	assert.True(t, eku.Enabled)
	assert.Len(t, eku.Usages, 3)
	assert.Contains(t, eku.Usages, "SERVER_AUTH")
	assert.Contains(t, eku.Usages, "CLIENT_AUTH")
	assert.Contains(t, eku.Usages, "CODE_SIGNING")
	assert.False(t, eku.ExtensionCritical)
}

func TestCertificateExtensions_AuthorityKeyIdentifier(t *testing.T) {
	t.Parallel()
	aki := &AuthorityKeyIdentifier{
		AuthorityCertIssuer: true,
		Enabled:             true,
		ExtensionCritical:   false,
	}

	assert.True(t, aki.AuthorityCertIssuer)
	assert.True(t, aki.Enabled)
	assert.False(t, aki.ExtensionCritical)
}

// Test constants
func TestCertificateConstants(t *testing.T) {
	t.Parallel()
	// Test create types
	assert.Equal(t, CertificateCreateType("CERTIFICATE_CREATE_INTERNAL"), CertificateCreateInternal)
	assert.Equal(t, CertificateCreateType("CERTIFICATE_CREATE_IMPORTED"), CertificateCreateImported)
	assert.Equal(t, CertificateCreateType("CERTIFICATE_CREATE_CSR"), CertificateCreateCSR)
	assert.Equal(t, CertificateCreateType("CERTIFICATE_CREATE_IMPORTED_CSR"), CertificateCreateImportedCSR)
	assert.Equal(t, CertificateCreateType("CERTIFICATE_CREATE_ACME"), CertificateCreateACME)

	// Test key types
	assert.Equal(t, CertificateKeyType("RSA"), CertificateKeyTypeRSA)
	assert.Equal(t, CertificateKeyType("EC"), CertificateKeyTypeEC)

	// Test EC curves
	assert.Equal(t, CertificateECCurve("BrainpoolP512R1"), CertificateECCurveBrainpoolP512R1)
	assert.Equal(t, CertificateECCurve("SECP256K1"), CertificateECCurveSECP256K1)
	assert.Equal(t, CertificateECCurve("ed25519"), CertificateECCurveEd25519)

	// Test digest algorithms
	assert.Equal(t, CertificateDigestAlgorithm("SHA1"), CertificateDigestSHA1)
	assert.Equal(t, CertificateDigestAlgorithm("SHA256"), CertificateDigestSHA256)
	assert.Equal(t, CertificateDigestAlgorithm("SHA512"), CertificateDigestSHA512)
}

// Table-driven tests for different create types
func TestCertificateClient_Create_AllTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		createType   CertificateCreateType
		setupRequest func() *CertificateCreateRequest
		expectName   string
	}{
		{
			name:       "Internal Certificate",
			createType: CertificateCreateInternal,
			setupRequest: func() *CertificateCreateRequest {
				return &CertificateCreateRequest{
					Name:            "internal-test",
					CreateType:      string(CertificateCreateInternal),
					KeyLength:       2048,
					KeyType:         string(CertificateKeyTypeRSA),
					DigestAlgorithm: string(CertificateDigestSHA256),
					Lifetime:        365,
					Common:          "internal.test.com",
				}
			},
			expectName: "internal-test",
		},
		{
			name:       "Imported Certificate",
			createType: CertificateCreateImported,
			setupRequest: func() *CertificateCreateRequest {
				return &CertificateCreateRequest{
					Name:        "imported-test",
					CreateType:  string(CertificateCreateImported),
					Certificate: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
					Privatekey:  "-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----",
				}
			},
			expectName: "imported-test",
		},
		{
			name:       "CSR Certificate",
			createType: CertificateCreateCSR,
			setupRequest: func() *CertificateCreateRequest {
				return &CertificateCreateRequest{
					Name:            "csr-test",
					CreateType:      string(CertificateCreateCSR),
					KeyLength:       2048,
					KeyType:         string(CertificateKeyTypeRSA),
					DigestAlgorithm: string(CertificateDigestSHA256),
					Common:          "csr.test.com",
				}
			},
			expectName: "csr-test",
		},
		{
			name:       "ACME Certificate",
			createType: CertificateCreateACME,
			setupRequest: func() *CertificateCreateRequest {
				return &CertificateCreateRequest{
					Name:             "acme-test",
					CreateType:       string(CertificateCreateACME),
					TOS:              true,
					CSRID:            1,
					AcmeDirectoryURI: "https://acme-v02.api.letsencrypt.org/directory",
				}
			},
			expectName: "acme-test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTestServer(t)
			defer server.Close()

			mockCert := Certificate{
				ID:       1,
				Name:     tt.expectName,
				Type:     1,
				Internal: "YES",
				Parsed:   true,
			}
			server.SetJobResponse("certificate.create", mockCert)

			client := server.CreateTestClient(t)
			defer client.Close()

			req := tt.setupRequest()
			ctx := NewTestContext(t)
			cert, err := client.Certificate.Create(ctx, req)
			require.NoError(t, err)
			require.NotNil(t, cert)
			assert.Equal(t, tt.expectName, cert.Name)
		})
	}
}

// Table-driven tests for different key types and curves
func TestCertificateClient_Create_KeyTypes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		keyType   CertificateKeyType
		ecCurve   CertificateECCurve
		keyLength int
		expectKey string
	}{
		{
			name:      "RSA 2048",
			keyType:   CertificateKeyTypeRSA,
			keyLength: 2048,
			expectKey: "RSA",
		},
		{
			name:      "RSA 4096",
			keyType:   CertificateKeyTypeRSA,
			keyLength: 4096,
			expectKey: "RSA",
		},
		{
			name:      "EC P-256",
			keyType:   CertificateKeyTypeEC,
			ecCurve:   CertificateECCurveSECP256K1,
			expectKey: "EC",
		},
		{
			name:      "EC P-384",
			keyType:   CertificateKeyTypeEC,
			ecCurve:   CertificateECCurveSECP384R1,
			expectKey: "EC",
		},
		{
			name:      "EC Ed25519",
			keyType:   CertificateKeyTypeEC,
			ecCurve:   CertificateECCurveEd25519,
			expectKey: "EC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTestServer(t)
			defer server.Close()

			mockCert := Certificate{
				ID:        1,
				Name:      "key-test",
				Type:      1,
				KeyType:   tt.expectKey,
				KeyLength: tt.keyLength,
				Internal:  "YES",
				Parsed:    true,
			}
			server.SetJobResponse("certificate.create", mockCert)

			client := server.CreateTestClient(t)
			defer client.Close()

			req := &CertificateCreateRequest{
				Name:            "key-test",
				CreateType:      string(CertificateCreateInternal),
				KeyType:         string(tt.keyType),
				KeyLength:       tt.keyLength,
				DigestAlgorithm: string(CertificateDigestSHA256),
				Common:          "key.test.com",
			}

			if tt.keyType == CertificateKeyTypeEC {
				req.ECCurve = string(tt.ecCurve)
			}

			ctx := NewTestContext(t)
			cert, err := client.Certificate.Create(ctx, req)
			require.NoError(t, err)
			require.NotNil(t, cert)
			assert.Equal(t, tt.expectKey, cert.KeyType)
		})
	}
}
