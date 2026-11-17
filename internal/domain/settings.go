package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MaxIndicators  = 10
	MaxTimezoneLen = 64
)

var indicatorColorRe = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type Indicator struct {
	Color   string
	MaxDays *int32
}

type Settings struct {
	UserID           UserID
	HasNotifications bool
	NotificationTime time.Duration
	Timezone         string
	Indicators       []Indicator
	Version          int32
}

type SettingsChanges struct {
	HasNotifications *bool
	NotificationTime *time.Duration
	Timezone         *string
	Indicators       *[]Indicator
}

type SettingsPatch struct {
	UserID UserID
	SettingsChanges
}

func NewSettingsPatch(userID UserID, c SettingsChanges) (*SettingsPatch, error) {
	if uuid.UUID(userID) == uuid.Nil {
		return nil, NewValidationErr("userId", "user id must not be nil")
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}

	return &SettingsPatch{UserID: userID, SettingsChanges: c}, nil
}

func (c *SettingsChanges) Validate() error {
	if c.HasNotifications == nil && c.NotificationTime == nil && c.Timezone == nil && c.Indicators == nil {
		return NewValidationErr("body", "at least one field is required")
	}
	if t := c.NotificationTime; t != nil && (*t < 0 || *t >= 24*time.Hour) {
		return NewValidationErr("notificationTime", "must be between 00:00:00 and 23:59:59")
	}
	if err := c.validateTimezone(); err != nil {
		return err
	}
	if c.Indicators != nil {
		list, err := normalizeIndicators(*c.Indicators)
		if err != nil {
			return err
		}
		c.Indicators = &list
	}
	return nil
}

func (c *SettingsChanges) validateTimezone() error {
	if c.Timezone == nil {
		return nil
	}
	tz := *c.Timezone
	if tz == "" || tz == "Local" || len(tz) > MaxTimezoneLen {
		return NewValidationErr("timezone", "must be an IANA time zone name")
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return NewValidationErr("timezone", "unknown time zone")
	}
	return nil
}

func normalizeIndicators(in []Indicator) ([]Indicator, error) {
	if len(in) > MaxIndicators {
		return nil, NewValidationErr("indicators", fmt.Sprintf("at most %d indicators", MaxIndicators))
	}

	out := make([]Indicator, 0, len(in))
	colors := make(map[string]struct{}, len(in))
	days := make(map[int32]struct{}, len(in))
	hasNoLimit := false

	for _, ind := range in {
		if !indicatorColorRe.MatchString(ind.Color) {
			return nil, NewValidationErr("indicators", "color must look like #RRGGBB")
		}
		color := strings.ToUpper(ind.Color)
		if _, dup := colors[color]; dup {
			return nil, NewValidationErr("indicators", "duplicate color")
		}
		colors[color] = struct{}{}

		switch {
		case ind.MaxDays == nil:
			if hasNoLimit {
				return nil, NewValidationErr("indicators", "only one indicator may have no maxDays")
			}
			hasNoLimit = true
		case *ind.MaxDays < 0:
			return nil, NewValidationErr("indicators", "maxDays must be >= 0")
		default:
			if _, dup := days[*ind.MaxDays]; dup {
				return nil, NewValidationErr("indicators", "duplicate maxDays")
			}
			days[*ind.MaxDays] = struct{}{}
		}

		out = append(out, Indicator{Color: color, MaxDays: ind.MaxDays})
	}
	return out, nil
}
