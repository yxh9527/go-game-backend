package gogame

import "encoding/json"

// Envelope Redis Pub/Sub 信封。Data 必须是 JSON 字符串，不是嵌套对象。
type Envelope struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

func NewEnvelope(event string, payload any) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	env := Envelope{Event: event, Data: string(raw)}
	b, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
