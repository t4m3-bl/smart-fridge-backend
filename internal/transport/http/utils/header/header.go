package header

import (
	"smart-fridge-backend/internal/domain"
	"strconv"
)

func GetVersion(header string) (int32, error) {
	if len(header) < 3 || header[0] != '"' || header[len(header)-1] != '"' {
		return 0, domain.NewValidationErr("If-Match", `must be a single quoted version, e.g. "3"`)
	}
	n, err := strconv.ParseInt(header[1:len(header)-1], 10, 32)
	if err != nil || n < 1 {
		return 0, domain.NewValidationErr("If-Match", "invalid version")
	}
	return int32(n), nil
}

func ResolveVersion(ifMatch, ifNoneMatch *string) (int32, error) {
	switch {
	case ifMatch != nil && ifNoneMatch != nil:
		return 0, domain.NewValidationErr("If-Match", "send either If-Match or If-None-Match")
	case ifNoneMatch != nil:
		if *ifNoneMatch != "*" {
			return 0, domain.NewValidationErr("If-None-Match", `must be "*"`)
		}
		return 0, nil
	case ifMatch != nil:
		return GetVersion(*ifMatch)
	default:
		return 0, domain.ErrPreconditionRequired
	}
}
