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

type NoteChange struct {
	Type ChangeType
	Note *model.Note
}

type NoteRepository interface {
	Create(ctx context.Context, note *model.Note) (*model.Note, error)
	GetByID(ctx context.Context, id, userID string) (*model.Note, error)
	GetAllByUserID(ctx context.Context, userID string) ([]*model.Note, error)
	Update(ctx context.Context, note *model.Note) (*model.Note, error)
	Delete(ctx context.Context, id, userID string) error
	Subscribe(userID string) (<-chan *NoteChange)
	Unsubscribe(userID string, ch <-chan *NoteChange)
}

type GormNoteRepository struct {
	db           *gorm.DB
	subscribers  map[string]map[chan *NoteChange]bool
	mu           sync.RWMutex
}

func NewGormNoteRepository(db *gorm.DB) *GormNoteRepository {
	db.AutoMigrate(&model.Note{})
	
	return &GormNoteRepository{
		db:          db,
		subscribers: make(map[string]map[chan *NoteChange]bool),
	}
}

func (r *GormNoteRepository) Create(ctx context.Context, note *model.Note) (*model.Note, error) {
	tx := r.db.WithContext(ctx).Create(note)
	if tx.Error != nil {
		return nil, tx.Error
	}

	go r.notify(&NoteChange{
		Type: Created,
		Note: note,
	}, note.UserID)

	return note, nil
}

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

func (r *GormNoteRepository) GetAllByUserID(ctx context.Context, userID string) ([]*model.Note, error) {
	var notes []*model.Note
	tx := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("updated_at DESC").Find(&notes)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return notes, nil
}

func (r *GormNoteRepository) Update(ctx context.Context, note *model.Note) (*model.Note, error) {
	var existingNote model.Note
	tx := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", note.ID, note.UserID).First(&existingNote)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("note not found or unauthorized")
		}
		return nil, tx.Error
	}

	updateTx := r.db.WithContext(ctx).Model(&existingNote).Updates(map[string]interface{}{
		"title":   note.Title,
		"content": note.Content,
	})
	if updateTx.Error != nil {
		return nil, updateTx.Error
	}

	r.db.WithContext(ctx).First(&existingNote, "id = ?", note.ID)
	
	go r.notify(&NoteChange{
		Type: Updated,
		Note: &existingNote,
	}, note.UserID)

	return &existingNote, nil
}

func (r *GormNoteRepository) Delete(ctx context.Context, id, userID string) error {
	var existingNote model.Note
	tx := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&existingNote)
	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return errors.New("note not found or unauthorized")
		}
		return tx.Error
	}

	deleteTx := r.db.WithContext(ctx).Delete(&existingNote)
	if deleteTx.Error != nil {
		return deleteTx.Error
	}

	go r.notify(&NoteChange{
		Type: Deleted,
		Note: &existingNote,
	}, userID)

	return nil
}

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

func (r *GormNoteRepository) Unsubscribe(userID string, ch <-chan *NoteChange) {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if channels, ok := r.subscribers[userID]; ok {
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

func (r *GormNoteRepository) notify(change *NoteChange, userID string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if channels, ok := r.subscribers[userID]; ok {
		for ch := range channels {
			select {
			case ch <- change:
			default:
			}
		}
	}
}