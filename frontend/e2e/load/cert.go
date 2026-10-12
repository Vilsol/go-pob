//go:build ignore

// Writes a throwaway self-signed certificate for data.pob.gg, which the load probe's local data host
// presents (Chromium runs with --ignore-certificate-errors). Usage: go run cert.go <dir>
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

func main() {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "data.pob.gg"},
		DNSNames:     []string{"data.pob.gg"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		panic(err)
	}
	dir := os.Args[1]
	must(os.WriteFile(filepath.Join(dir, "cert.pem"), pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600))
	must(os.WriteFile(filepath.Join(dir, "key.pem"), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
