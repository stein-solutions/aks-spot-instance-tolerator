package health

import (
	"net"
	"net/http"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stein-solutions/aks-spot-instance-tolerator/internal/config"
)

func TestServer(t *testing.T) {

	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = Describe("probes", func() {

	var destinationPort string

	BeforeEach(func() {
		port, err := GetFreePort()
		Expect(err).NotTo(HaveOccurred())

		destinationPort = strconv.Itoa(port)
		cfg := &config.Config{
			HealthPort: destinationPort,
		}

		go StartHealthProbes(cfg)
	})

	Context("healthz", func() {
		It("should return status OK", func() {
			resp, err := http.Get("http://localhost:" + destinationPort + "/healthz")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	Context("readyz", func() {
		It("should return status OK", func() {
			resp, err := http.Get("http://localhost:" + destinationPort + "/readyz")
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
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
