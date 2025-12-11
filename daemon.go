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

const SocketPath = "/tmp/tnote.sock"

// NoteService holds the in-memory state
type NoteService struct {
	mu     sync.Mutex
	notes  []*Note
	nextID int
	server *rpc.Server
}

// StartDaemon initializes the server
func StartDaemon() {
	// Remove previous socket if exists (cleanup from bad crash)
	os.Remove(SocketPath)

	service := &NoteService{
		notes:  make([]*Note, 0),
		nextID: 1,
	}

	rpcServer := rpc.NewServer()
	rpcServer.RegisterName("NoteService", service)

	l, err := net.Listen("unix", SocketPath)
	if err != nil {
		panic(err)
	}

	// Handle graceful shutdown on OS signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		service.shutdown()
	}()

	rpcServer.Accept(l)
}

// shutdown cleans up and exits
func (s *NoteService) shutdown() {
	os.Remove(SocketPath)
	os.Exit(0)
}

// checkAutoShutdown checks if memory is empty and triggers exit
func (s *NoteService) checkAutoShutdown() {
	if len(s.notes) == 0 {
		// We spin this off to allow the RPC response to return to the client first
		go func() {
			time.Sleep(100 * time.Millisecond)
			s.shutdown()
		}()
	}
}

// --- RPC Methods ---

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
	reply.Message = fmt.Sprintf("Note added with ID: %d", n.ID)
	return nil
}

func (s *NoteService) List(args EmptyArgs, reply *ListReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Return a copy to avoid race conditions
	list := make([]Note, len(s.notes))
	for i, n := range s.notes {
		list[i] = *n
	}
	reply.Notes = list
	return nil
}

func (s *NoteService) resolveID(idStr string) (*Note, int, error) {
	if len(s.notes) == 0 {
		return nil, -1, fmt.Errorf("list is empty")
	}

	if strings.ToLower(idStr) == "first" {
		return s.notes[0], 0, nil
	}
	if strings.ToLower(idStr) == "last" {
		lastIdx := len(s.notes) - 1
		return s.notes[lastIdx], lastIdx, nil
	}

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

func (s *NoteService) Remove(args IDArgs, reply *NoteReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, idx, err := s.resolveID(args.IDStr)
	if err != nil {
		return err
	}

	// Remove from slice
	s.notes = append(s.notes[:idx], s.notes[idx+1:]...)
	reply.Message = fmt.Sprintf("Removed note %d", note.ID)

	s.checkAutoShutdown()
	return nil
}

func (s *NoteService) Clear(args EmptyArgs, reply *NoteReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.notes = []*Note{}
	reply.Message = "All notes cleared"
	s.checkAutoShutdown()
	return nil
}

func (s *NoteService) Pin(args IDArgs, reply *NoteReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, _, err := s.resolveID(args.IDStr)
	if err != nil {
		return err
	}
	note.Pinned = true
	reply.Note = note
	reply.Message = fmt.Sprintf("Pinned note %d", note.ID)
	return nil
}

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
