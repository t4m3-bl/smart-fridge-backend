package config

import (
	"fmt"
	"strings"
)

type Env int

const (
	DEV Env = iota
	TEST
	PROD
)

func (m Env) String() string {
	switch m {
	case DEV:
		return "dev"
	case TEST:
		return "test"
	case PROD:
		return "prod"
	default:
		return fmt.Sprintf("Env(%d)", int(m))
	}
}

func (m *Env) UnmarshalText(text []byte) error {
	switch strings.ToLower(strings.TrimSpace(string(text))) {
	case "dev":
		*m = DEV
	case "test":
		*m = TEST
	case "prod":
		*m = PROD
	default:
		return fmt.Errorf("unknown env %q, expected dev, test or prod", text)
	}
	return nil
}
