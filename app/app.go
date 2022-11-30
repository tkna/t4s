package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	t4sv1 "github.com/tkna/t4s/api/v1"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type Board struct {
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Data   [][]int `json:"data"`
}

type Action struct {
	Op string `json:"op"`
}

var (
	Cli         client.Client
	Namespace   string
	T4sName     string
	T4sUID      types.UID
	BoardName   string
	BoardUID    types.UID
	TargetT4s   *t4sv1.T4s
	TargetBoard *t4sv1.Board
)

func init() {
	var err error
	Cli, err = getClient()
	if err != nil {
		log.Fatal(err)
	}
	Namespace = os.Getenv("NAMESPACE")
	T4sName = os.Getenv("T4S_NAME")
	BoardName = os.Getenv("BOARD_NAME")

	ctx := context.Background()
	TargetT4s = &t4sv1.T4s{}
	TargetBoard = &t4sv1.Board{}
	if err := Cli.Get(ctx, client.ObjectKey{Namespace: Namespace, Name: T4sName}, TargetT4s); err != nil {
		log.Fatal(err)
	}
	T4sUID = TargetT4s.GetUID()
	err = Cli.Get(ctx, client.ObjectKey{Namespace: Namespace, Name: BoardName}, TargetBoard)
	if errors.IsNotFound(err) {
		log.Println("Board not found")
		return
	}
	if err != nil {
		log.Fatal(err)
	}
	BoardUID = TargetBoard.GetUID()
}

func main() {
	e := echo.New()
	e.Static("/", "static")
	e.GET("/board", getBoard)
	e.POST("/board", newBoard)
	e.GET("/colors", getColors)
	e.GET("/wait", getWait)
	e.POST("/actions", postAction, middleware.RateLimiter(
		middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      10,
				Burst:     1,
				ExpiresIn: 200 * time.Millisecond,
			},
		),
	))
	e.Debug = true
	e.Logger.Debug(e.Start(":8000"))
}

func getBoard(c echo.Context) error {
	log.Println("getBoard")
	ctx := context.Background()
	err := Cli.Get(ctx, client.ObjectKey{Namespace: Namespace, Name: BoardName}, TargetBoard)
	if errors.IsNotFound(err) {
		log.Println("Board not found")
		return c.NoContent(http.StatusOK)
	}
	if err != nil {
		log.Println(err)
		return err
	}

	b := &Board{}
	b.Width = TargetBoard.Spec.Width
	b.Height = TargetBoard.Spec.Height
	b.Data = TargetBoard.Status.Data
	if len(TargetBoard.Status.CurrentMino) != 0 {
		for _, coord := range TargetBoard.Status.CurrentMino[0].AbsoluteCoords {
			b.Data[coord.Y][coord.X] = TargetBoard.Status.CurrentMino[0].MinoID
		}
	}
	return c.JSON(http.StatusOK, b)
}

func newBoard(c echo.Context) error {
	log.Println("newBoard")
	ctx := context.Background()

	// Fetch latest T4s
	if err := Cli.Get(ctx, client.ObjectKey{Namespace: Namespace, Name: T4sName}, TargetT4s); err != nil {
		log.Println(err)
		return err
	}
	if !TargetT4s.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Println("T4s is being deleted")
		return nil
	}
	T4sUID = TargetT4s.GetUID()

	// Delete the existing board
	if !TargetBoard.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Println("Board is being deleted")
		return nil
	}
	err := Cli.Delete(ctx, TargetBoard)
	if errors.IsNotFound(err) {
		log.Println("Board not found")
	} else if err != nil {
		log.Println(err)
		return err
	}

	// Create a new board
	TargetBoard = &t4sv1.Board{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: Namespace,
			Name:      BoardName,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "t4s.tkna.net/v1",
					Kind:               "T4s",
					Name:               T4sName,
					UID:                T4sUID,
					Controller:         pointer.Bool(true),
					BlockOwnerDeletion: pointer.Bool(true),
				},
			},
		},
		Spec: t4sv1.BoardSpec{
			Width:  TargetT4s.Spec.Width,
			Height: TargetT4s.Spec.Height,
			Wait:   TargetT4s.Spec.Wait,
			State:  t4sv1.Playing,
		},
	}
	if err := Cli.Create(ctx, TargetBoard); err != nil {
		log.Println(err)
		return err
	}
	BoardUID = TargetBoard.GetUID()

	log.Println("new board created")
	return c.NoContent(http.StatusOK)
}

func postAction(c echo.Context) error {
	log.Println("postAction")
	action := new(Action)
	if err := c.Bind(action); err != nil {
		log.Println(err)
		return err
	}

	ctx := context.Background()
	ac := t4sv1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    Namespace,
			GenerateName: "action-",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "t4s.tkna.net/v1",
					Kind:               "Board",
					Name:               BoardName,
					UID:                BoardUID,
					Controller:         pointer.Bool(true),
					BlockOwnerDeletion: pointer.Bool(true),
				},
			},
		},
		Spec: t4sv1.ActionSpec{
			Op: action.Op,
		},
	}
	if err := Cli.Create(ctx, &ac); err != nil {
		log.Println(err)
		return err
	}
	return c.JSON(http.StatusOK, action)
}

func getColors(c echo.Context) error {
	log.Println("getColors")
	ctx := context.Background()
	minoList := t4sv1.MinoList{}
	if err := Cli.List(ctx, &minoList, &client.ListOptions{Namespace: Namespace}); err != nil {
		log.Println(err)
		return err
	}

	cls := make([][]interface{}, len(minoList.Items))
	for i, mino := range minoList.Items {
		var cl = make([]interface{}, 2)
		cl[0] = mino.Spec.MinoID
		cl[1] = mino.Spec.Color
		cls[i] = cl
	}

	return c.JSON(http.StatusOK, cls)
}

func getWait(c echo.Context) error {
	log.Println("getWait")
	ctx := context.Background()
	// Fetch latest T4s
	if err := Cli.Get(ctx, client.ObjectKey{Namespace: Namespace, Name: T4sName}, TargetT4s); err != nil {
		log.Println(err)
		return err
	}
	T4sUID = TargetT4s.GetUID()
	return c.JSON(http.StatusOK, TargetT4s.Spec.Wait)
}

func getClient() (client.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	scm := runtime.NewScheme()
	if err := t4sv1.AddToScheme(scm); err != nil {
		return nil, err
	}
	cli, err := client.New(cfg, client.Options{Scheme: scm})
	if err != nil {
		return nil, err
	}
	return cli, nil
}
