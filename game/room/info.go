package room

import "github.com/kercylan98/minotaur/game"

type Info[PlayerID comparable, P game.Player[PlayerID], R Room] struct {
	room        R
	playerLimit int       // 玩家人数上限, <= 0 表示无限制
	owner       *PlayerID // 房主
	seat        *Seat[PlayerID, P, R]
}
