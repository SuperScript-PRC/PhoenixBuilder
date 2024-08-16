package Happy2018new_depends

import "sync"

// ------------------------- general -------------------------

// 一个简单化的实现，
// 用于向其他线程通知状态更改
type StateChange struct {
	// 一个集合，
	// 其中每一个管道都对应一个线程。
	// 当发生状态更改，
	// 每一个管道都将收到信息
	signal []chan struct{}
	// 互斥锁，
	// 用于防止对 signal 的并发读写
	lock sync.Mutex
}

// ------------------------- player position -------------------------

// 描述玩家的游戏名称
type PlayerName string

// 描述多个玩家的坐标和朝向信息
type PlayersPosInfo map[PlayerName]PosInfo

// 描述单个玩家的坐标和朝向信息
type PosInfo struct {
	Dimension byte       // 玩家所处的维度
	Position  [3]float32 // 玩家的位置
	YRot      float32    // 玩家的偏航角
}
