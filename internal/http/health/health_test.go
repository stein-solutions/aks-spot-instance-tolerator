package health

import (
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestServer(t *testing.T) {

	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}

var _ = Describe("probes", func() {
	Context("healthz", func() {
		resp, err := http.Get("http://localhost:8080/healthz")
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})

	Context("readyz", func() {
		resp, err := http.Get("http://localhost:8080/readyz")
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})

})
