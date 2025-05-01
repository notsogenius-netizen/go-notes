package graph

import "github.com/notsogenius-netizen/go-notes/api-gateway/internal/notes_client"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct{
	NotesClient *notes_client.Client
}
