package controller

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"log/slog"
	"path/filepath"
	"slices"
	"time"

	"github.com/stein-solutions/aks-spot-instance-tolerator/internal/config"
	"github.com/stein-solutions/aks-spot-instance-tolerator/internal/k8sClient"
	myutil "github.com/stein-solutions/aks-spot-instance-tolerator/internal/util"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WebhookController struct {
	k8sClient             k8sClient.K8sClientInterface
	config                *config.Config
	fileSystemWatcher     *myutil.SecretWatcher
	reconcileScheduledFor time.Time
}

func (wc *WebhookController) getHostnames() []string {
	return []string{wc.config.SvcName + "." + wc.config.Namespace + ".svc.cluster.local",
		wc.config.SvcName + "." + wc.config.Namespace + ".svc", wc.config.SvcName}
}

func (wc *WebhookController) scheduleSecretReconcilationFor(until time.Time) {
	if wc.reconcileScheduledFor.Equal(until) {
		return
	}

	wc.reconcileScheduledFor = until
	go func() {
		slog.Info(fmt.Sprintf("WebhookController - scheduled reconcile for %s.", until.String()))
		time.Sleep(time.Until(until))
		slog.Info(fmt.Sprintf("WebhookController - periodic reconcile fired. Its %s.", time.Now().String()))

		secret, err := wc.k8sClient.Clientset().CoreV1().Secrets(wc.config.Namespace).Get(context.TODO(), wc.config.SecretName, metav1.GetOptions{})
		if err != nil {
			slog.Error(fmt.Sprintf("Error getting secret. %s", err))
		}
		wc.reconcileSecret(secret)
	}()
}

func (wc *WebhookController) updateSecret(secret *v1.Secret) error {
	ca, cert, certKey, err := CreateSSLConfig(wc.getHostnames(),
		time.Second*time.Duration(wc.config.TlsValidForSeconds))

	if err != nil {
		log.Fatal(err)
	}
	secret.Data = make(map[string][]byte)
	oldCert := secret.Data["ca.cert"]
	err = wc.updateWebhook(oldCert, ca)
	if err != nil {
		return err
	}
	secret.Data["tls.cert"] = cert
	secret.Data["tls.key"] = certKey
	secret.Data["ca.cert"] = ca
	_, err = wc.k8sClient.Clientset().CoreV1().Secrets(wc.config.Namespace).Update(context.TODO(), secret, metav1.UpdateOptions{})
	if err != nil {
		fmt.Println("Error updating secret:", err)
	}
	slog.Info("Secret updated successfully")

	wc.fileSystemWatcher.PutSecretDataOverride(filepath.Join(wc.config.CertDirPath, "tls.cert"), string(cert))
	wc.fileSystemWatcher.PutSecretDataOverride(filepath.Join(wc.config.CertDirPath, "tls.key"), string(certKey))
	wc.fileSystemWatcher.PutSecretDataOverride(filepath.Join(wc.config.CertDirPath, "ca.cert"), string(ca))

	return nil
}

func (wc *WebhookController) updateWebhook(oldCert []byte, newCert []byte) error {

	caBundle := append(oldCert, newCert...)

	webhookName := wc.config.WebhookName
	webhookConfiguration, err := wc.k8sClient.Clientset().AdmissionregistrationV1().MutatingWebhookConfigurations().
		Get(context.TODO(), webhookName, metav1.GetOptions{})
	if err != nil {
		slog.Error(fmt.Sprintf("Error getting webhook configuration. %s", err))
		return err
	}
	webhookConfiguration.Webhooks[0].ClientConfig.CABundle = caBundle
	_, err = wc.k8sClient.Clientset().AdmissionregistrationV1().MutatingWebhookConfigurations().
		Update(context.TODO(), webhookConfiguration, metav1.UpdateOptions{})
	if err != nil {
		slog.Error(fmt.Sprintf("Error updating webhook configuration. %s", err))
		return err
	}

	slog.Info("Webhook CR updated successfully")
	return nil
}

func (wc *WebhookController) reconcileSecret(secret *v1.Secret) error {

	if _, exists := secret.Data["tls.cert"]; !exists {
		slog.Info("tls.cert not found in secret. Creating...")
		return wc.updateSecret(secret)
	}

	if _, exists := secret.Data["tls.key"]; !exists {
		slog.Info("tls.key not found in secret. Creating...")
		return wc.updateSecret(secret)
	}

	if _, exists := secret.Data["ca.cert"]; !exists {
		slog.Info("ca.cert not found in secret. Creating...")
		return wc.updateSecret(secret)
	}

	block, _ := pem.Decode(secret.Data["tls.cert"])
	if block == nil {
		slog.Info("Error decoding tls.cert. Thus creating...")
		return wc.updateSecret(secret)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		slog.Info("Error parsing tls.cert. Thus creating...")
		return wc.updateSecret(secret)
	}

	now := time.Now()
	notAfterWithGrace := cert.NotAfter.Add(-time.Second * time.Duration(wc.config.TlsRenewEarlySeconds))
	if now.Before(cert.NotBefore) || now.After(notAfterWithGrace) {
		slog.Info("tls.cert is expired. Updating...")
		return wc.updateSecret(secret)
	}

	for _, hostname := range wc.getHostnames() {
		if !slices.Contains(cert.DNSNames, hostname) {
			slog.Info(fmt.Sprintf("tls.cert is not valid for %s. Updating...", hostname))
			return wc.updateSecret(secret)
		}
	}

	wc.scheduleSecretReconcilationFor(cert.NotAfter.Add(-time.Second * time.Duration(wc.config.TlsRenewEarlySeconds)))

	slog.Info("Secret is up to date. Nothing to reconcile")
	return nil
}

func NewWebhookController(client k8sClient.K8sClientInterface, config *config.Config, fileSystemWatcher *myutil.SecretWatcher) *WebhookController {
	controller := WebhookController{
		k8sClient:         client,
		config:            config,
		fileSystemWatcher: fileSystemWatcher,
	}

	return &controller
}

func (wc *WebhookController) StartWebhookController(ch chan<- bool) {
	slog.Info("Starting webhook controller")

	watcher, err := wc.k8sClient.Clientset().CoreV1().Secrets(wc.config.Namespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", wc.config.SecretName),
	})

	slog.Debug(fmt.Sprintf("Watching secret: %s", wc.config.SecretName))
	if err != nil {
		slog.Error(fmt.Sprintf("Error watching secret. Aborting... %s", err.Error()))
		return
	}

	go func() {
		for event := range watcher.ResultChan() {
			secret := event.Object.(*v1.Secret)
			err := wc.reconcileSecret(secret)
			if err == nil {
				ch <- true
			}
		}
	}()
}
