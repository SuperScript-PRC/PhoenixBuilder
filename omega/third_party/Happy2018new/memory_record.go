package Happy2018new

import (
	GameInterface "phoenixbuilder/game_control/game_interface"
	"phoenixbuilder/omega/defines"
)

type MemoryRecord struct {
	*defines.BasicComponent
	apis GameInterface.GameInterface
}
