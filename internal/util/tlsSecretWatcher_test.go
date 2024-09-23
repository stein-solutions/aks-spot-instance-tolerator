package util

import (
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSecretWatcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "SecretWatcher Suite")
}

var _ = Describe("SecretWatcher", func() {
	var (
		sw         *SecretWatcher
		path       string
		tempFolder string
	)

	BeforeEach(func() {
		// Erstellen Sie eine temporäre Datei für Tests
		tmpFile, err := os.CreateTemp("", "secret")
		Expect(err).NotTo(HaveOccurred())
		tmpFile.WriteString("secret")
		path = tmpFile.Name()
		tmpFile.Close()

		sw = NewSecretWatcher(path)
	})

	AfterEach(func() {
		// Entfernen Sie die temporäre Datei nach den Tests
		os.Remove(path)
	})

	Describe("NewSecretWatcher", func() {
		Context("happy case", func() {
			It("should create a new SecretWatcher", func() {
				Expect(sw).NotTo(BeNil())
				Expect(sw.path).To(Equal(path))
				Expect(sw.secretData).To(BeEmpty())
			})
		})
	})

	Describe("SecretOverrides", func() {

		var (
			testFile *os.File
		)
		BeforeEach(func() {
			folder, err := os.MkdirTemp(os.TempDir(), "aks-spot-instance-tolerator-test-")
			tempFolder = folder
			Expect(err).NotTo(HaveOccurred())
			tmpFile, err := os.CreateTemp(tempFolder, "secret")
			Expect(err).NotTo(HaveOccurred())
			testFile = tmpFile
			tmpFile.WriteString("secret")
			tmpFile.Close()

			sw = NewSecretWatcher(tempFolder)
			sw.LoadSecret()

			data, exits := sw.GetSecretData(tmpFile.Name())
			Expect(exits).To(BeTrue())
			Expect(data).To(Equal("secret"))
		})

		Context("when adding a secret override", func() {
			It("should return overridden value", func() {
				sw.PutSecretDataOverride(testFile.Name(), "not so secret")

				data, exits := sw.GetSecretData(testFile.Name())
				Expect(exits).To(BeTrue())
				Expect(data).To(Equal("not so secret"))
			})
		})

		Context("when getting a secret override", func() {
			It("it should reset on next load", func() {
				sw.PutSecretDataOverride(testFile.Name(), "not so secret")

				data, exits := sw.GetSecretData(testFile.Name())
				Expect(exits).To(BeTrue())
				Expect(data).To(Equal("not so secret"))

				sw.LoadSecret()

				data, exits = sw.GetSecretData(testFile.Name())
				Expect(exits).To(BeTrue())
				Expect(data).To(Equal("secret"))
			})
		})

	})

	Describe("LoadSecret", func() {

		Context("when the path is a directory", func() {
			It("should load all the files in the directory", func() {
				// Create a new temporary directory with a prefix "myapp-"
				tempFolder, err := os.MkdirTemp(os.TempDir(), "aks-spot-instance-tolerator-test-")
				Expect(err).NotTo(HaveOccurred())
				tmpFile, err := os.CreateTemp(tempFolder, "secret")
				Expect(err).NotTo(HaveOccurred())
				tmpFile.WriteString("secret")
				tmpFile.Close()

				tmpFile, err = os.CreateTemp(tempFolder, "secret2")
				Expect(err).NotTo(HaveOccurred())
				tmpFile.WriteString("some other secret")
				tmpFile.Close()

				sw = NewSecretWatcher(tempFolder)
				sw.LoadSecret()

				Expect(sw.secretData).To(ContainElement("secret"))
				Expect(sw.secretData).To(ContainElement("some other secret"))
			})

			It("should ignore subdirectories", func() {
				// Create a new temporary directory with a prefix "myapp-"
				tempFolder, err := os.MkdirTemp(os.TempDir(), "aks-spot-instance-tolerator-test-")
				Expect(err).NotTo(HaveOccurred())
				tmpFile, err := os.CreateTemp(tempFolder, "secret")
				Expect(err).NotTo(HaveOccurred())
				tmpFile.WriteString("secret")
				tmpFile.Close()

				_, err = os.MkdirTemp(tempFolder, "another_directory")
				Expect(err).NotTo(HaveOccurred())

				sw = NewSecretWatcher(tempFolder)
				err = sw.LoadSecret()
				Expect(err).NotTo(HaveOccurred())

				Expect(sw.secretData).To(ContainElement("secret"))
				Expect(sw.secretData).NotTo(ContainElement("another_directory"))

			})
		})

		Context("when the secret file exists", func() {
			It("should load the secret without error", func() {
				err := sw.LoadSecret()
				Expect(err).NotTo(HaveOccurred())
			})

			It("should contain the secret data", func() {
				err := sw.LoadSecret()
				Expect(err).NotTo(HaveOccurred())
				Expect(sw.secretData).To(ContainElement("secret"))
			})

			It("should have the file system path as key", func() {
				sw.LoadSecret()
				_, exist := sw.secretData[path]
				Expect(exist).To(BeTrue())
			})
		})

		Context("when the secret file does not exist", func() {
			It("should return an error", func() {
				sw.path = "nonexistentfile"
				err := sw.LoadSecret()
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the secret changes", func() {
			It("should not reload the secret", func() {
				err := sw.LoadSecret()
				Expect(err).NotTo(HaveOccurred())

				tmpFile, err := os.Create(sw.path)
				Expect(err).NotTo(HaveOccurred())
				tmpFile.WriteString("secret2")
				path = tmpFile.Name()
				tmpFile.Close()

				Expect(sw.secretData).NotTo(ContainElement("secret2"))
				Expect(sw.secretData).To(ContainElement("secret"))
			})
		})
	})

	Describe("WatchSecret", func() {
		Context("when the secret changes", func() {
			It("should reload the secret after write event", func() {
				err := sw.WatchSecret()
				Expect(err).NotTo(HaveOccurred())

				tmpFile, err := os.Create(sw.path)
				Expect(err).NotTo(HaveOccurred())
				tmpFile.WriteString("secret2")
				tmpFile.Close()

				Eventually(func() string {
					return sw.secretData[sw.path]
				}, time.Second*10, time.Second).Should(Equal("secret2"))

				Expect(sw.secretData).NotTo(ContainElement("secret"))
			})

		})
	})

	Context("when the path is a directory", func() {
		It("should reload all secrets on write", func() {

			tempFolder, err := os.MkdirTemp(os.TempDir(), "aks-spot-instance-tolerator-test-")
			Expect(err).NotTo(HaveOccurred())
			tmpFile, err := os.CreateTemp(tempFolder, "secret")
			Expect(err).NotTo(HaveOccurred())
			tmpFile.WriteString("secret")
			tmpFile.Close()

			tmpFile, err = os.CreateTemp(tempFolder, "secret2")
			Expect(err).NotTo(HaveOccurred())
			tmpFile.WriteString("some other secret")
			tmpFile.Close()

			sw = NewSecretWatcher(tempFolder)
			err = sw.WatchSecret()
			Expect(err).NotTo(HaveOccurred())

			Expect(sw.secretData).To(ContainElement("secret"))
			Expect(sw.secretData).To(ContainElement("some other secret"))

			file, err := os.Create(tmpFile.Name())
			Expect(err).NotTo(HaveOccurred())
			file.WriteString("new secret")
			file.Close()

			Eventually(func() string {
				return sw.secretData[tmpFile.Name()]
			}, time.Second*10, time.Second).Should(Equal("new secret"))
			Expect(sw.secretData).To(ContainElement("secret"))
		})

		It("should load new secrets", func() {

			tempFolder, err := os.MkdirTemp(os.TempDir(), "aks-spot-instance-tolerator-test-")
			Expect(err).NotTo(HaveOccurred())
			tmpFile1, err := os.CreateTemp(tempFolder, "secret-*.txt")
			Expect(err).NotTo(HaveOccurred())
			tmpFile1.WriteString("secret")
			tmpFile1.Close()

			sw = NewSecretWatcher(tempFolder)
			err = sw.WatchSecret()
			Expect(err).NotTo(HaveOccurred())

			tmpFile2, err := os.CreateTemp(tempFolder, "secret-new-*.txt")
			Expect(err).NotTo(HaveOccurred())
			tmpFile2.Close()

			Eventually(func() bool {
				_, exist := sw.secretData[tmpFile2.Name()]
				return exist
			}, time.Second*60, time.Second).Should(BeTrue())

			Expect(sw.secretData).To(HaveKey(tmpFile2.Name()))
		})

	})
})
