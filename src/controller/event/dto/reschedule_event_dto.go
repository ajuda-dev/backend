package dto

import "time"

type RescheduleEventDto struct {
	StartAt time.Time `json:"start_at"`
	Comment string    `json:"comment"`
}
