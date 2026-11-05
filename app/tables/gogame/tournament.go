package gogame

// MatchSchedule stores an explicitly zoned daily or one-time tournament template.
type MatchSchedule struct {
	Id           int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	AgentId      int64  `gorm:"column:agent_id;index;not null;" json:"agent_id"`
	ScheduleKind string `gorm:"column:schedule_kind;size:16;not null;" json:"schedule_kind"`
	Timezone     string `gorm:"column:timezone;size:64;not null;" json:"timezone"`
	DailyTime    string `gorm:"column:daily_time;size:8;" json:"daily_time"`
	StartTime    int64  `gorm:"column:start_time;default:0;" json:"start_time"`
	Enabled      int32  `gorm:"column:enabled;default:1;" json:"enabled"`
	Payload      string `gorm:"column:payload;type:longtext;not null;" json:"payload"`
}

func (*MatchSchedule) TableName() string { return "gg_match_schedule" }

// MatchOccurrence deduplicates a template occurrence across match instances.
type MatchOccurrence struct {
	OccurrenceKey string `gorm:"column:occurrence_key;size:128;primaryKey;" json:"occurrence_key"`
	ScheduleId    int64  `gorm:"column:schedule_id;index;not null;" json:"schedule_id"`
	MatchId       int64  `gorm:"column:match_id;uniqueIndex;not null;" json:"match_id"`
}

func (*MatchOccurrence) TableName() string { return "gg_match_occurrence" }

// MatchRuntime journals registrations, payment keys, refunds and the broadcast outbox.
// It commits in the same manager-DB transaction as the Match state.
type MatchRuntime struct {
	MatchId int64  `gorm:"column:match_id;primaryKey;autoIncrement:false;" json:"match_id"`
	Payload string `gorm:"column:payload;type:longtext;not null;" json:"payload"`
}

func (*MatchRuntime) TableName() string { return "gg_match_runtime" }
