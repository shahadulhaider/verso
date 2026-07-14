package repository

import (
	"testing"
)

func TestEncodeCursorRoundTrip(t *testing.T) {
	id := "01HXYZ1234567890ABCDEFGH"
	encoded := EncodeCursor(id)
	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded != id {
		t.Errorf("got %q, want %q", decoded, id)
	}
}

func TestDecodeCursorInvalid(t *testing.T) {
	_, err := DecodeCursor("!!!invalid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestEncodeCursorEmpty(t *testing.T) {
	encoded := EncodeCursor("")
	decoded, err := DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded != "" {
		t.Errorf("got %q, want empty", decoded)
	}
}
