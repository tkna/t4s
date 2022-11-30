package e2e

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	//go:embed manifests/t4s.yaml
	t4sYAML []byte
)

var _ = Describe("t4s", func() {
	It("should generate resources", func() {
		namespace := uuid.NewString()

		By("creating namespace")
		kubectlSafe(nil, "create", "ns", namespace)

		By("creating T4s")
		kubectlSafe(t4sYAML, "apply", "-n", namespace, "-f", "-")

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "t4s", "t4s-1")
			return err
		}).Should(Succeed())

		Eventually(func() error {
			out, err := kubectl(nil, "get", "-n", namespace, "deployment", "t4s-app")
			if err != nil {
				return err
			}
			if !bytes.Contains(out, []byte("1/1")) {
				return fmt.Errorf("Deployment t4s-app is not ready")
			}
			return nil
		}).Should(Succeed())

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "service", "t4s-app")
			return err
		}).Should(Succeed())
	})

	It("should delete resources", func() {
		namespace := uuid.NewString()

		By("creating namespace")
		kubectlSafe(nil, "create", "ns", namespace)

		By("creating T4s")
		kubectlSafe(t4sYAML, "apply", "-n", namespace, "-f", "-")

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "deployment", "t4s-app")
			return err
		}).Should(Succeed())

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "service", "t4s-app")
			return err
		}).Should(Succeed())

		By("deleting T4s")
		kubectlSafe(nil, "delete", "-n", namespace, "t4s", "t4s-1")

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "deployment", "t4s-app")
			return err
		}).ShouldNot(Succeed())

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "service", "t4s-app")
			return err
		}).ShouldNot(Succeed())
	})

	It("should send http request to the t4s-app server", func() {
		namespace := uuid.NewString()

		By("creating namespace")
		kubectlSafe(nil, "create", "ns", namespace)

		By("creating T4s")
		kubectlSafe(t4sYAML, "apply", "-n", namespace, "-f", "-")

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "t4s", "t4s-1")
			return err
		}).Should(Succeed())

		Eventually(func() error {
			out, err := kubectl(nil, "get", "-n", namespace, "deployment", "t4s-app")
			if err != nil {
				return err
			}
			if !bytes.Contains(out, []byte("1/1")) {
				return fmt.Errorf("Deployment t4s-app is not ready")
			}
			return nil
		}).Should(Succeed())

		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "service", "t4s-app")
			return err
		}).Should(Succeed())

		By("creating alpine pod")
		Eventually(func() error {
			_, err := kubectl(nil, "run", "-n", namespace, "alpine", "--restart=Never", "--image=alpine", "--", "sleep", "60")
			return err
		}).Should(Succeed())

		By("installing curl")
		Eventually(func() error {
			_, err := kubectl(nil, "exec", "-n", namespace, "alpine", "--", "apk", "--update", "add", "curl")
			return err
		}).Should(Succeed())

		By("sending http POST request to http://t4s-app:8000/board")
		Eventually(func() error {
			resp, err := kubectl(nil, "exec", "-n", namespace, "alpine", "--", "curl", "-i", "-X", "POST", "t4s-app:8000/board")
			if err != nil {
				return err
			}
			if !bytes.Contains(resp, []byte("200 OK")) {
				return fmt.Errorf("failed to post http://t4s-app:8000/board")
			}
			return nil
		}).Should(Succeed())

		By("checking the board")
		Eventually(func() error {
			_, err := kubectl(nil, "get", "-n", namespace, "board", "board")
			return err
		}).Should(Succeed())

		By("sending http GET request to http://t4s-app:8000/")
		Eventually(func() error {
			resp, err := kubectl(nil, "exec", "-n", namespace, "alpine", "--", "curl", "-i", "t4s-app:8000")
			if err != nil {
				return err
			}
			if !bytes.Contains(resp, []byte("200 OK")) {
				return fmt.Errorf("failed to get http://t4s-app:8000/")
			}
			return nil
		}).Should(Succeed())

		By("sending http GET request to http://t4s-app:8000/board")
		Eventually(func() error {
			resp, err := kubectl(nil, "exec", "-n", namespace, "alpine", "--", "curl", "-i", "t4s-app:8000/board")
			if err != nil {
				return err
			}
			if !bytes.Contains(resp, []byte("200 OK")) {
				return fmt.Errorf("failed to get http://t4s-app:8000/board")
			}
			return nil
		}).Should(Succeed())

		By("sending http GET request to http://t4s-app:8000/colors")
		Eventually(func() error {
			resp, err := kubectl(nil, "exec", "-n", namespace, "alpine", "--", "curl", "-i", "t4s-app:8000/colors")
			if err != nil {
				return err
			}
			if !bytes.Contains(resp, []byte("200 OK")) {
				return fmt.Errorf("failed to get http://t4s-app:8000/colors")
			}
			return nil
		}).Should(Succeed())

		By("sending http GET request to http://t4s-app:8000/wait")
		Eventually(func() error {
			resp, err := kubectl(nil, "exec", "-n", namespace, "alpine", "--", "curl", "-i", "t4s-app:8000/wait")
			if err != nil {
				return err
			}
			if !bytes.Contains(resp, []byte("200 OK")) {
				return fmt.Errorf("failed to get http://t4s-app:8000/wait")
			}
			return nil
		}).Should(Succeed())

		By("sending http POST request to http://t4s-app:8000/actions")
		Eventually(func() error {
			resp, err := kubectl(nil, "exec", "-n", namespace, "alpine", "--", "curl", "-i", "-X", "POST", "-H", `Content-Type:application/json`,
				"-d", `{"op": "left"}`, "t4s-app:8000/actions")
			if err != nil {
				return err
			}
			if !bytes.Contains(resp, []byte("200 OK")) {
				return fmt.Errorf("failed to post http://t4s-app:8000/actions")
			}
			return nil
		}).Should(Succeed())
	})
})
