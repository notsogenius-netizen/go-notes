package service

import (
	"context"

	"github.com/notsogenius-netizen/go-notes/notes-service/internal/model"
	"github.com/notsogenius-netizen/go-notes/notes-service/internal/repository"
	pb "github.com/notsogenius-netizen/go-notes/notes-service/pkg/proto/gen"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type NoteService struct {
	pb.UnimplementedNoteServiceServer
	repo repository.NoteRepository
}

func NewNoteService(repo repository.NoteRepository) *NoteService {
	return &NoteService{
		repo: repo,
	}
}

func (s *NoteService) CreateNote(ctx context.Context, req *pb.CreateNoteRequest) (*pb.Note, error) {
	note := &model.Note{
		Title:   req.Title,
		Content: req.Content,
		UserID:  req.UserId,
	}
	
	createdNote, err := s.repo.Create(ctx, note)
	if err != nil {
		return nil, err
	}
	
	return &pb.Note{
		Id:        createdNote.ID,
		Title:     createdNote.Title,
		Content:   createdNote.Content,
		UserId:    createdNote.UserID,
		CreatedAt: timestamppb.New(createdNote.CreatedAt),
		UpdatedAt: timestamppb.New(createdNote.UpdatedAt),
	}, nil
}

func (s *NoteService) GetNote(ctx context.Context, req *pb.GetNoteRequest) (*pb.Note, error) {
	note, err := s.repo.GetByID(ctx, req.Id, req.UserId)
	if err != nil {
		return nil, err
	}
	
	return &pb.Note{
		Id:        note.ID,
		Title:     note.Title,
		Content:   note.Content,
		UserId:    note.UserID,
		CreatedAt: timestamppb.New(note.CreatedAt),
		UpdatedAt: timestamppb.New(note.UpdatedAt),
	}, nil
}

func (s *NoteService) ListNotes(ctx context.Context, req *pb.ListNotesRequest) (*pb.ListNotesResponse, error) {
	notes, err := s.repo.GetAllByUserID(ctx, req.UserId)
	if err != nil {
		return nil, err
	}
	
	var protoNotes []*pb.Note
	for _, note := range notes {
		protoNotes = append(protoNotes, &pb.Note{
			Id:        note.ID,
			Title:     note.Title,
			Content:   note.Content,
			UserId:    note.UserID,
			CreatedAt: timestamppb.New(note.CreatedAt),
			UpdatedAt: timestamppb.New(note.UpdatedAt),
		})
	}
	
	return &pb.ListNotesResponse{
		Notes: protoNotes,
	}, nil
}

func (s *NoteService) UpdateNote(ctx context.Context, req *pb.UpdateNoteRequest) (*pb.Note, error) {
	note := &model.Note{
		ID:      req.Id,
		Title:   req.Title,
		Content: req.Content,
		UserID:  req.UserId,
	}
	
	updatedNote, err := s.repo.Update(ctx, note)
	if err != nil {
		return nil, err
	}
	
	return &pb.Note{
		Id:        updatedNote.ID,
		Title:     updatedNote.Title,
		Content:   updatedNote.Content,
		UserId:    updatedNote.UserID,
		CreatedAt: timestamppb.New(updatedNote.CreatedAt),
		UpdatedAt: timestamppb.New(updatedNote.UpdatedAt),
	}, nil
}

func (s *NoteService) DeleteNote(ctx context.Context, req *pb.DeleteNoteRequest) (*emptypb.Empty, error) {
	err := s.repo.Delete(ctx, req.Id, req.UserId)
	if err != nil {
		return nil, err
	}
	
	return &emptypb.Empty{}, nil
}

func (s *NoteService) WatchNotes(req *pb.WatchNotesRequest, stream pb.NoteService_WatchNotesServer) error {
	userID := req.UserId
	
	changeCh := s.repo.Subscribe(userID)
	
	defer s.repo.Unsubscribe(userID, changeCh)
	
	for {
		select {
		case change, ok := <-changeCh:
			if !ok {
				return nil
			}
			
			var changeType pb.NoteChange_ChangeType
			switch change.Type {
			case repository.Created:
				changeType = pb.NoteChange_CREATED
			case repository.Updated:
				changeType = pb.NoteChange_UPDATED
			case repository.Deleted:
				changeType = pb.NoteChange_DELETED
			}
			
			protoNote := &pb.Note{
				Id:        change.Note.ID,
				Title:     change.Note.Title,
				Content:   change.Note.Content,
				UserId:    change.Note.UserID,
				CreatedAt: timestamppb.New(change.Note.CreatedAt),
				UpdatedAt: timestamppb.New(change.Note.UpdatedAt),
			}
			
			if err := stream.Send(&pb.NoteChange{
				Type: changeType,
				Note: protoNote,
			}); err != nil {
				return err
			}
			
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}