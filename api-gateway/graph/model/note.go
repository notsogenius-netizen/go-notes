package model

import "time"

type Note struct {
    ID        string    `json:"id"`
    Title     string    `json:"title"`
    Content   string    `json:"content"`
    UserID    string    `json:"userId"`
    CreatedAt time.Time `json:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt"`
}

type NoteChange struct {
    Type NoteChangeType `json:"type"`
    Note *Note          `json:"note"`
}

type NoteChangeType string
const (
	Created NoteChangeType = "CREATED"
	Updated NoteChangeType = "UPDATED"
	Deleted NoteChangeType = "DELETED"
)