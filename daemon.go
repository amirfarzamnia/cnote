package main

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// SocketPath is the location of the Unix domain socket.
// /tmp is RAM-backed on most Linux distros, making this extremely fast.
const SocketPath = "/tmp/cnote.sock"

// NoteService acts as the RPC server holding the in-memory state.
type NoteService struct {
	mu     sync.Mutex // Mutex ensures thread-safety during concurrent access
	notes  []*Note    // The slice where notes live
	nextID int        // Auto-increment counter
}

// StartDaemon initializes the background process.
// This is only called when the user runs 'cnote add' and no daemon exists.
func StartDaemon() {
	// 1. Clean up potential stale socket files from previous crashes
	os.Remove(SocketPath)

	// 2. Initialize state
	service := &NoteService{
		notes:  make([]*Note, 0),
		nextID: 1,
	}

	// 3. Register RPC Service
	rpcServer := rpc.NewServer()
	rpcServer.RegisterName("NoteService", service)

	// 4. Listen on Unix Socket (faster/safer than TCP for local CLI)
	l, err := net.Listen("unix", SocketPath)
	if err != nil {
		panic(err)
	}

	// 5. Handle OS Interrupts (Ctrl+C) gracefully
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		service.shutdown()
	}()

	// 6. Begin serving requests
	rpcServer.Accept(l)
}

// shutdown cleans up resources and exits the process.
func (s *NoteService) shutdown() {
	os.Remove(SocketPath)
	os.Exit(0)
}

// checkAutoShutdown looks at the note count.
// If zero, it triggers a self-destruct sequence to free system memory.
func (s *NoteService) checkAutoShutdown() {
	if len(s.notes) == 0 {
		// Run in a goroutine to allow the current RPC call to return successfully
		// to the client before the server dies.
		go func() {
			time.Sleep(100 * time.Millisecond)
			s.shutdown()
		}()
	}
}

// resolveID converts "first", "last", or "123" into a specific Note and index.
func (s *NoteService) resolveID(idStr string) (*Note, int, error) {
	if len(s.notes) == 0 {
		return nil, -1, fmt.Errorf("list is empty")
	}

	// Handle keywords
	if strings.ToLower(idStr) == "first" {
		return s.notes[0], 0, nil
	}
	if strings.ToLower(idStr) == "last" {
		lastIdx := len(s.notes) - 1
		return s.notes[lastIdx], lastIdx, nil
	}

	// Handle numeric ID
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, -1, fmt.Errorf("invalid ID format")
	}

	for i, n := range s.notes {
		if n.ID == id {
			return n, i, nil
		}
	}
	return nil, -1, fmt.Errorf("note with ID %d not found", id)
}

// --- RPC Methods ---

// Add creates a new note.
func (s *NoteService) Add(args AddArgs, reply *NoteReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	n := &Note{
		ID:        s.nextID,
		Text:      args.Text,
		Pinned:    false,
		CreatedAt: time.Now(),
	}
	s.notes = append(s.notes, n)
	s.nextID++

	reply.Note = n
	reply.Message = fmt.Sprintf("🎩 Note added (ID: %d)", n.ID)
	return nil
}

// List returns all notes.
func (s *NoteService) List(args EmptyArgs, reply *ListReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Return a copy to ensure thread safety
	list := make([]Note, len(s.notes))
	for i, n := range s.notes {
		list[i] = *n
	}
	reply.Notes = list
	return nil
}

// Remove deletes a note and checks if the server should shut down.
func (s *NoteService) Remove(args IDArgs, reply *NoteReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, idx, err := s.resolveID(args.IDStr)
	if err != nil {
		return err
	}

	// Delete from slice
	s.notes = append(s.notes[:idx], s.notes[idx+1:]...)
	reply.Message = fmt.Sprintf("🗑️ Removed note %d", note.ID)

	// Crucial: Check if we should kill the process
	s.checkAutoShutdown()
	return nil
}

// Clear deletes everything and shuts down.
func (s *NoteService) Clear(args EmptyArgs, reply *NoteReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.notes = []*Note{}
	reply.Message = "✨ All notes cleared. Session ended."
	s.checkAutoShutdown()
	return nil
}

// Pin marks a note as important.
func (s *NoteService) Pin(args IDArgs, reply *NoteReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, _, err := s.resolveID(args.IDStr)
	if err != nil {
		return err
	}
	note.Pinned = true
	reply.Note = note
	reply.Message = fmt.Sprintf("📌 Pinned note %d", note.ID)
	return nil
}

// Unpin removes importance.
func (s *NoteService) Unpin(args IDArgs, reply *NoteReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, _, err := s.resolveID(args.IDStr)
	if err != nil {
		return err
	}
	note.Pinned = false
	reply.Note = note
	reply.Message = fmt.Sprintf("Unpinned note %d", note.ID)
	return nil
}

// Show returns details for a single note.
func (s *NoteService) Show(args IDArgs, reply *NoteReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, _, err := s.resolveID(args.IDStr)
	if err != nil {
		return err
	}
	reply.Note = note
	return nil
}
