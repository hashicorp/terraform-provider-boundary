// Copyright IBM Corp. 2020, 2026
// SPDX-License-Identifier: MPL-2.0
//
// This is a provider-local subset of the Vault testing helpers in
// github.com/hashicorp/boundary/internal/credential/vault

package vault

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
)

const (
	defaultVaultVersion = "1.15.1"
	vaultRepository     = "hashicorp/vault"
)

// TestVaultServer is a Vault server running in a Docker container for tests.
type TestVaultServer struct {
	RootToken string
	Addr      string

	CaCert        []byte
	ServerCert    []byte
	ClientCert    []byte
	ClientKey     []byte
	TlsServerName string
	TlsSkipVerify bool

	pool       *dockertest.Pool
	resource   *dockertest.Resource
	httpClient *http.Client
	stopped    atomic.Bool
}

// TestVaultTLS represents the TLS configuration level of a test Vault server.
type TestVaultTLS int

const (
	// TestNoTLS disables TLS.
	TestNoTLS TestVaultTLS = iota
	// TestServerTLS configures server-side TLS.
	TestServerTLS
	// TestClientTLS configures TLS with a required client certificate.
	TestClientTLS
)

// TestOption configures a test Vault server.
type TestOption func(*testOptions)

type testOptions struct {
	vaultTLS TestVaultTLS
}

// WithTestVaultTLS configures the Vault server to use the specified TLS mode.
func WithTestVaultTLS(mode TestVaultTLS) TestOption {
	return func(opts *testOptions) {
		opts.vaultTLS = mode
	}
}

// NewTestVaultServer starts a Docker-backed Vault server and registers cleanup
// with t.
func NewTestVaultServer(t testing.TB, options ...TestOption) *TestVaultServer {
	t.Helper()

	opts := testOptions{vaultTLS: TestNoTLS}
	for _, option := range options {
		option(&opts)
	}
	if opts.vaultTLS < TestNoTLS || opts.vaultTLS > TestClientTLS {
		t.Fatalf("unsupported Vault TLS mode %d", opts.vaultTLS)
	}

	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	server := &TestVaultServer{
		RootToken: fmt.Sprintf("icu-root-%s", t.Name()),
		pool:      pool,
	}

	const serverTLSConfig = `{
  "listener": [{
    "tcp": {
      "address": "0.0.0.0:8200",
      "tls_disable": "false",
      "tls_cert_file": "/vault/config/certificates/certificate.pem",
      "tls_key_file": "/vault/config/certificates/key.pem"
    }
  }]
}`
	const clientTLSConfig = `{
  "listener": [{
    "tcp": {
      "address": "0.0.0.0:8200",
      "tls_disable": "false",
      "tls_cert_file": "/vault/config/certificates/certificate.pem",
      "tls_key_file": "/vault/config/certificates/key.pem",
      "tls_require_and_verify_client_cert": "true",
      "tls_client_ca_file": "/vault/config/certificates/client-ca-certificate.pem"
    }
  }]
}`

	runOptions := &dockertest.RunOptions{
		Repository: dockerVaultRepository(),
		Tag:        defaultVaultVersion,
		Env:        []string{fmt.Sprintf("VAULT_DEV_ROOT_TOKEN_ID=%s", server.RootToken)},
	}

	vaultConfig := vaultapi.DefaultConfig()
	if opts.vaultTLS != TestNoTLS {
		runOptions.Env = append(runOptions.Env, "VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8300")
		if opts.vaultTLS == TestClientTLS {
			runOptions.Env = append(runOptions.Env, fmt.Sprintf("VAULT_LOCAL_CONFIG=%s", clientTLSConfig))
		} else {
			runOptions.Env = append(runOptions.Env, fmt.Sprintf("VAULT_LOCAL_CONFIG=%s", serverTLSConfig))
		}
		server.configureTLS(t, opts.vaultTLS, runOptions, vaultConfig)
	}

	resource, err := pool.RunWithOptions(runOptions)
	require.NoError(t, err)
	server.resource = resource
	t.Cleanup(func() {
		if !server.stopped.Load() {
			server.cleanup(t)
		}
	})
	if opts.vaultTLS == TestNoTLS {
		server.Addr = fmt.Sprintf("http://localhost:%s", resource.GetPort("8200/tcp"))
	} else {
		server.Addr = fmt.Sprintf("https://localhost:%s", resource.GetPort("8200/tcp"))
	}

	vaultConfig.Address = server.Addr
	server.httpClient = vaultConfig.HttpClient
	client := server.newClient(t)
	client.SetToken(server.RootToken)
	require.NoError(t, pool.Retry(func() error {
		_, err := client.Sys().Health()
		return err
	}))

	require.NoError(t, client.Sys().PutPolicy("boundary-controller", `path "auth/token/lookup-self" {
  capabilities = ["read"]
}
path "auth/token/renew-self" {
  capabilities = ["update"]
}
path "auth/token/revoke-self" {
  capabilities = ["update"]
}
path "sys/leases/renew" {
  capabilities = ["update"]
}
path "sys/leases/revoke" {
  capabilities = ["update"]
}`))

	return server
}

func dockerVaultRepository() string {
	if mirror := os.Getenv("DOCKER_MIRROR"); mirror != "" {
		return fmt.Sprintf("%s/%s", mirror, vaultRepository)
	}
	return vaultRepository
}

func (v *TestVaultServer) newClient(t testing.TB) *vaultapi.Client {
	t.Helper()
	config := vaultapi.DefaultConfig()
	config.Address = v.Addr
	config.HttpClient = v.httpClient
	client, err := vaultapi.NewClient(config)
	require.NoError(t, err)
	return client
}

func (v *TestVaultServer) configureTLS(
	t testing.TB,
	mode TestVaultTLS,
	runOptions *dockertest.RunOptions,
	config *vaultapi.Config,
) {
	t.Helper()

	serverCA := makeTestCertificate(t, nil, "", true, false)
	serverCert := makeTestCertificate(t, serverCA, "localhost", false, true)
	v.ServerCert = serverCert.cert
	v.CaCert = serverCA.cert

	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o777))
	for name, contents := range map[string][]byte{
		"ca-certificate.pem": v.CaCert,
		"certificate.pem":    serverCert.cert,
		"key.pem":            serverCert.key,
	} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), contents, 0o777))
	}

	tlsConfig := &tls.Config{RootCAs: certificatePool(t, v.CaCert)}
	if mode == TestClientTLS {
		clientCA := makeTestCertificate(t, nil, "", true, false)
		clientCert := makeTestCertificate(t, clientCA, "", false, false)
		v.ClientCert = clientCert.cert
		v.ClientKey = clientCert.key
		require.NoError(t, os.WriteFile(filepath.Join(dir, "client-ca-certificate.pem"), clientCA.cert, 0o777))
		pair, err := tls.X509KeyPair(v.ClientCert, v.ClientKey)
		require.NoError(t, err)
		tlsConfig.Certificates = []tls.Certificate{pair}
	}

	transport, ok := config.HttpClient.Transport.(*http.Transport)
	require.True(t, ok, "Vault client transport is not an HTTP transport")
	transport.TLSClientConfig = tlsConfig
	runOptions.Mounts = append(runOptions.Mounts, fmt.Sprintf("%s:/vault/config/certificates", dir))
}

func (v *TestVaultServer) cleanup(t testing.TB) {
	t.Helper()
	var err error
	for attempt := 0; attempt < 10; attempt++ {
		err = v.pool.Purge(v.resource)
		if err == nil {
			v.stopped.Store(true)
			return
		}
		time.Sleep(time.Second)
	}
	t.Errorf("failed to clean up Vault test container: %v", err)
}

// CreateToken creates a periodic, orphan token with the policies used by
// Boundary's Vault integration tests.
func (v *TestVaultServer) CreateToken(t testing.TB) (*vaultapi.Secret, string) {
	t.Helper()

	period := 24 * time.Hour
	if deadline, ok := testDeadline(t); ok {
		if remaining := time.Until(deadline); remaining > 0 {
			period = remaining
		}
	}
	renewable := true
	client := v.newClient(t)
	client.SetToken(v.RootToken)

	secret, err := client.Auth().Token().Create(&vaultapi.TokenCreateRequest{
		DisplayName: t.Name(),
		NoParent:    true,
		Renewable:   &renewable,
		Period:      period.String(),
		Policies:    []string{"default", "boundary-controller"},
	})
	require.NoError(t, err)
	require.NotNil(t, secret)
	token, err := secret.TokenID()
	require.NoError(t, err)
	return secret, token
}
