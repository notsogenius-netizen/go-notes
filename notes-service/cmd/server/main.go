package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/notsogenius-netizen/go-notes/notes-service/internal/repository"
	"github.com/notsogenius-netizen/go-notes/notes-service/internal/service"
	pb "github.com/notsogenius-netizen/go-notes/notes-service/pkg/proto/gen"
)

func main() {
	// Set up database connection
	dsn := "host=localhost user=postgres password=mypassword dbname=developer_notes port=5432 sslmode=disable"
	// Use environment variables in a real application
	if envDSN := os.Getenv("DATABASE_URL"); envDSN != "" {
		dsn = envDSN
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Create repository
	noteRepo := repository.NewGormNoteRepository(db)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register service
	noteService := service.NewNoteService(noteRepo)
	pb.RegisterNoteServiceServer(grpcServer, noteService)

	// Start gRPC server
	port := "50051"
	if envPort := os.Getenv("GRPC_PORT"); envPort != "" {
		port = envPort
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Starting gRPC server on port %s", port)

	// Handle shutdown gracefully
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for interrupt signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	log.Println("Shutting down gRPC server...")
	grpcServer.GracefulStop()
	log.Println("Server shut down")
}