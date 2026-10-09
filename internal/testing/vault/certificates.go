// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0
//
// Certificate-generation subset of Vault test helpers in
// github.com/hashicorp/boundary/internal/credential/vault

package vault

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testCertificate struct {
	cert       []byte
	key        []byte
	parsedCert *x509.Certificate
	privateKey *ecdsa.PrivateKey
}

func makeTestCertificate(t testing.TB, ca *testCertificate, host string, isCA, isServer bool) *testCertificate {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)

	notBefore, notAfter := time.Now().Add(-time.Minute), time.Now().Add(24*time.Hour)
	if deadline, ok := testDeadline(t); ok && deadline.Before(notAfter) {
		notAfter = deadline
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{Organization: []string{"Acme Test Certificates"}},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		IsCA:                  isCA,
		BasicConstraintsValid: true,
	}
	if isCA {
		template.KeyUsage |= x509.KeyUsageCertSign
	} else if isServer {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		if ip := net.ParseIP(host); ip != nil {
			template.IPAddresses = []net.IP{ip}
		} else {
			template.DNSNames = []string{host}
		}
	} else {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}

	issuer := &template
	issuerKey := key
	if ca != nil {
		issuer = ca.parsedCert
		issuerKey = ca.privateKey
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, issuer, &key.PublicKey, issuerKey)
	require.NoError(t, err)
	privateKey, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKey})

	parsedCert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return &testCertificate{
		cert:       certPEM,
		key:        keyPEM,
		parsedCert: parsedCert,
		privateKey: key,
	}
}

func testDeadline(t testing.TB) (time.Time, bool) {
	t.Helper()
	if deadlineProvider, ok := t.(interface {
		Deadline() (time.Time, bool)
	}); ok {
		return deadlineProvider.Deadline()
	}
	return time.Time{}, false
}

func certificatePool(t testing.TB, ca []byte) *x509.CertPool {
	t.Helper()
	pool := x509.NewCertPool()
	require.True(t, pool.AppendCertsFromPEM(ca), "failed to add Vault CA certificate")
	return pool
}
