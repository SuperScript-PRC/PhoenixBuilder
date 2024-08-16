package Happy2018new_depends

import (
	"phoenixbuilder/omega/defines"
	"sync"
)

// 从 frame 获取玩家坐标及朝向信息
func GetPlayerPositionContext(frame defines.MainFrame) (
	table PlayersPosInfo,
	lock *sync.RWMutex,
	regist func() <-chan struct{},
	has bool,
) {
	table_origin, has := frame.GetContext("global::storage::player_pos_table")
	if !has {
		return
	}
	table = table_origin.(PlayersPosInfo)
	// player_pos_info
	lock_origin, has := frame.GetContext("global::sync_mutex::player_pos_table")
	if !has {
		return
	}
	lock = lock_origin.(*sync.RWMutex)
	// lock
	regist_origin, has := frame.GetContext("global::regist_state_change::player_pos_table")
	if !has {
		return
	}
	regist = regist_origin.(func() <-chan struct{})
	// regist
	return
	// return
}
