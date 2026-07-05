package mailpit

import "testing"

func TestDecodeEventAndSummary(t *testing.T) {
	ev, err := DecodeEvent([]byte(`{"Type":"new","Data":{"ID":"a","Subject":"Hi","Read":false}}`))
	if err != nil {
		t.Fatalf("DecodeEvent: %v", err)
	}
	if ev.Type != "new" {
		t.Fatalf("Type = %q", ev.Type)
	}
	sum, ok := ev.Summary()
	if !ok || sum.ID != "a" || sum.Subject != "Hi" {
		t.Fatalf("Summary = %+v ok=%v", sum, ok)
	}
}

func TestSummaryFalseForNonMessageEvent(t *testing.T) {
	ev, err := DecodeEvent([]byte(`{"Type":"truncate","Data":null}`))
	if err != nil {
		t.Fatalf("DecodeEvent: %v", err)
	}
	if _, ok := ev.Summary(); ok {
		t.Fatalf("expected no summary for truncate")
	}
}
