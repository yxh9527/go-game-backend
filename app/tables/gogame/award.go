package gogame

import (
	"encoding/json"
	"fmt"
)

// AwardItem 奖励条目。钱必须带货币类型。
type AwardItem struct {
	Kind     string `json:"kind"`               // money / ticket
	Currency string `json:"currency,omitempty"` // kind=money 时必填
	Amount   string `json:"amount"`
	TicketID int64  `json:"ticketId,omitempty"` // kind=ticket 时必填
}

// AwardRank 名次奖励。
type AwardRank struct {
	Rank  int         `json:"rank"`
	Items []AwardItem `json:"items"`
}

// AwardTemplate 奖励模板，Detail 存 AwardRank JSON 数组。
type AwardTemplate struct {
	Id         int64  `gorm:"column:id;primaryKey;autoIncrement;" json:"id"`
	Name       string `gorm:"column:name;size:128;" json:"name"`
	Detail     string `gorm:"column:detail;type:text;" json:"detail"`
	CreateTime int64  `gorm:"column:create_time;default:0;" json:"create_time"`
	UpdateTime int64  `gorm:"column:update_time;default:0;" json:"update_time"`
}

func (t *AwardTemplate) TableName() string {
	return "gg_award_template"
}

func ValidateAwardJSON(raw string) error {
	if raw == "" {
		return nil
	}
	var ranks []AwardRank
	if err := json.Unmarshal([]byte(raw), &ranks); err != nil {
		return err
	}
	for _, rank := range ranks {
		if rank.Rank <= 0 {
			return fmt.Errorf("invalid rank")
		}
		for _, item := range rank.Items {
			switch item.Kind {
			case RewardKindMoney:
				if item.Currency == "" || item.Amount == "" {
					return fmt.Errorf("money reward requires currency and amount")
				}
			case RewardKindTicket:
				if item.TicketID <= 0 || item.Amount == "" {
					return fmt.Errorf("ticket reward requires ticketId and amount")
				}
			default:
				return fmt.Errorf("unknown reward kind %s", item.Kind)
			}
		}
	}
	return nil
}
