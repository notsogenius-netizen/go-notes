package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/notsogenius-netizen/go-notes/api-gateway/graph"
	"github.com/notsogenius-netizen/go-notes/api-gateway/internal/notes_client"
)

const (
	notesServiceAddr = "localhost:50051" // Your Notes service address
	defaultPort      = "8080"
)

func main() {
	notesClient, err := notes_client.NewClient(notesServiceAddr)
	if err != nil {
		log.Fatalf("failed to create notes client: %v", err)
	}
	defer notesClient.Close()

	// Configure GraphQL server
	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{
		Resolvers: &graph.Resolver{
			NotesClient: notesClient,
		},
	}))

	// Set up HTTP routes
	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv) // Removed auth middleware

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}