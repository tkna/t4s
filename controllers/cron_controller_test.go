package controllers

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	t4sv1 "github.com/tkna/t4s/api/v1"
)

var _ = Describe("Cron controller", func() {
	ctx := context.Background()
	var stopFunc func()
	var reconciler *CronReconciler

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme,
			LeaderElection:     false,
			MetricsBindAddress: "0",
		})
		Expect(err).ShouldNot(HaveOccurred())

		reconciler = &CronReconciler{
			Client: mgr.GetClient(),
			Scheme: scheme,
		}
		err = reconciler.SetupWithManager(mgr)
		Expect(err).ShouldNot(HaveOccurred())

		ctx, cancel := context.WithCancel(ctx)
		stopFunc = cancel
		go func() {
			err := mgr.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		stopFunc()
		time.Sleep(100 * time.Millisecond)
	})

	It("should initialize the Cron and create Actions", func() {
		By("creating a namespace and a Cron")
		nsName := "test-ns-cron-init"
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: nsName,
			},
		}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		cron := &t4sv1.Cron{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: nsName,
				Name:      "cron",
			},
			Spec: t4sv1.CronSpec{
				Period: 1000,
			},
		}
		err = k8sClient.Create(ctx, cron)
		Expect(err).ShouldNot(HaveOccurred())

		By("checking Action will be created")
		actions := &t4sv1.ActionList{}
		Eventually(func() error {
			if err := k8sClient.List(ctx, actions, &client.ListOptions{Namespace: nsName}); err != nil {
				return err
			}
			if len(actions.Items) == 0 {
				return fmt.Errorf("number of actions is 0")
			}
			if actions.Items[0].Spec.Op != "down" {
				return fmt.Errorf("actions.Items[0].Spec.Op != 'down'")
			}
			return nil
		}).Should(Succeed())

		By("checking Action will be created continually")
		time.Sleep(time.Second * 2)
		Eventually(func() error {
			if err := k8sClient.List(ctx, actions, &client.ListOptions{Namespace: nsName}); err != nil {
				return err
			}
			if len(actions.Items) < 2 {
				return fmt.Errorf("number of actions is not increasing")
			}
			if actions.Items[1].Spec.Op != "down" {
				return fmt.Errorf("actions.Items[1].Spec.Op != 'down'")
			}
			return nil
		}).Should(Succeed())

		By("Deleting Cron")
		err = k8sClient.Delete(ctx, cron)
		Expect(err).ShouldNot(HaveOccurred())
	})
})
