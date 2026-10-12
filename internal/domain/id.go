package domain

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type ID uuid.UUID

func NewID() (ID, error) {
	u, err := uuid.NewV7()
	if err != nil {
		return ID{}, fmt.Errorf("UUIDv7 generation error: %w", err)
	}
	return ID(u), nil
}

func (id *ID) String() string {
	return uuid.UUID(*id).String()
}

func (id *ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(uuid.UUID(*id).String())
}

func (id *ID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := uuid.Parse(s)
	if err != nil {
		return err
	}

	*id = ID(parsed)
	return nil
}
