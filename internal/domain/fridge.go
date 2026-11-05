package domain

import "github.com/google/uuid"

type Product struct {
	ID                    uuid.UUID  `json:"id"`
	Name                  string     `json:"name"`
	CategoryID            *uuid.UUID `json:"category_id"`
	Description           *string    `json:"description"`
	Quantity              *string    `json:"quantity"`
	ImageLink             *string    `json:"image_link"`
	ManufacturedAt        *string    `json:"manufactured_at"`
	ExpiresAt             string     `json:"expires_at"`
	OpenedAt              *string    `json:"opened_at"`
	ShelfLifeAfterOpening *int32     `json:"shelf_life_after_opening"`
	Version               int32      `json:"version"`
}
type Category struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Version int32     `json:"version"`
}
type Indicator struct {
	Color   string `json:"color"`
	MaxDays *int32 `json:"max_days"`
}
type Settings struct {
	HasNotifications bool        `json:"has_notifications"`
	NotificationTime string      `json:"notification_time"`
	Timezone         string      `json:"timezone"`
	Version          int32       `json:"version"`
	Indicators       []Indicator `json:"indicators"`
}
type SettingsPatch struct {
	HasNotifications *bool
	NotificationTime *string
	Timezone         *string
	Indicators       *[]Indicator
}
