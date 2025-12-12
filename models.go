package main

import (
	"time"
)

// Note represents a single casual note entry.
type Note struct {
	ID        int       `json:"id"`         // Incremental ID
	Text      string    `json:"text"`       // The content of the note
	Pinned    bool      `json:"pinned"`     // Visual priority status
	CreatedAt time.Time `json:"created_at"` // Timestamp of creation
}

// AddArgs represents arguments for adding a note.
type AddArgs struct {
	Text string
}

// IDArgs represents arguments for commands targeting a specific note.
// IDStr can be a number ("1"), "first", or "last".
type IDArgs struct {
	IDStr string
}

// EmptyArgs is used for commands that require no input (like List or Clear).
type EmptyArgs struct{}

// NoteReply is the standard response for single-note operations.
type NoteReply struct {
	Note    *Note  // The note object (if applicable)
	Message string // Human-readable success message
	Error   string // Error message (if any)
}

// ListReply is the response for the List command.
type ListReply struct {
	Notes []Note // Slice of all active notes
	Error string
}
