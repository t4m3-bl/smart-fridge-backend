package mapper

import (
	"time"

	openapitypes "github.com/oapi-codegen/runtime/types"
)

func DateToTime(value *openapitypes.Date) *time.Time {
	if value == nil {
		return nil
	}

	return new(value.Time)
}
