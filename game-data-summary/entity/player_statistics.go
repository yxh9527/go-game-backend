package entity

type Statistics struct {
	UserId    int32 `db:"uid" json:"uid"`               //玩家id
	GameId    int32 `db:"gid" json:"gid"`               //游戏id
	AreaId    int32 `db:"aid" json:"aid"`               //房间难度id
	Times     int32 `db:"times" json:"times"`           //本级别手数
	BetTimes  int32 `db:"bet_times" json:"bet_times"`   //下注入池手数
	IncrTimes int32 `db:"incr_times" json:"incr_times"` //加注入池手数
}
