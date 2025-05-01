package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/notsogenius-netizen/go-notes/notes-service/internal/model"
	"gorm.io/gorm"
)

// ChangeType represents the type of change to a note
type ChangeType string

const (
	Created ChangeType = "created"
	Updated ChangeType = "updated" 
	Deleted ChangeType = "deleted"
)

// NoteChange represents a change to a note with its type
type NoteChange struct {
	Type ChangeType
	Note *model.Note
}

// NoteRepository defines the interface for note operations
type NoteRepository interface {
	Create(ctx context.Context, note *model.Note) (*model.Note, error)
	GetByID(ctx context.Context, id, userID string) (*model.Note, error)
	GetAllByUserID(ctx context.Context, userID string) ([]*model.Note, error)
	Update(ctx context.Context, note *model.Note) (*model.Note, error)
	Delete(ctx context.Context, id, userID string) error
	Subscribe(userID string) (<-chan *NoteChange)
	Unsubscribe(userID string, ch <-chan *NoteChange)
}

// GormNoteRepository implements NoteRepository using GORM
type GormNoteRepository struct {
	db           *gorm.DB
	subscribers  map[string]map[chan *NoteChange]bool
	mu           sync.RWMutex
}

// NewGormNoteRepository creates a new GormNoteRepository
func NewGormNoteRepository(db *gorm.DB) *GormNoteRepository {
	// Auto migrate the schema
	db.AutoMigrate(&model.Note{})
	
	return &GormNoteRepository{
		db:          db,
		subscribers: make(map[string]map[chan *NoteChange]bool),
	}
}

// Create adds a new note to the database
func (r *GormNoteRepository) Create(ctx context.Context, note *model.Note) (*model.Note, error) {
	tx := r.db.WithContext(ctx).Create(note)
	if tx.Error != nil {
		return nil, tx.Error
	}

	// Notify subscribers
	go r.notify(&NoteChange{
		Type: Created,
		Note: note,
	}, note.UserID)

	return note, nil
}

// GetByID retrieves a note by its ID and user ID
func (r *GormNoteRepository) GetByID(ctx context.Context, id, userID string) (*model.Note, error) {
	var note model.Note
	tx := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&note)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("note not found")
		}
		return nil, tx.Error
	}
	return &note, nil
}

// GetAllByUserID retrieves all notes for a user
func (r *GormNoteRepository) GetAllByUserID(ctx context.Context, userID string) ([]*model.Note, error) {
	var notes []*model.Note
	tx := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("updated_at DESC").Find(&notes)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return notes, nil
}

// Update updates an existing note
func (r *GormNoteRepository) Update(ctx context.Context, note *model.Note) (*model.Note, error) {
	// First check if note exists and belongs to user
	var existingNote model.Note
	tx := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", note.ID, note.UserID).First(&existingNote)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("note not found or unauthorized")
		}
		return nil, tx.Error
	}

	// Update note
	updateTx := r.db.WithContext(ctx).Model(&existingNote).Updates(map[string]interface{}{
		"title":   note.Title,
		"content": note.Content,
	})
	if updateTx.Error != nil {
		return nil, updateTx.Error
	}

	// Get updated note
	r.db.WithContext(ctx).First(&existingNote, "id = ?", note.ID)
	
	// Notify subscribers
	go r.notify(&NoteChange{
		Type: Updated,
		Note: &existingNote,
	}, note.UserID)

	return &existingNote, nil
}

// Delete removes a note from the database
func (r *GormNoteRepository) Delete(ctx context.Context, id, userID string) error {
	// First check if note exists and belongs to user
	var existingNote model.Note
	tx := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&existingNote)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return errors.New("note not found or unauthorized")
		}
		return tx.Error
	}

	// Delete note
	deleteTx := r.db.WithContext(ctx).Delete(&existingNote)
	if deleteTx.Error != nil {
		return deleteTx.Error
	}

	// Notify subscribers
	go r.notify(&NoteChange{
		Type: Deleted,
		Note: &existingNote,
	}, userID)

	return nil
}

// Subscribe registers a subscriber for note changes
func (r *GormNoteRepository) Subscribe(userID string) (<-chan *NoteChange) {
	ch := make(chan *NoteChange, 100)
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, ok := r.subscribers[userID]; !ok {
		r.subscribers[userID] = make(map[chan *NoteChange]bool)
	}
	r.subscribers[userID][ch] = true
	
	return ch
}

// Unsubscribe removes a subscriber
func (r *GormNoteRepository) Unsubscribe(userID string, ch <-chan *NoteChange) {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if channels, ok := r.subscribers[userID]; ok {
        // We need to find the channel in the map by its pointer identity
        // rather than trying to type assert it
        for storedCh := range channels {
            if ch == storedCh {
                delete(channels, storedCh)
                break
            }
        }
        
        if len(channels) == 0 {
            delete(r.subscribers, userID)
        }
    }
}

// notify sends a note change to all subscribers for a user
func (r *GormNoteRepository) notify(change *NoteChange, userID string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if channels, ok := r.subscribers[userID]; ok {
		for ch := range channels {
			select {
			case ch <- change:
				// Successfully sent
			default:
				// Channel is full or closed
			}
		}
	}
}