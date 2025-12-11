package main

import (
	"time"
)

// Note defines the structure of a note
type Note struct {
	ID        int       `json:"id"`
	Text      string    `json:"text"`
	Pinned    bool      `json:"pinned"`
	CreatedAt time.Time `json:"created_at"`
}

// RPC Args/Reply structures
type AddArgs struct {
	Text string
}

type IDArgs struct {
	IDStr string // Can be "1", "2", "first", "last"
}

type EmptyArgs struct{}

type NoteReply struct {
	Note    *Note
	Message string
	Error   string
}

type ListReply struct {
	Notes []Note
	Error string
}
