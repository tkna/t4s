package constants

import (
	"os"
)

const (
	// Default path to the built-in mino yaml.
	DefaultMinoConf = "conf/minoes.yaml"

	// Name of the board.
	BoardName = "board"
)

var (
	AppImage = os.Getenv("app_image")
	MinoConf = os.Getenv("mino_conf")
)
