package terminal

import (
	"bytes"
	"testing"
)

func TestRingBufferAppendsUnderCap(t *testing.T) {
	rb := newRingBuffer(1024)
	rb.Write([]byte("hello "))
	rb.Write([]byte("world"))
	if got := rb.Bytes(); !bytes.Equal(got, []byte("hello world")) {
		t.Fatalf("bytes = %q", got)
	}
}

func TestRingBufferEvictsOldestOverCap(t *testing.T) {
	rb := newRingBuffer(10)
	rb.Write([]byte("0123456789"))
	rb.Write([]byte("abcde"))
	got := rb.Bytes()
	if len(got) != 10 {
		t.Fatalf("len = %d want 10", len(got))
	}
	if !bytes.Equal(got, []byte("56789abcde")) {
		t.Fatalf("bytes = %q want %q", got, "56789abcde")
	}
}

func TestRingBufferSingleWriteLargerThanCap(t *testing.T) {
	rb := newRingBuffer(4)
	rb.Write([]byte("abcdefgh"))
	if got := rb.Bytes(); !bytes.Equal(got, []byte("efgh")) {
		t.Fatalf("bytes = %q want %q", got, "efgh")
	}
}
