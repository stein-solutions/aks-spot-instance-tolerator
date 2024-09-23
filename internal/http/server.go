package http

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/stein-solutions/aks-spot-instance-tolerator/internal/config"
	"github.com/stein-solutions/aks-spot-instance-tolerator/internal/util"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	scheme = runtime.NewScheme()
	codecs = serializer.NewCodecFactory(scheme)
)

type Server struct {
}

func StartHttpServer(cfg *config.Config, fileWatcher *util.SecretWatcher) *http.Server {
	if cfg == nil {
		cfg = config.NewConfig()
	}

	tlsConfig := &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			certFile := filepath.Join(cfg.CertDirPath, "tls.cert")
			keyFile := filepath.Join(cfg.CertDirPath, "tls.key")

			certData, exist := fileWatcher.GetSecretData(certFile)
			if !exist {
				return nil, fmt.Errorf("cert file not found. file: %s", certFile)
			}
			keyData, exist := fileWatcher.GetSecretData(keyFile)
			if !exist {
				return nil, fmt.Errorf("key File not found")
			}

			// Zertifikate und Schlüsseldateien laden
			cert, err := tls.X509KeyPair([]byte(certData), []byte(keyData))
			if err != nil {
				return nil, err
			}

			return &cert, nil

		},
	}

	server := http.Server{
		Addr:      "0.0.0.0:" + cfg.WebhookPort,
		Handler:   &Server{},
		TLSConfig: tlsConfig,
	}

	slog.Info(fmt.Sprintf("Listening on port %s\n", cfg.WebhookPort))
	go func() {
		err := server.ListenAndServeTLS("", "")
		if err != nil {
			//todo: probably abort the program here
			slog.Error(fmt.Sprintf("Failed to listen on port %s: %v", cfg.WebhookPort, err))
		}
	}()

	return &server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		if data, err := io.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	if len(body) == 0 {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	review := admissionv1.AdmissionReview{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, &review); err != nil {
		http.Error(w, fmt.Sprintf("could not deserialize request: %v", err), http.StatusBadRequest)
		return
	}

	response := admissionv1.AdmissionReview{
		TypeMeta: review.TypeMeta,
		Response: &admissionv1.AdmissionResponse{
			UID:     review.Request.UID,
			Allowed: true,
		},
	}

	if review.Request.Kind.Kind == "Pod" {
		patch := `[
			{
				"op": "add",
				"path": "/spec/tolerations",
				"value": [
					{
						"key": "kubernetes.azure.com/scalesetpriority",
						"operator": "Equal",
						"value": "spot",
						"effect": "NoSchedule"
					}
				]
			}
		]`
		patchType := admissionv1.PatchTypeJSONPatch
		response.Response.Patch = []byte(patch)
		response.Response.PatchType = &patchType
	}

	respBytes, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not serialize response: %v", err), http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(respBytes); err != nil {
		http.Error(w, fmt.Sprintf("could not write response: %v", err), http.StatusInternalServerError)
	}
}
