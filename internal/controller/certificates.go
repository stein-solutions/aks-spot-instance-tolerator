package controller

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

func createCA(hosts []string, notBefore, notAfter time.Time) (caTemplate *x509.Certificate, caKey *ecdsa.PrivateKey, err error) {
	// Generiere einen privaten Schlüssel
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	// Erstelle ein selbstsigniertes Zertifikat
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}

	rootTemplate := x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			rootTemplate.IPAddresses = append(rootTemplate.IPAddresses, ip)
		} else {
			rootTemplate.DNSNames = append(rootTemplate.DNSNames, h)
		}
	}

	return &rootTemplate, rootKey, nil
}

func createCert(hosts []string, notBefore, notAfter time.Time) (certTemplate *x509.Certificate, certKey *ecdsa.PrivateKey, err error) {

	certKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}

	certTemplate = &x509.Certificate{
		SerialNumber:          serialNumber,
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			certTemplate.IPAddresses = append(certTemplate.IPAddresses, ip)
		} else {
			certTemplate.DNSNames = append(certTemplate.DNSNames, h)
		}
	}

	return certTemplate, certKey, nil
}

func pemBlockForKey(priv *ecdsa.PrivateKey) []byte {
	b, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		fmt.Printf("Unable to marshal ECDSA private key: %v", err)
		return nil
	}

	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
}

func CreateSSLConfig(hosts []string, validity time.Duration) (ca, cert, key []byte, err error) {
	notBefore := time.Now()
	notAfter := notBefore.Add(validity)

	caTemplate, caKey, err := createCA(hosts, notBefore, notAfter)
	if err != nil {
		return nil, nil, nil, err
	}
	certTemplate, certKey, err := createCert(hosts, notBefore, notAfter)
	if err != nil {
		return nil, nil, nil, err
	}

	caDerBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, err
	}

	ca = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDerBytes})

	certDerBytes, err := x509.CreateCertificate(rand.Reader, certTemplate, caTemplate, &certKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, err
	}
	cert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDerBytes})

	return ca, cert, pemBlockForKey(certKey), nil
}
