package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Note struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid"`
	Title     string    `json:"title" gorm:"not null"`
	Content   string    `json:"content" gorm:"type:text"`
	UserID    string    `json:"user_id" gorm:"index;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (note *Note) BeforeCreate(tx *gorm.DB) error {
	if note.ID == "" {
		note.ID = uuid.New().String()
	}
	return nil
}