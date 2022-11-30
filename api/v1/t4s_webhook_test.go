package v1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Webhook Test", func() {
	ctx := context.Background()

	It("should not create 2 T4s in one namespace", func() {
		By("creating the first T4s")
		t1 := &T4s{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "t4s-1",
			},
			Spec: T4sSpec{
				Width:    10,
				Height:   20,
				Wait:     1000,
				NodePort: 30080,
			},
		}
		err := k8sClient.Create(ctx, t1)
		Expect(err).ShouldNot(HaveOccurred())

		By("creating the second T4s")
		t2 := &T4s{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "default",
				Name:      "t4s-2",
			},
			Spec: T4sSpec{
				Width:    10,
				Height:   20,
				Wait:     1000,
				NodePort: 30080,
			},
		}
		err = k8sClient.Create(ctx, t2)
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).Should(ContainSubstring("T4s is not allowed to be created more than 2 in one namespace"))
	})
})
