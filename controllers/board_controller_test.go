package controllers

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	t4sv1 "github.com/tkna/t4s/api/v1"
	"github.com/tkna/t4s/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Board controller", func() {
	ctx := context.Background()
	var stopFunc func()
	var reconciler *BoardReconciler

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme:             scheme,
			LeaderElection:     false,
			MetricsBindAddress: "0",
		})
		Expect(err).ShouldNot(HaveOccurred())

		reconciler = &BoardReconciler{
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

	It("should initialize the Board with State == GameOver", func() {
		By("creating a namespace, a Mino, and a Board")
		nsName := "test-ns-board-init-gameover"
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: nsName,
			},
		}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		mino := &t4sv1.Mino{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: nsName,
				Name:      "mino-i",
			},
			Spec: t4sv1.MinoSpec{
				MinoID: 1,
				Coords: []t4sv1.Coord{
					{X: -1, Y: 0},
					{X: 0, Y: 0},
					{X: 1, Y: 0},
					{X: 2, Y: 0},
				},
				Color: "#a0d8ef",
			},
		}
		err = k8sClient.Create(ctx, mino)
		Expect(err).ShouldNot(HaveOccurred())

		board := &t4sv1.Board{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: nsName,
				Name:      constants.BoardName,
			},
			Spec: t4sv1.BoardSpec{
				Width:  10,
				Height: 20,
				Wait:   1000,
				State:  t4sv1.GameOver,
			},
		}
		err = k8sClient.Create(ctx, board)
		Expect(err).ShouldNot(HaveOccurred())

		By("checking BoardStatus will be updated")
		Eventually(func() error {
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: constants.BoardName}, board); err != nil {
				return err
			}
			if board.Status.Data == nil {
				return errors.New("board.Status.Data is nil")
			}
			if board.Status.State != t4sv1.GameOver {
				return errors.New("board.Status.State != t4sv1.GameOver")
			}
			return nil
		}).Should(Succeed())

		By("checking Cron will NOT be created")
		cron := &t4sv1.Cron{}
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: "cron"}, cron)
		}).ShouldNot(Succeed())
	})

	It("should initialize the Board with State == Playing, set a new current Mino, and create a Cron", func() {
		By("creating a namespace, a Mino, and a Board")
		nsName := "test-ns-board-init-playing"
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: nsName,
			},
		}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		mino := &t4sv1.Mino{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: nsName,
				Name:      "mino-i",
			},
			Spec: t4sv1.MinoSpec{
				MinoID: 1,
				Coords: []t4sv1.Coord{
					{X: -1, Y: 0},
					{X: 0, Y: 0},
					{X: 1, Y: 0},
					{X: 2, Y: 0},
				},
				Color: "#a0d8ef",
			},
		}
		err = k8sClient.Create(ctx, mino)
		Expect(err).ShouldNot(HaveOccurred())

		board := &t4sv1.Board{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: nsName,
				Name:      constants.BoardName,
			},
			Spec: t4sv1.BoardSpec{
				Width:  10,
				Height: 20,
				Wait:   1000,
				State:  t4sv1.Playing,
			},
		}
		err = k8sClient.Create(ctx, board)
		Expect(err).ShouldNot(HaveOccurred())

		By("checking BoardStatus will be updated")
		Eventually(func() error {
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: constants.BoardName}, board); err != nil {
				return err
			}
			if board.Status.Data == nil {
				return errors.New("board.Status.Data is nil")
			}
			if len(board.Status.CurrentMino) != 1 {
				return errors.New("len(board.Status.CurrentMino) != 1")
			}
			if board.Status.CurrentMino[0].MinoID != 1 {
				return errors.New("board.Status.CurrentMino[0].MinoID != 1")
			}
			if board.Status.State != t4sv1.Playing {
				return errors.New("board.Status.State != t4sv1.Playing")
			}
			return nil
		}).Should(Succeed())

		By("checking Cron will be created")
		cron := &t4sv1.Cron{}
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: "cron"}, cron)
		}).Should(Succeed())
		Expect(cron.Spec).To(MatchFields(IgnoreExtras, Fields{
			"Period": Equal(1000),
		}))
	})

	It("should move the current mino down successfully", func() {
		By("creating a namespace and a Mino")
		nsName := "test-ns-board-down"
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: nsName,
			},
		}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		mino := &t4sv1.Mino{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: nsName,
				Name:      "mino-i",
			},
			Spec: t4sv1.MinoSpec{
				MinoID: 1,
				Coords: []t4sv1.Coord{
					{X: -1, Y: 0},
					{X: 0, Y: 0},
					{X: 1, Y: 0},
					{X: 2, Y: 0},
				},
				Color: "#a0d8ef",
			},
		}
		err = k8sClient.Create(ctx, mino)
		Expect(err).ShouldNot(HaveOccurred())

		By("creating a Board")
		board := &t4sv1.Board{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: nsName,
				Name:      constants.BoardName,
			},
			Spec: t4sv1.BoardSpec{
				Width:  10,
				Height: 20,
				Wait:   1000,
				State:  t4sv1.Playing,
			},
		}
		err = k8sClient.Create(ctx, board)
		Expect(err).ShouldNot(HaveOccurred())
		// Wait for Board to be fully reconciled.
		Eventually(func() error {
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: constants.BoardName}, board); err != nil {
				return err
			}
			if len(board.Status.CurrentMino) != 1 {
				return fmt.Errorf("len(board.Status.CurrentMino) is not 1")
			}
			return nil
		}).Should(Succeed())

		By("creating an Action")
		action := &t4sv1.Action{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: nsName,
				Name:      "action-1",
			},
			Spec: t4sv1.ActionSpec{
				Op: "down",
			},
		}
		err = k8sClient.Create(ctx, action)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: "action-1"}, action)
		}).Should(Succeed())

		By("updating the status of the Board")
		board.Status.Data = make([][]int, board.Spec.Height)
		for i := 0; i < board.Spec.Height; i++ {
			board.Status.Data[i] = make([]int, board.Spec.Width)
		}
		board.Status.State = t4sv1.Playing
		board.Status.CurrentMino = []t4sv1.CurrentMino{
			{
				MinoID: 2,
				Center: t4sv1.Coord{X: 3, Y: 5},
				RelativeCoords: []t4sv1.Coord{
					{X: -1, Y: 0},
					{X: 0, Y: 0},
					{X: 1, Y: 0},
					{X: 2, Y: 0},
				},
				AbsoluteCoords: []t4sv1.Coord{
					{X: 2, Y: 5},
					{X: 3, Y: 5},
					{X: 4, Y: 5},
					{X: 5, Y: 5},
				},
			},
		}
		err = k8sClient.Status().Update(ctx, board)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() error {
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: constants.BoardName}, board); err != nil {
				return err
			}
			if len(board.Status.CurrentMino) != 1 {
				return fmt.Errorf("len(board.Status.CurrentMino) is not 1")
			}
			if board.Status.CurrentMino[0].MinoID != 2 {
				return fmt.Errorf("board.Status.CurrentMino is not 2")
			}
			return nil
		}).Should(Succeed())

		By("triggering reconciliation")
		_, err = reconciler.Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: nsName,
				Name:      constants.BoardName,
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		By("checking the current mino will move down successfully")
		Eventually(func() error {
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: constants.BoardName}, board); err != nil {
				return err
			}
			if len(board.Status.CurrentMino) != 1 {
				return errors.New("len(board.Status.CurrentMino) is not 1")
			}
			if !reflect.DeepEqual(board.Status.CurrentMino[0].Center, t4sv1.Coord{X: 3, Y: 6}) {
				return fmt.Errorf("board.Status.CurrentMino[0].Center doesn't have the expected value %v, got %v", t4sv1.Coord{X: 3, Y: 6}, board.Status.CurrentMino[0].Center)
			}
			expected := []t4sv1.Coord{
				{X: 2, Y: 6},
				{X: 3, Y: 6},
				{X: 4, Y: 6},
				{X: 5, Y: 6},
			}
			if !reflect.DeepEqual(board.Status.CurrentMino[0].AbsoluteCoords, expected) {
				return fmt.Errorf("board.Status.CurrentMino[0].AbsoluteCoords doesn't have the expected value %v, got %v", expected, board.Status.CurrentMino[0].AbsoluteCoords)
			}
			return nil
		}).Should(Succeed())
	})

	It("should remove a row successfully", func() {
		By("creating a namespace and a Mino")
		nsName := "test-ns-board-remove-row"
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: nsName,
			},
		}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		mino := &t4sv1.Mino{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: nsName,
				Name:      "mino-i",
			},
			Spec: t4sv1.MinoSpec{
				MinoID: 1,
				Coords: []t4sv1.Coord{
					{X: -1, Y: 0},
					{X: 0, Y: 0},
					{X: 1, Y: 0},
					{X: 2, Y: 0},
				},
				Color: "#a0d8ef",
			},
		}
		err = k8sClient.Create(ctx, mino)
		Expect(err).ShouldNot(HaveOccurred())

		By("creating a Board")
		board := &t4sv1.Board{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: nsName,
				Name:      constants.BoardName,
			},
			Spec: t4sv1.BoardSpec{
				Width:  6,
				Height: 3,
				Wait:   1000,
				State:  t4sv1.Playing,
			},
		}
		err = k8sClient.Create(ctx, board)
		Expect(err).ShouldNot(HaveOccurred())
		// Wait for Board to be fully reconciled.
		Eventually(func() error {
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: constants.BoardName}, board); err != nil {
				return err
			}
			if len(board.Status.CurrentMino) != 1 {
				return fmt.Errorf("len(board.Status.CurrentMino) is not 1")
			}
			return nil
		}).Should(Succeed())

		By("creating an Action")
		action := &t4sv1.Action{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: nsName,
				Name:      "action-1",
			},
			Spec: t4sv1.ActionSpec{
				Op: "down",
			},
		}
		err = k8sClient.Create(ctx, action)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: "action-1"}, action)
		}).Should(Succeed())

		By("updating the status of the board")
		board.Status.Data = [][]int{
			{0, 0, 0, 0, 0, 0},
			{1, 0, 0, 0, 0, 1},
			{1, 0, 0, 0, 0, 1},
		}
		board.Status.State = t4sv1.Playing
		board.Status.CurrentMino = []t4sv1.CurrentMino{
			{
				MinoID: 2,
				Center: t4sv1.Coord{X: 2, Y: 2},
				RelativeCoords: []t4sv1.Coord{
					{X: -1, Y: 0},
					{X: 0, Y: 0},
					{X: 1, Y: 0},
					{X: 2, Y: 0},
				},
				AbsoluteCoords: []t4sv1.Coord{
					{X: 1, Y: 2},
					{X: 2, Y: 2},
					{X: 3, Y: 2},
					{X: 4, Y: 2},
				},
			},
		}
		err = k8sClient.Status().Update(ctx, board)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() error {
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: constants.BoardName}, board); err != nil {
				return err
			}
			if len(board.Status.CurrentMino) != 1 {
				return fmt.Errorf("len(board.Status.CurrentMino) is not 1")
			}
			if board.Status.CurrentMino[0].MinoID != 2 {
				return fmt.Errorf("board.Status.CurrentMino is not 2")
			}
			return nil
		}).Should(Succeed())

		By("triggering reconciliation")
		_, err = reconciler.Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: nsName,
				Name:      constants.BoardName,
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		By("checking the current mino will move down successfully")
		Eventually(func() error {
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: constants.BoardName}, board); err != nil {
				return err
			}
			if board.Status.CurrentMino != nil {
				return errors.New("board.Status.CurrentMino should be nil")
			}
			expected := [][]int{
				{0, 0, 0, 0, 0, 0},
				{0, 0, 0, 0, 0, 0},
				{1, 0, 0, 0, 0, 1},
			}
			if !reflect.DeepEqual(board.Status.Data, expected) {
				return fmt.Errorf("board.Status.Data doesn't have the expected value %v, got %v", expected, board.Status.Data)
			}
			return nil
		}).Should(Succeed())
	})

	It("should change State to GameOver and delete Cron", func() {
		By("creating a namespace and a Mino")
		nsName := "test-ns-board-gameover"
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: nsName,
			},
		}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		mino := &t4sv1.Mino{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: nsName,
				Name:      "mino-i",
			},
			Spec: t4sv1.MinoSpec{
				MinoID: 1,
				Coords: []t4sv1.Coord{
					{X: -1, Y: 0},
					{X: 0, Y: 0},
					{X: 1, Y: 0},
					{X: 2, Y: 0},
				},
				Color: "#a0d8ef",
			},
		}
		err = k8sClient.Create(ctx, mino)
		Expect(err).ShouldNot(HaveOccurred())

		By("creating a Board")
		board := &t4sv1.Board{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: nsName,
				Name:      constants.BoardName,
			},
			Spec: t4sv1.BoardSpec{
				Width:  6,
				Height: 3,
				Wait:   1000,
				State:  t4sv1.Playing,
			},
		}
		err = k8sClient.Create(ctx, board)
		Expect(err).ShouldNot(HaveOccurred())
		// Wait for Board to be fully reconciled.
		Eventually(func() error {
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: constants.BoardName}, board); err != nil {
				return err
			}
			if len(board.Status.CurrentMino) != 1 {
				return fmt.Errorf("len(board.Status.CurrentMino) is not 1")
			}
			return nil
		}).Should(Succeed())

		By("checking Cron is created")
		cron := &t4sv1.Cron{}
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: "cron"}, cron)
		}).Should(Succeed())

		By("updating the status of the board")
		board.Status.Data = [][]int{
			{0, 0, 0, 0, 0, 0},
			{1, 0, 1, 0, 0, 1},
			{1, 0, 1, 0, 0, 1},
		}
		board.Status.State = t4sv1.Playing
		board.Status.CurrentMino = nil

		err = k8sClient.Status().Update(ctx, board)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(func() error {
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: constants.BoardName}, board); err != nil {
				return err
			}
			if board.Status.CurrentMino != nil {
				return fmt.Errorf("board.Status.CurrentMino should be nil")
			}
			return nil
		}).Should(Succeed())

		By("triggering reconciliation")
		_, err = reconciler.Reconcile(ctx, ctrl.Request{
			NamespacedName: types.NamespacedName{
				Namespace: nsName,
				Name:      constants.BoardName,
			},
		})
		Expect(err).ShouldNot(HaveOccurred())

		By("checking BoardStatus will be GameOver")
		Eventually(func() error {
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: constants.BoardName}, board); err != nil {
				return err
			}
			if board.Status.CurrentMino != nil {
				return errors.New("board.Status.CurrentMino should be nil")
			}
			if board.Status.State != t4sv1.GameOver {
				return errors.New("board.Status.State should be t4sv1.GameOver")
			}
			return nil
		}).Should(Succeed())

		By("checking Cron will be deleted")
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: nsName, Name: "cron"}, cron)
		}).ShouldNot(Succeed())
	})
})
