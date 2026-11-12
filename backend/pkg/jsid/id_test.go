package jsid

import (
	"encoding/json"
	"testing"
)

func TestIDMarshalString(t *testing.T) {
	b, err := json.Marshal(ID(42))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"42"` {
		t.Fatalf("got %s", b)
	}
}

func TestIDUnmarshalNumberOrString(t *testing.T) {
	var a ID
	if err := json.Unmarshal([]byte(`99`), &a); err != nil || a != 99 {
		t.Fatalf("number: %v %d", err, a)
	}
	var b ID
	if err := json.Unmarshal([]byte(`"77"`), &b); err != nil || b != 77 {
		t.Fatalf("string: %v %d", err, b)
	}
}

type sample struct {
	ID ID
}

func TestStructFieldKeepsPascalCase(t *testing.T) {
	b, err := json.Marshal(sample{ID: 5})
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"ID":"5"}` {
		t.Fatalf("got %s", b)
	}
}
