package main_test

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

func startMainWithArgs(args ...string) *gexec.Session {
	command := exec.Command(garagepiBinPath, args...)
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
	Eventually(session).Should(gbytes.Say("garagepi starting"))
	return session
}

func validateSuccessAnyLengthBody(resp *http.Response, err error) {
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	validateBody(resp, true)
}

func validateSuccessNonZeroLengthBody(resp *http.Response) {
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	validateBody(resp, false)
}

func validateBody(resp *http.Response, anySize bool) {
	body, err := ioutil.ReadAll(resp.Body)
	Expect(err).NotTo(HaveOccurred())

	if anySize {
		Expect(len(body)).Should(BeNumerically(">=", 0))
	} else {
		Expect(len(body)).Should(BeNumerically(">", 0))
	}
}

var _ = Describe("GaragepiExecutable", func() {
	var (
		args []string
	)

	BeforeEach(func() {
		args = []string{}
	})

	Describe("long-running operation", func() {
		var (
			session *gexec.Session
		)

		AfterEach(func() {
			session.Terminate()
		})

		Describe("routing", func() {
			BeforeEach(func() {
				args = append(args, fmt.Sprintf("-httpPort=%d", httpPort))
				args = append(args, "-dev")
				args = append(args, "-enableHTTPS=false")
				args = append(args, "-forceHTTPS=false")
			})

			It("Should accept GET requests to /", func() {
				session = startMainWithArgs(args...)
				Eventually(session).Should(gbytes.Say("garagepi started"))

				resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", httpPort))
				Expect(err).NotTo(HaveOccurred())
				validateSuccessNonZeroLengthBody(resp)
			})

			It("Should reject GET requests to /api/v1/toggle with 404", func() {
				session = startMainWithArgs(args...)
				Eventually(session).Should(gbytes.Say("garagepi started"))

				resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/toggle", httpPort))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
			})

			It("Should accept POST requests to /api/v1/toggle", func() {
				session = startMainWithArgs(args...)
				Eventually(session).Should(gbytes.Say("garagepi started"))

				resp, err := http.Post(fmt.Sprintf("http://localhost:%d/api/v1/toggle", httpPort), "", strings.NewReader(""))
				Expect(err).NotTo(HaveOccurred())
				validateSuccessNonZeroLengthBody(resp)
			})

			It("Should accept GET requests to /api/v1/light", func() {
				session = startMainWithArgs(args...)
				Eventually(session).Should(gbytes.Say("garagepi started"))

				resp, err := http.Get(fmt.Sprintf("http://localhost:%d/api/v1/light", httpPort))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusServiceUnavailable))
			})

			It("Should accept POST requests to /api/v1/light", func() {
				session = startMainWithArgs(args...)
				Eventually(session).Should(gbytes.Say("garagepi started"))

				resp, err := http.Post(fmt.Sprintf("http://localhost:%d/api/v1/light", httpPort), "", strings.NewReader(""))
				Expect(err).NotTo(HaveOccurred())
				validateSuccessNonZeroLengthBody(resp)
			})

			It("Should accept GET requests to /webcam", func() {
				session = startMainWithArgs(args...)
				Eventually(session).Should(gbytes.Say("garagepi started"))

				resp, err := http.Get(fmt.Sprintf("http://localhost:%d/webcam", httpPort))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
			})

			It("Should serve static files", func() {
				session = startMainWithArgs(args...)
				Eventually(session).Should(gbytes.Say("garagepi started"))

				resp, err := http.Get(fmt.Sprintf("http://localhost:%d/static/css/application.css", httpPort))
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(http.StatusOK))
			})
		})

		Describe("request handling", func() {
			BeforeEach(func() {
				args = append(args, "-dev")
			})

			Context("when enableHTTP and enableHTTPS are both false", func() {
				BeforeEach(func() {
					args = append(args, "-enableHTTP=false")
					args = append(args, "-enableHTTPS=false")
				})

				It("exits with error", func() {
					session = startMainWithArgs(args...)
					Eventually(session).Should(gexec.Exit(2))
				})
			})

			Context("when enableHTTP is true and enableHTTPS is false", func() {
				BeforeEach(func() {
					args = append(args, "-enableHTTP=true")
					args = append(args, fmt.Sprintf("-httpPort=%d", httpPort))
					args = append(args, "-enableHTTPS=false")
				})

				Context("when forceHTTPS is false", func() {
					BeforeEach(func() {
						args = append(args, "-forceHTTPS=false")
					})

					It("accepts HTTP connections", func() {
						session = startMainWithArgs(args...)
						Eventually(session).Should(gbytes.Say("garagepi started"))

						resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", httpPort))
						Expect(err).NotTo(HaveOccurred())

						Expect(resp.StatusCode).To(Equal(http.StatusOK))
					})
				})

				Context("when forceHTTPS is true", func() {
					BeforeEach(func() {
						args = append(args, "-forceHTTPS=true")
					})

					It("exits with error", func() {
						session = startMainWithArgs(args...)
						Eventually(session).Should(gexec.Exit(2))
					})
				})
			})

			Context("when enableHTTPS is true", func() {
				BeforeEach(func() {
					args = append(args, "-enableHTTPS=true")
					args = append(args, fmt.Sprintf("-httpsPort=%d", httpsPort))
				})

				It("exits with error when -keyFile is not provided", func() {
					args = append(args, "-certFile=someCert")
					args = append(args, "-keyFile=")

					session = startMainWithArgs(args...)
					Eventually(session).Should(gexec.Exit(2))
				})

				It("exits with error when -certFile is not provided", func() {
					args = append(args, "-keyFile=someKey")
					args = append(args, "-certFile=")

					session = startMainWithArgs(args...)
					Eventually(session).Should(gexec.Exit(2))
				})

				Context("when both -certFile and -keyFile are provided", func() {
					var (
						keyFile  string
						certFile string

						client *http.Client
					)

					BeforeEach(func() {
						testDir := getDirOfCurrentFile()
						fixturesDir := filepath.Join(testDir, "..", "fixtures")
						keyFile = filepath.Join(fixturesDir, "key.pem")
						certFile = filepath.Join(fixturesDir, "cert.pem")

						args = append(args, "-keyFile="+keyFile)
						args = append(args, "-certFile="+certFile)

						// Load client cert
						cert, err := tls.LoadX509KeyPair(certFile, keyFile)
						if err != nil {
							log.Fatal(err)
						}

						// Load CA cert
						caCert, err := ioutil.ReadFile(certFile)
						if err != nil {
							log.Fatal(err)
						}
						caCertPool := x509.NewCertPool()
						caCertPool.AppendCertsFromPEM(caCert)

						// Setup HTTPS client
						tlsConfig := &tls.Config{
							Certificates: []tls.Certificate{cert},
							RootCAs:      caCertPool,
						}
						tlsConfig.BuildNameToCertificate()
						transport := &http.Transport{TLSClientConfig: tlsConfig}
						client = &http.Client{Transport: transport}
					})

					It("accepts HTTPS connections", func() {
						session = startMainWithArgs(args...)
						Eventually(session).Should(gbytes.Say("garagepi started"))

						resp, err := client.Get(fmt.Sprintf("https://localhost:%d/", httpsPort))
						Expect(err).NotTo(HaveOccurred())
						validateSuccessNonZeroLengthBody(resp)
					})

					Context("when enableHTTP is false", func() {
						BeforeEach(func() {
							args = append(args, "-enableHTTP=false")
						})

						Context("when forceHTTPS is true", func() {
							BeforeEach(func() {
								args = append(args, "-forceHTTPS=true")
							})

							It("exits with error", func() {
								session = startMainWithArgs(args...)
								Eventually(session).Should(gexec.Exit(2))
							})
						})
					})

					Context("when enableHTTP is true", func() {
						BeforeEach(func() {
							args = append(args, "-enableHTTP=true")
							args = append(args, fmt.Sprintf("-httpPort=%d", httpPort))
						})

						Context("when forceHTTPS is true", func() {
							BeforeEach(func() {
								args = append(args, "-forceHTTPS=true")
							})

							Context("when redirectPort is the same as HTTPSPort", func() {
								BeforeEach(func() {
									args = append(args, fmt.Sprintf("-redirectPort=%d", httpsPort))
								})

								It("redirects HTTP to provided redirect port", func() {
									session = startMainWithArgs(args...)
									Eventually(session).Should(gbytes.Say("garagepi started"))

									req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/", httpPort), nil)
									Expect(err).NotTo(HaveOccurred())

									transport := http.Transport{}
									resp, err := transport.RoundTrip(req)

									Expect(resp.StatusCode).To(Equal(http.StatusFound))

									expectedLocation := fmt.Sprintf("localhost:%d", httpsPort)

									location, err := resp.Location()
									Expect(err).NotTo(HaveOccurred())
									Expect(location.Scheme).To(Equal("https"))
									Expect(location.Host).To(Equal(expectedLocation))
								})
							})

							Context("when redirectPort differs from HTTPSPort", func() {
								var redirectPort int
								BeforeEach(func() {
									redirectPort = 34567
									args = append(args, fmt.Sprintf("-redirectPort=%d", redirectPort))
								})

								It("redirects HTTP to provided redirect port", func() {
									session = startMainWithArgs(args...)
									Eventually(session).Should(gbytes.Say("garagepi started"))

									req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/", httpPort), nil)
									Expect(err).NotTo(HaveOccurred())

									transport := http.Transport{}
									resp, err := transport.RoundTrip(req)

									Expect(resp.StatusCode).To(Equal(http.StatusFound))

									expectedLocation := fmt.Sprintf("localhost:%d", redirectPort)

									location, err := resp.Location()
									Expect(err).NotTo(HaveOccurred())
									Expect(location.Scheme).To(Equal("https"))
									Expect(location.Host).To(Equal(expectedLocation))
								})

							})
						})

						Context("when forceHTTPS is false", func() {
							BeforeEach(func() {
								args = append(args, "-forceHTTPS=false")
							})

							It("does not redirect HTTP to HTTPS", func() {
								session = startMainWithArgs(args...)
								Eventually(session).Should(gbytes.Say("garagepi started"))

								req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/", httpPort), nil)
								Expect(err).NotTo(HaveOccurred())

								transport := http.Transport{}
								resp, err := transport.RoundTrip(req)

								Expect(resp.StatusCode).To(Equal(http.StatusOK))
							})
						})
					})
				})
			})
		})

		Describe("authentication", func() {
			Context("when dev is enabled", func() {
				BeforeEach(func() {
					args = append(args, fmt.Sprintf("-httpPort=%d", httpPort))
					args = append(args, "-dev")
				})

				It("accepts unauthenticated requests", func() {
					session = startMainWithArgs(args...)
					Eventually(session).Should(gbytes.Say("garagepi started"))

					resp, err := http.Get(fmt.Sprintf("http://localhost:%d", httpPort))
					Expect(err).NotTo(HaveOccurred())
					validateSuccessNonZeroLengthBody(resp)
				})
			})

			Context("when dev is disabled", func() {
				BeforeEach(func() {
					args = append(args, fmt.Sprintf("-httpPort=%d", httpPort))
					args = append(args, "-dev=false")
				})

				It("exits with error when -username is not provided", func() {
					args = append(args, "-username=")
					args = append(args, "-password=password")

					session = startMainWithArgs(args...)
					Eventually(session).Should(gexec.Exit(2))
				})

				It("exits with error when -password is not provided", func() {
					args = append(args, "-username=username")
					args = append(args, "-password=")

					session = startMainWithArgs(args...)
					Eventually(session).Should(gexec.Exit(2))
				})

				Context("when username and password are provided", func() {
					BeforeEach(func() {
						args = append(args, "-username=some-user")
						args = append(args, "-password=teE73F4vf0")
					})

					It("redirects unauthenticated requests", func() {
						session = startMainWithArgs(args...)
						Eventually(session).Should(gbytes.Say("garagepi started"))

						req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/", httpPort), nil)
						Expect(err).NotTo(HaveOccurred())

						transport := http.Transport{}
						resp, err := transport.RoundTrip(req)
						Expect(err).NotTo(HaveOccurred())

						Expect(resp.StatusCode).To(Equal(http.StatusFound))
					})

					It("redirects unauthorized requests", func() {
						session = startMainWithArgs(args...)
						Eventually(session).Should(gbytes.Say("garagepi started"))

						req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/", httpPort), nil)
						Expect(err).NotTo(HaveOccurred())

						req.SetBasicAuth("baduser", "badpassword")

						transport := http.Transport{}
						resp, err := transport.RoundTrip(req)
						Expect(err).NotTo(HaveOccurred())

						Expect(resp.StatusCode).To(Equal(http.StatusFound))
					})

					It("accepts authorized requests", func() {
						session = startMainWithArgs(args...)
						Eventually(session).Should(gbytes.Say("garagepi started"))

						req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/", httpPort), nil)
						Expect(err).NotTo(HaveOccurred())

						req.SetBasicAuth("some-user", "teE73F4vf0")

						client := &http.Client{}
						resp, err := client.Do(req)
						Expect(err).NotTo(HaveOccurred())

						Expect(resp.StatusCode).To(Equal(http.StatusOK))
					})
				})
			})
		})

		Describe("Signal handling", func() {
			BeforeEach(func() {
				args = append(args, fmt.Sprintf("-httpPort=%d", httpPort))
				args = append(args, "-dev")
				args = append(args, "-enableHTTPS=false")
				args = append(args, "-forceHTTPS=false")
			})

			It("shuts downs when interrupted", func() {
				session = startMainWithArgs(args...)
				Eventually(session).Should(gbytes.Say("garagepi started"))

				session.Interrupt()
				Eventually(session).Should(gexec.Exit())
			})

			It("shuts downs when terminated", func() {
				session = startMainWithArgs(args...)
				Eventually(session).Should(gbytes.Say("garagepi started"))

				session.Terminate()
				Eventually(session).Should(gexec.Exit())
			})

			It("shuts downs when killed", func() {
				session = startMainWithArgs(args...)
				Eventually(session).Should(gbytes.Say("garagepi started"))

				session.Kill()
				Eventually(session).Should(gexec.Exit())
			})
		})

		Describe("Writing pid file", func() {
			var (
				tempDirPath string
				pidFilePath string
			)

			BeforeEach(func() {
				var err error
				tempDirPath, err = ioutil.TempDir(os.TempDir(), "garagepi-integration-test")
				Expect(err).NotTo(HaveOccurred())

				args = append(args, "-dev")
			})

			AfterEach(func() {
				err := os.RemoveAll(tempDirPath)
				Expect(err).ToNot(HaveOccurred())
			})

			Context("when the pidfile location is valid", func() {
				BeforeEach(func() {
					pidFilePath = fmt.Sprintf("%s/healthcheck.pid", tempDirPath)
					args = append(args, fmt.Sprintf("-pidFile=%s", pidFilePath))
				})

				It("writes its pid to the provided file", func() {
					Expect(fileExists(pidFilePath)).To(BeFalse())
					session = startMainWithArgs(args...)
					Eventually(session).Should(gbytes.Say("garagepi started"))
					Expect(fileExists(pidFilePath)).To(BeTrue())
				})
			})
		})
	})

	Describe("Invalid pidfile", func() {
		var (
			pidFilePath string
			tempDirPath string
		)

		BeforeEach(func() {
			var err error
			tempDirPath, err = ioutil.TempDir(os.TempDir(), "garagepi-integration-test")
			Expect(err).NotTo(HaveOccurred())

			pidFilePath = fmt.Sprintf("%s/invalid_path/healthcheck.pid", tempDirPath)
			args = append(args, fmt.Sprintf("-pidFile=%s", pidFilePath))
			args = append(args, "-dev")
		})

		It("exits with error", func() {
			session := startMainWithArgs(args...)

			Eventually(session.Err).Should(gbytes.Say(pidFilePath))
			Eventually(session).Should(gexec.Exit())
			Expect(session.ExitCode()).ToNot(Equal(0))
		})

	})

	Describe("Displaying version", func() {
		It("displays version with 'version'", func() {
			args = append(args, fmt.Sprintf("version"))

			command := exec.Command(garagepiBinPath, args...)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gbytes.Say("dev"))
			Eventually(session).Should(gexec.Exit(0))
		})

		It("displays version with '-v'", func() {
			args = append(args, fmt.Sprintf("-v"))

			command := exec.Command(garagepiBinPath, args...)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gbytes.Say("dev"))
			Eventually(session).Should(gexec.Exit(0))
		})

		It("displays version with '--version'", func() {
			args = append(args, fmt.Sprintf("--version"))

			command := exec.Command(garagepiBinPath, args...)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			Eventually(session).Should(gbytes.Say("dev"))
			Eventually(session).Should(gexec.Exit(0))
		})
	})
})

func getDirOfCurrentFile() string {
	_, filename, _, _ := runtime.Caller(1)
	return path.Dir(filename)
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
