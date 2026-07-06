package mailpit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListSendsPagingAndDecodes(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/messages" {
			gotQuery = r.URL.RawQuery
			_, _ = io.WriteString(w, `{"start":0,"total":2,"unread":1,"messages_count":2,"messages":[{"ID":"a","Subject":"Hi","Read":false},{"ID":"b","Subject":"Yo","Read":true}]}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	res, err := NewClient(srv.URL).List(context.Background(), 0, 50)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if gotQuery != "start=0&limit=50" {
		t.Fatalf("query = %q", gotQuery)
	}
	if res.Total != 2 || res.Unread != 1 || len(res.Messages) != 2 || res.Messages[0].Subject != "Hi" {
		t.Fatalf("decoded = %+v", res)
	}
}

func TestGetDecodesMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/message/a" {
			_, _ = io.WriteString(w, `{"ID":"a","Subject":"Hi","HTML":"<b>x</b>","Text":"x"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	msg, err := NewClient(srv.URL).Get(context.Background(), "a")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if msg.ID != "a" || msg.HTML != "<b>x</b>" {
		t.Fatalf("decoded = %+v", msg)
	}
}

func TestSetReadSendsBody(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == "/api/v1/messages" {
			_ = json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).SetRead(context.Background(), []string{"a", "b"}, true); err != nil {
		t.Fatalf("SetRead: %v", err)
	}
	if body["Read"] != true {
		t.Fatalf("Read = %v", body["Read"])
	}
	ids, _ := body["IDs"].([]any)
	if len(ids) != 2 || ids[0] != "a" {
		t.Fatalf("IDs = %v", body["IDs"])
	}
}

func TestDeleteSendsIDs(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete && r.URL.Path == "/api/v1/messages" {
			_ = json.NewDecoder(r.Body).Decode(&body)
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	if err := NewClient(srv.URL).Delete(context.Background(), []string{"a"}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	ids, _ := body["IDs"].([]any)
	if len(ids) != 1 || ids[0] != "a" {
		t.Fatalf("IDs = %v", body["IDs"])
	}
}

func TestPartReturnsBytesAndContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/message/m1/part/2" {
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("%PDF-1.7 body"))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	data, ct, err := NewClient(srv.URL).Part(context.Background(), "m1", "2")
	if err != nil {
		t.Fatalf("Part: %v", err)
	}
	if ct != "application/pdf" {
		t.Fatalf("content type = %q", ct)
	}
	if string(data) != "%PDF-1.7 body" {
		t.Fatalf("data = %q", data)
	}
}
