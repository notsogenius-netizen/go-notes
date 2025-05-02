package notes_client

import (
	"context"
	"io"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/notsogenius-netizen/go-notes/api-gateway/proto/gen"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.NoteServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:   conn,
		client: pb.NewNoteServiceClient(conn),
	}, nil
}

func (c *Client) CreateNote(ctx context.Context, userID, title, content string) (*pb.Note, error) {
	return c.client.CreateNote(ctx, &pb.CreateNoteRequest{
		UserId:  userID,
		Title:   title,
		Content: content,
	})
}

func (c *Client) GetNote(ctx context.Context, id, user_id string) (*pb.Note, error) {
	return c.client.GetNote(ctx, &pb.GetNoteRequest{
		Id: id,
		UserId: user_id,
	})
}

func (c *Client) ListNotes(ctx context.Context, userID string) ([]*pb.Note, error) {
	resp, err := c.client.ListNotes(ctx, &pb.ListNotesRequest{
		UserId: userID,
	})
	if err != nil {
		return nil, err
	}
	return resp.Notes, nil
}

func (c *Client) UpdateNote(ctx context.Context, id, title, content string) (*pb.Note, error) {
	return c.client.UpdateNote(ctx, &pb.UpdateNoteRequest{
		Id:      id,
		Title:   title,
		Content: content,
	})
}

func (c *Client) DeleteNote(ctx context.Context, id string) error {
	_, err := c.client.DeleteNote(ctx, &pb.DeleteNoteRequest{
		Id: id,
	})
	return err
}

func (c *Client) SubscribeToNoteChanges(ctx context.Context, userID string, ch chan<- *pb.NoteChange) error {
	stream, err := c.client.WatchNotes(ctx, &pb.WatchNotesRequest{
		UserId: userID,
	})
	if err != nil {
		return err
	}

	go func() {
		for {
			event, err := stream.Recv()
			if err == io.EOF {
				log.Println("Stream closed by server")
				close(ch)
				return
			}
			if err != nil {
				log.Printf("Error receiving from stream: %v", err)
				close(ch)
				return
			}
			ch <- event
		}
	}()

	return nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
