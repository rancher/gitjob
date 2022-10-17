package e2e_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rancher/gitjob/e2e/testenv"
	"github.com/rancher/gitjob/e2e/testenv/kubectl"
)

var _ = Describe("Gitjob Examples", func() {
	var (
		asset string
		// deployment name should be different for each test case to
		// avoid conflicts. It matches the name used in the manifest
		// found in the git repo.
		deployment string
		k          kubectl.Command
	)

	BeforeEach(func() {
		k = env.Kubectl
	})

	JustBeforeEach(func() {
		out, err := k.Apply("-f", testenv.AssetPath(asset))
		Expect(err).ToNot(HaveOccurred(), out)
	})

	AfterEach(func() {
		out, err := k.Delete("-f", testenv.AssetPath(asset))
		Expect(err).ToNot(HaveOccurred(), out)

		if deployment != "" {
			out, err = k.Delete("deployment", deployment)
			Expect(err).ToNot(HaveOccurred(), out)
		}
	})

	When("referencing a github repo via https", func() {
		When("creating a gitjob resource", func() {
			BeforeEach(func() {
				asset = "gitjob.yaml"
				deployment = "nginx-deployment"
			})

			It("creates the deployment", func() {
				Eventually(func() string {
					out, _ := k.Get("pods")
					return out
				}, testenv.Timeout).Should(ContainSubstring(deployment))
			})
		})
	})

	When("referencing a private github repo via ssh", func() {
		BeforeEach(func() {
			keyPath := os.Getenv("GIT_SSH_KEY")
			out, err := k.Create(
				"secret", "generic", "ssh-key-secret", "--type", "kubernetes.io/ssh-auth",
				"--from-file=ssh-privatekey="+keyPath,
			)
			Expect(err).ToNot(HaveOccurred(), out)
		})

		AfterEach(func() {
			out, err := k.Delete("secret", "ssh-key-secret")
			Expect(err).ToNot(HaveOccurred(), out)
		})

		When("creating a gitjob resource", func() {
			BeforeEach(func() {
				asset = "gitjob-private-repo.yaml"
				deployment = "private-nginx"
			})

			It("creates the deployment", func() {
				Eventually(func() string {
					out, _ := k.Get("pods")
					return out
				}, testenv.Timeout).Should(ContainSubstring(deployment))
			})
		})
	})

	When("referencing a private github repo via http basic auth", func() {
		BeforeEach(func() {
			user := os.Getenv("BASIC_AUTH_USER")
			pat := os.Getenv("BASIC_AUTH_PAT")
			out, err := k.Create(
				"secret", "generic",
				"basic-auth-secret",
				"--type", "kubernetes.io/basic-auth",
				"--from-literal=username="+user,
				"--from-literal=password="+pat,
			)
			Expect(err).ToNot(HaveOccurred(), out)
		})

		AfterEach(func() {
			out, err := k.Delete("secret", "basic-auth-secret")
			Expect(err).ToNot(HaveOccurred(), out)
		})

		When("creating a gitjob resource", func() {
			BeforeEach(func() {
				asset = "gitjob-private-http-repo.yaml"
				deployment = ""
			})

			It("runs the kubectl command", func() {
				Eventually(func() string {
					out, _ := k.Get("configmap")
					return out
				}, testenv.Timeout).Should(ContainSubstring("private-http-test"))
			})
		})
	})
})
