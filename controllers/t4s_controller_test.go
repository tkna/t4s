package controllers

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	t4sv1 "github.com/tkna/t4s/api/v1"
	"github.com/tkna/t4s/pkg/constants"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("T4s controller", func() {
	ctx := context.Background()
	var stopFunc func()

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme,
			LeaderElection:     false,
			MetricsBindAddress: "0",
		})
		Expect(err).ShouldNot(HaveOccurred())

		reconciler := &T4sReconciler{
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

	When("the type of the front service is NodePort", func() {
		It("should create resources related to a T4s resource", func() {
			By("creating a namespace and T4s")
			nsName := "test-ns-np"
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
				},
			}
			err := k8sClient.Create(ctx, ns)
			Expect(err).NotTo(HaveOccurred())

			t4s := &t4sv1.T4s{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns-np",
					Name:      "test",
				},
				Spec: t4sv1.T4sSpec{
					Width:       10,
					Height:      20,
					Wait:        1000,
					ServiceType: "NodePort",
					NodePort:    30080,
				},
			}

			err = k8sClient.Create(ctx, t4s)
			Expect(err).ShouldNot(HaveOccurred())

			By("checking Board will be created")
			board := &t4sv1.Board{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: constants.BoardName}, board)
			}).Should(Succeed())
			Expect(board.Spec).To(MatchFields(IgnoreExtras, Fields{
				"Width":  Equal(t4s.Spec.Width),
				"Height": Equal(t4s.Spec.Height),
				"Wait":   Equal(t4s.Spec.Wait),
				"State":  Equal(t4sv1.GameOver),
			}))

			By("checking the deployment app will be created")
			dep := &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: "t4s-app"}, dep)
			}).Should(Succeed())
			Expect(dep.Labels).To(MatchAllKeys(Keys{
				"tier": Equal("app"),
			}))
			Expect(dep.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(dep.Spec.Template.Spec.Containers[0]).To(MatchFields(IgnoreExtras, Fields{
				"Name": Equal(constants.BoardName),
				"Env": ConsistOf([]corev1.EnvVar{
					{
						Name:  "NAMESPACE",
						Value: nsName,
					},
					{
						Name:  "T4S_NAME",
						Value: "test",
					},
					{
						Name:  "BOARD_NAME",
						Value: constants.BoardName,
					},
				}),
			}))

			By("checking a service for app will be created")
			svc := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: "t4s-app"}, svc)
			}).Should(Succeed())
			Expect(svc.Labels).To(MatchAllKeys(Keys{
				"tier": Equal("app"),
			}))
			Expect(svc.Spec).To(MatchFields(IgnoreExtras, Fields{
				"Type": Equal(corev1.ServiceTypeNodePort),
				"Ports": ConsistOf([]corev1.ServicePort{
					{
						Protocol:   corev1.ProtocolTCP,
						Port:       8000,
						TargetPort: intstr.FromInt(8000),
						NodePort:   30080,
					},
				}),
			}))

			By("checking Minoes will be created")
			minoes := &t4sv1.MinoList{}
			Eventually(func() error {
				return k8sClient.List(ctx, minoes, &client.ListOptions{Namespace: nsName})
			}).Should(Succeed())
			Expect(minoes.Items).To(HaveLen(7))
			Expect(minoes.Items[0].Name).To(Equal("mino-i"))
			Expect(minoes.Items[0].Spec).To(MatchFields(IgnoreExtras, Fields{
				"MinoID": Equal(1),
			}))
		})
	})

	When("the type of the front service is LoadBalancer", func() {
		It("should create a front service which has the type of LoadBalancer", func() {
			By("creating a namespace and T4s")
			nsName := "test-ns-lb"
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
				},
			}
			err := k8sClient.Create(ctx, ns)
			Expect(err).NotTo(HaveOccurred())

			t4s := &t4sv1.T4s{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: nsName,
					Name:      "test",
				},
				Spec: t4sv1.T4sSpec{
					Width:                    10,
					Height:                   20,
					Wait:                     1000,
					ServiceType:              "LoadBalancer",
					LoadBalancerIP:           "10.0.0.1",
					LoadBalancerSourceRanges: []string{"192.168.0.1/32"},
				},
			}

			err = k8sClient.Create(ctx, t4s)
			Expect(err).ShouldNot(HaveOccurred())

			By("checking a service for app will be created")
			svc := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: "t4s-app"}, svc)
			}).Should(Succeed())
			Expect(svc.Labels).To(MatchAllKeys(Keys{
				"tier": Equal("app"),
			}))
			Expect(svc.Spec).To(MatchFields(IgnoreExtras, Fields{
				"Type":                     Equal(corev1.ServiceTypeLoadBalancer),
				"LoadBalancerIP":           Equal("10.0.0.1"),
				"LoadBalancerSourceRanges": ConsistOf([]string{"192.168.0.1/32"}),
			}))
		})
	})

	It("should update or recreate the Board when T4s is updated", func() {
		By("creating a namespace and T4s")
		nsName := "test-ns-t4s-update"
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: nsName,
			},
		}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		t4s := &t4sv1.T4s{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: nsName,
				Name:      "test",
			},
			Spec: t4sv1.T4sSpec{
				Width:  10,
				Height: 20,
				Wait:   1000,
			},
		}
		err = k8sClient.Create(ctx, t4s)
		Expect(err).ShouldNot(HaveOccurred())

		By("checking Board will be created")
		board := &t4sv1.Board{}
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: constants.BoardName}, board)
		}).Should(Succeed())
		Expect(board.Spec).To(MatchFields(IgnoreExtras, Fields{
			"Width":  Equal(10),
			"Height": Equal(20),
			"Wait":   Equal(1000),
			"State":  Equal(t4sv1.GameOver),
		}))

		By("checking Board will be updated when Wait is updated")
		t4s.Spec.Wait = 500
		err = k8sClient.Update(ctx, t4s)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() error {
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: constants.BoardName}, board); err != nil {
				return err
			}
			if board.Spec.Width != 10 || board.Spec.Height != 20 || board.Spec.Wait != 500 {
				return errors.New("Board doesn't have the expected spec")
			}
			return nil
		}).Should(Succeed())

		By("checking Board will be recreated when Width or Height is updated")
		t4s.Spec.Width = 15
		t4s.Spec.Height = 10
		err = k8sClient.Update(ctx, t4s)
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(func() error {
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: constants.BoardName}, board); err != nil {
				return err
			}
			if board.Spec.Width != 15 || board.Spec.Height != 10 || board.Spec.Wait != 500 {
				return errors.New("Board doesn't have the expected spec")
			}
			return nil
		}).Should(Succeed())
	})
})
