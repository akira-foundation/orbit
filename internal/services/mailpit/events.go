package mailpit

import (
	"context"
	"encoding/json"

	"github.com/gorilla/websocket"
)

type Event struct {
	Type string          `json:"Type"`
	Data json.RawMessage `json:"Data"`
}

func DecodeEvent(raw []byte) (Event, error) {
	var ev Event
	err := json.Unmarshal(raw, &ev)
	return ev, err
}

func (e Event) Summary() (MessageSummary, bool) {
	if e.Type != "new" && e.Type != "update" {
		return MessageSummary{}, false
	}
	var sum MessageSummary
	if err := json.Unmarshal(e.Data, &sum); err != nil || sum.ID == "" {
		return MessageSummary{}, false
	}
	return sum, true
}

func Watch(ctx context.Context, wsURL string, onEvent func(Event)) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		ev, err := DecodeEvent(raw)
		if err != nil {
			continue
		}
		onEvent(ev)
	}
}
