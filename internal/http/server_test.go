package http

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stein-solutions/aks-spot-instance-tolerator/internal/config"
	"github.com/stein-solutions/aks-spot-instance-tolerator/internal/util"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestServer(t *testing.T) {

	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = Describe("Server", func() {
	Context("tls", func() {

		var (
			mockCfg *config.Config
			pwd     string
		)

		BeforeEach(func() {
			mockCfg = config.NewConfig()
			pwdLocal, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			pwd = pwdLocal

			port, err := GetFreePort()
			Expect(err).NotTo(HaveOccurred())
			mockCfg.WebhookPort = strconv.Itoa(port)
		})

		It("should handle tls traffic", func() {
			mockCfg.CertDirPath = filepath.Join(pwd, "/../../data/test/internal/http/testCerts/certs/example1/")

			mockFileWatcher := util.NewSecretWatcher(mockCfg.CertDirPath)
			err := mockFileWatcher.WatchSecret()
			Expect(err).NotTo(HaveOccurred())

			StartHttpServer(mockCfg, mockFileWatcher)

			time.Sleep(1 * time.Second)

			client := getHttpClient(mockCfg)
			resp, err := client.Get("https://0.0.0.0:" + mockCfg.WebhookPort)

			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(400))
		})

		It("should handle certificate renewal", func() {
			example1TlsPath := filepath.Join(pwd, "/../../data/test/internal/http/testCerts/certs/example1/")
			pathToWatch, err := os.MkdirTemp(os.TempDir(), "aks-spot-instance-tolerator-test")
			Expect(err).NotTo(HaveOccurred())

			mockCfg.CertDirPath = pathToWatch

			copyFile(filepath.Join(example1TlsPath, "tls.cert"), filepath.Join(pathToWatch, "tls.cert"))
			copyFile(filepath.Join(example1TlsPath, "tls.key"), filepath.Join(pathToWatch, "tls.key"))

			mockFileWatcher := util.NewSecretWatcher(mockCfg.CertDirPath)
			err = mockFileWatcher.WatchSecret()
			Expect(err).NotTo(HaveOccurred())

			StartHttpServer(mockCfg, mockFileWatcher)

			clientOldTls := getHttpClient(mockCfg)
			_, err = clientOldTls.Get("https://0.0.0.0:" + mockCfg.WebhookPort)
			Expect(err).NotTo(HaveOccurred())

			example2TlsPath := filepath.Join(pwd, "/../../data/test/internal/http/testCerts/certs/example2/")
			copyFile(filepath.Join(example2TlsPath, "tls.cert"), filepath.Join(pathToWatch, "tls.cert"))
			copyFile(filepath.Join(example2TlsPath, "tls.key"), filepath.Join(pathToWatch, "tls.key"))

			time.Sleep(2 * time.Second)

			clientNewTls := getHttpClient(mockCfg)
			_, err = clientNewTls.Get("https://0.0.0.0:" + mockCfg.WebhookPort)
			Expect(err).NotTo(HaveOccurred())

			_, err = clientOldTls.Get("https://0.0.0.0:" + mockCfg.WebhookPort)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("MutatingWebhook", func() {
		It("should return 400 on invalid input", func() {
			requestBytes, err := json.Marshal("empty body")
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequest("POST", "/mutate", bytes.NewReader(requestBytes))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc((&Server{}).ServeHTTP)

			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(400))
		})

		It("should return admissionreview with added toleration ", func() {
			//Create a sample admission review request
			request := admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UID: "12345",
					Kind: metav1.GroupVersionKind{
						Group:   "",
						Version: "v1",
						Kind:    "Pod",
					},
					Object: runtime.RawExtension{
						Raw: []byte(`{"metadata": {"name": "test-pod"}}`),
					},
				},
			}

			requestBytes, err := json.Marshal(request)
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequest("POST", "/mutate", bytes.NewReader(requestBytes))
			Expect(err).NotTo(HaveOccurred())

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc((&Server{}).ServeHTTP)

			handler.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(200))

			respBody, err := io.ReadAll(rr.Body)
			Expect(err).NotTo(HaveOccurred())

			response := admissionv1.AdmissionReview{}
			err = json.Unmarshal(respBody, &response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Response).NotTo(BeNil())
			Expect(response.Response.Patch).NotTo(BeNil())

			expectedPatch := `[
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
			Expect(string(response.Response.Patch)).To(Equal(expectedPatch))
		})
	})
})

// GetFreePort asks the kernel for a free open port that is ready to use.
func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func copyFile(src, dst string) error {
	// Open the source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create the destination file
	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// Copy the contents from source to destination
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	// Flush the file to ensure data is written
	err = destinationFile.Sync()
	if err != nil {
		return err
	}

	return nil
}

func getHttpClient(mockCfg *config.Config) *http.Client {
	certPath := filepath.Join(mockCfg.CertDirPath, "tls.cert")

	// Laden des Zertifikats
	cert, err := os.ReadFile(certPath)
	Expect(err).NotTo(HaveOccurred())

	// Erstellen eines neuen Zertifikatspools und Hinzufügen des Zertifikats
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(cert)

	// Erstellen einer benutzerdefinierten TLS-Konfiguration
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}

	// Erstellen eines benutzerdefinierten HTTP-Transports
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	// Erstellen eines HTTP-Clients mit dem benutzerdefinierten Transport
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	return client
}
