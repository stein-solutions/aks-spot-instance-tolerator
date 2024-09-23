package health

import (
	"log/slog"
	"net/http"

	"github.com/stein-solutions/aks-spot-instance-tolerator/internal/config"
)

func StartHealthProbes(config *config.Config) {

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	go http.ListenAndServe("0.0.0.0:"+config.HealthPort, mux)
	slog.Info("Started Health Probes")
}
