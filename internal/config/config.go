package config

import (
	"os"
	"time"
)

type Config struct {
	Namespace            string
	SvcName              string
	SecretName           string
	KubeConfig           string
	CertDirPath          string
	WebhookName          string
	WebhookPort          string
	HealthPort           string
	TlsValidForSeconds   int
	TlsRenewEarlySeconds int
}

func NewConfig() *Config {
	return &Config{
		Namespace:            getNamespace(),
		SvcName:              getServiceName(),
		SecretName:           getSecretName(),
		KubeConfig:           getKubeConfig(),
		CertDirPath:          getCertDirPath(),
		WebhookName:          getWebhookName(),
		WebhookPort:          getWebhookPort(),
		HealthPort:           getHealthPort(),
		TlsValidForSeconds:   int(time.Hour.Seconds() * 24 * 10),
		TlsRenewEarlySeconds: int(time.Hour.Seconds() * 24 * 5),
	}
}

func getWebhookPort() string {
	if webhookPort, exists := os.LookupEnv("AKS_SPOT_INSTANCE_TOLERATOR_WEBHOOK_PORT"); exists {
		return webhookPort
	}
	return "8443"
}

func getHealthPort() string {
	if webhookPort, exists := os.LookupEnv("AKS_SPOT_INSTANCE_TOLERATOR_HEALTH_PORT"); exists {
		return webhookPort
	}
	return "8080"
}

func getWebhookName() string {
	if webhookName, exists := os.LookupEnv("AKS_SPOT_INSTANCE_TOLERATOR_WEBHOOK_NAME"); exists {
		return webhookName
	}
	return "aks-spot-instance-tolerator-webhook"
}

func getServiceName() string {
	if svcName, exists := os.LookupEnv("AKS_SPOT_INSTANCE_TOLERATOR_SVC_NAME"); exists {
		return svcName
	}
	return "aks-spot-instance-tolerator-webhook-svc"
}

func getSecretName() string {
	if secretName, exists := os.LookupEnv("AKS_SPOT_INSTANCE_TOLERATOR_SECRET_NAME"); exists {
		return secretName
	}
	return "aks-spot-instance-tolerator-webhook-tls"
}

func getCertDirPath() string {
	if certDirPath, exists := os.LookupEnv("AKS_SPOT_INSTANCE_TOLERATOR_CERT_DIR_PATH"); exists {
		return certDirPath
	}
	return "/etc/webhook/certs"
}

func getKubeConfig() string {
	if kubeConfig, exists := os.LookupEnv("AKS_SPOT_INSTANCE_TOLERATOR_KUBECONFIG"); exists {
		return kubeConfig
	}
	homeDir, _ := os.UserHomeDir()
	return homeDir + "/.kube/config"
}

func getNamespace() string {
	if ns, exists := os.LookupEnv("POD_NAMESPACE"); exists {
		return ns
	}
	return "default"
}
