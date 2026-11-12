package jsid

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// ID is a numeric primary/foreign key that marshals to JSON string (BE-09).
// Request bodies accept both "123" and 123.
type ID uint64

func (id ID) Uint64() uint64 { return uint64(id) }

func (id ID) String() string { return strconv.FormatUint(uint64(id), 10) }

func (id ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.String())
}

func (id *ID) UnmarshalJSON(b []byte) error {
	if id == nil {
		return fmt.Errorf("jsid.ID: nil receiver")
	}
	if string(b) == "null" {
		*id = 0
		return nil
	}
	var asStr string
	if err := json.Unmarshal(b, &asStr); err == nil {
		if asStr == "" {
			*id = 0
			return nil
		}
		u, err := strconv.ParseUint(asStr, 10, 64)
		if err != nil {
			return err
		}
		*id = ID(u)
		return nil
	}
	var asNum uint64
	if err := json.Unmarshal(b, &asNum); err != nil {
		return err
	}
	*id = ID(asNum)
	return nil
}

func FromUint64(v uint64) ID { return ID(v) }

func PtrFromUint64(v *uint64) *ID {
	if v == nil {
		return nil
	}
	id := ID(*v)
	return &id
}
