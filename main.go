package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/stein-solutions/aks-spot-instance-tolerator/internal/config"
	"github.com/stein-solutions/aks-spot-instance-tolerator/internal/controller"
	"github.com/stein-solutions/aks-spot-instance-tolerator/internal/http"
	"github.com/stein-solutions/aks-spot-instance-tolerator/internal/http/health"
	"github.com/stein-solutions/aks-spot-instance-tolerator/internal/k8sClient"
	"github.com/stein-solutions/aks-spot-instance-tolerator/internal/util"
)

func main() {
	slog.Info("Starting...")
	config := config.NewConfig()
	slog.SetLogLoggerLevel(config.LogLevel)

	watcher := util.NewSecretWatcher(config.CertDirPath)
	watcher.WatchSecret()

	ch := make(chan bool)
	webhookController := controller.NewWebhookController(k8sClient.NewK8sClientDefault(), config, watcher)
	go webhookController.StartWebhookController(ch)

	success := <-ch
	if !success {
		fmt.Println("Failed to initialize webhook")
		os.Exit(1)
	}
	slog.Info("Webhook Controller initialized successfully - Starting Server")
	http.StartHttpServer(config, watcher)

	health.StartHealthProbes(config)

	select {}
}
