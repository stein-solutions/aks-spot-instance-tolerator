package controller

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stein-solutions/aks-spot-instance-tolerator/internal/config" // Add this import
	"github.com/stein-solutions/aks-spot-instance-tolerator/internal/util"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

type MockK8sClient struct {
	_clientset kubernetes.Interface
}

func NewMockK8sClient(objects ...runtime.Object) *MockK8sClient {
	return &MockK8sClient{
		_clientset: fake.NewSimpleClientset(objects...),
	}
}

func (m *MockK8sClient) Clientset() kubernetes.Interface {
	return m._clientset
}

func TestUpdateSecret_ShouldRenewCert(t *testing.T) {
	config := config.NewConfig()

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.SecretName,
			Namespace: config.Namespace,
		},
		Data: map[string][]byte{
			"ca.cert": []byte("old-cert"),
		},
	}
	webhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.WebhookName,
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte("old-cert"),
				},
			},
		},
	}

	config.TlsValidForSeconds = 3
	config.TlsRenewEarlySeconds = 1
	k8sClient := NewMockK8sClient(webhookConfig)
	fileWatcher := util.NewSecretWatcher("")

	resChan := make(chan bool)
	controller := NewWebhookController(k8sClient, config, fileWatcher)
	controller.StartWebhookController(resChan)

	_, err := k8sClient.Clientset().CoreV1().Secrets(config.Namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	<-resChan

	firstSecret, err := k8sClient.Clientset().CoreV1().Secrets(config.Namespace).Get(context.TODO(), config.SecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = validateCertificates(firstSecret.Data["tls.cert"], firstSecret.Data["tls.key"], firstSecret.Data["ca.cert"],
		[]string{"aks-spot-instance-tolerator-webhook-svc.default.svc.cluster.local", "aks-spot-instance-tolerator-webhook-svc.default.svc",
			"aks-spot-instance-tolerator-webhook-svc"})

	if err != nil {
		t.Fatalf("certificate validation failed: %v", err)
	}

	time.Sleep(4 * time.Second)
	secondSecret, err := k8sClient.Clientset().CoreV1().Secrets(config.Namespace).Get(context.TODO(), config.SecretName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = validateCertificates(secondSecret.Data["tls.cert"], secondSecret.Data["tls.key"], secondSecret.Data["ca.cert"],
		[]string{"aks-spot-instance-tolerator-webhook-svc.default.svc.cluster.local", "aks-spot-instance-tolerator-webhook-svc.default.svc",
			"aks-spot-instance-tolerator-webhook-svc"})
	if err != nil {
		t.Fatalf("certificate validation failed: %v", err)
	}
}

func TestUpdateSecre_HappyCase(t *testing.T) {
	config := config.NewConfig()

	// Mock data
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"ca.cert": []byte("old-cert"),
		},
	}
	webhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: config.WebhookName,
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte("old-cert"),
				},
			},
		},
	}

	k8sClient := NewMockK8sClient(secret, webhookConfig)

	fileWatcher := util.NewSecretWatcher("")

	controller := NewWebhookController(k8sClient, config, fileWatcher)

	err := controller.updateSecret(secret)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updatedSecret, err := k8sClient.Clientset().CoreV1().Secrets(config.Namespace).Get(context.TODO(), "test-secret", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = validateCertificates(updatedSecret.Data["tls.cert"], updatedSecret.Data["tls.key"], updatedSecret.Data["ca.cert"],
		[]string{"aks-spot-instance-tolerator-webhook-svc.default.svc.cluster.local", "aks-spot-instance-tolerator-webhook-svc.default.svc",
			"aks-spot-instance-tolerator-webhook-svc"})
	if err != nil {
		t.Fatalf("certificate validation failed: %v", err)
	}

	_, exist := fileWatcher.GetSecretData(filepath.Join(config.CertDirPath, "tls.cert"))
	if err != nil {
		t.Fatalf("expected tls.cert to be in file watcher, got %v", exist)
	}
	_, exist = fileWatcher.GetSecretData(filepath.Join(config.CertDirPath, "tls.key"))
	if err != nil {
		t.Fatalf("expected tls.key to be in file watcher, got %v", exist)
	}
	_, exist = fileWatcher.GetSecretData(filepath.Join(config.CertDirPath, "ca.cert"))
	if err != nil {
		t.Fatalf("expected ca.cert to be in file watcher, got %v", exist)
	}
}

func validateCertificates(tlsCertPEM, tlsKeyPEM, caCertPEM []byte, domains []string) error {
	// TLS-Zertifikat parsen
	tlsCertBlock, _ := pem.Decode(tlsCertPEM)
	if tlsCertBlock == nil {
		return fmt.Errorf("konnte tls.cert nicht parsen")
	}
	tlsCert, err := x509.ParseCertificate(tlsCertBlock.Bytes)
	if err != nil {
		return fmt.Errorf("konnte tls.cert nicht parsen: %v", err)
	}

	// TLS-Schlüssel parsen
	tlsKeyBlock, _ := pem.Decode(tlsKeyPEM)
	if tlsKeyBlock == nil {
		return fmt.Errorf("konnte tls.key nicht parsen")
	}
	_, err = x509.ParseECPrivateKey(tlsKeyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("konnte tls.key nicht parsen: %v", err)
	}

	// CA-Zertifikat parsen
	caCertBlock, _ := pem.Decode(caCertPEM)
	if caCertBlock == nil {
		return fmt.Errorf("konnte ca.cert nicht parsen")
	}
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return fmt.Errorf("konnte ca.cert nicht parsen: %v", err)
	}

	// Zertifikatskette validieren
	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	for _, domain := range domains {
		opts := x509.VerifyOptions{
			DNSName: domain,
			Roots:   roots,
		}
		if _, err := tlsCert.Verify(opts); err != nil {
			return fmt.Errorf("tls.cert ist nicht gültig für die Domäne %s: %v", domain, err)
		}
	}

	// Überprüfen, ob das Zertifikat derzeit gültig ist
	now := time.Now()
	if now.Before(tlsCert.NotBefore) || now.After(tlsCert.NotAfter) {
		return fmt.Errorf("tls.cert ist derzeit nicht gültig")
	}

	return nil
}
