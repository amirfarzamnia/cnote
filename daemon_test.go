package main

import (
	"fmt"
	"testing"
	"time"
)

// setupTestService creates a fresh NoteService instance for testing.
func setupTestService() *NoteService {
	// We do not start the actual daemon (net.Listen) in tests.
	// We just test the NoteService methods directly.
	return &NoteService{
		notes:  make([]*Note, 0),
		nextID: 1,
	}
}

// TestAdd ensures notes are added correctly with auto-incrementing IDs and timestamps.
func TestAdd(t *testing.T) {
	s := setupTestService()
	var reply NoteReply

	// 1. Add first note
	err := s.Add(AddArgs{Text: "Test Note 1"}, &reply)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if reply.Note.ID != 1 {
		t.Errorf("Expected ID 1, got %d", reply.Note.ID)
	}
	if s.nextID != 2 {
		t.Errorf("Expected nextID 2, got %d", s.nextID)
	}

	// 2. Add second note
	err = s.Add(AddArgs{Text: "Test Note 2"}, &reply)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if reply.Note.ID != 2 {
		t.Errorf("Expected ID 2, got %d", reply.Note.ID)
	}
	if len(s.notes) != 2 {
		t.Errorf("Expected 2 notes, got %d", len(s.notes))
	}
	// Check CreatedAt is close to current time
	if time.Since(reply.Note.CreatedAt) > time.Second {
		t.Errorf("CreatedAt timestamp is too far in the past")
	}
}

// TestList verifies the List method returns the correct notes.
func TestList(t *testing.T) {
	s := setupTestService()
	s.Add(AddArgs{Text: "N1"}, &NoteReply{})
	s.Add(AddArgs{Text: "N2"}, &NoteReply{})

	var reply ListReply
	err := s.List(EmptyArgs{}, &reply)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(reply.Notes) != 2 {
		t.Fatalf("Expected 2 notes in list, got %d", len(reply.Notes))
	}
	if reply.Notes[0].Text != "N1" {
		t.Errorf("Expected first note N1, got %s", reply.Notes[0].Text)
	}
}

// TestIDResolution verifies the "first", "last", and numeric ID logic.
func TestIDResolution(t *testing.T) {
	s := setupTestService()
	s.Add(AddArgs{Text: "A"}, &NoteReply{}) // ID 1
	s.Add(AddArgs{Text: "B"}, &NoteReply{}) // ID 2
	s.Add(AddArgs{Text: "C"}, &NoteReply{}) // ID 3

	// Test cases: {input, expectedID, expectedIndex}
	tests := []struct {
		input       string
		expectedID  int
		expectedIdx int
		shouldError bool
	}{
		{"1", 1, 0, false},
		{"3", 3, 2, false},
		{"first", 1, 0, false},
		{"last", 3, 2, false},
		{"4", 0, -1, true},   // Non-existent ID
		{"mid", 0, -1, true}, // Invalid keyword
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Input:%s", tt.input), func(t *testing.T) {
			note, idx, err := s.resolveID(tt.input)

			if tt.shouldError {
				if err == nil {
					t.Fatal("Expected an error but got none")
				}
				return // Success if error occurred
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if note.ID != tt.expectedID {
				t.Errorf("Expected ID %d, got %d", tt.expectedID, note.ID)
			}
			if idx != tt.expectedIdx {
				t.Errorf("Expected index %d, got %d", tt.expectedIdx, idx)
			}
		})
	}
}

// TestPinAndUnpin verifies pinning/unpinning a note.
func TestPinAndUnpin(t *testing.T) {
	s := setupTestService()
	s.Add(AddArgs{Text: "Pin Me"}, &NoteReply{}) // ID 1

	// 1. Pin
	var pinReply NoteReply
	err := s.Pin(IDArgs{IDStr: "1"}, &pinReply)
	if err != nil {
		t.Fatalf("Pin failed: %v", err)
	}
	if !pinReply.Note.Pinned {
		t.Error("Note should be pinned after Pin command")
	}

	// 2. Unpin
	var unpinReply NoteReply
	err = s.Unpin(IDArgs{IDStr: "first"}, &unpinReply)
	if err != nil {
		t.Fatalf("Unpin failed: %v", err)
	}
	if unpinReply.Note.Pinned {
		t.Error("Note should be unpinned after Unpin command")
	}
}

// TestRemove verifies note deletion and ID re-indexing logic.
func TestRemove(t *testing.T) {
	s := setupTestService()
	s.Add(AddArgs{Text: "A"}, &NoteReply{}) // ID 1
	s.Add(AddArgs{Text: "B"}, &NoteReply{}) // ID 2
	s.Add(AddArgs{Text: "C"}, &NoteReply{}) // ID 3

	// Remove the middle one (ID 2)
	var reply NoteReply
	err := s.Remove(IDArgs{IDStr: "2"}, &reply)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	if len(s.notes) != 2 {
		t.Fatalf("Expected 2 notes after removal, got %d", len(s.notes))
	}

	// Check remaining notes (should be A and C)
	if s.notes[0].Text != "A" || s.notes[1].Text != "C" {
		t.Errorf("Incorrect notes remaining: %v", s.notes)
	}
}

// TestClear verifies all notes are cleared.
func TestClear(t *testing.T) {
	s := setupTestService()
	s.Add(AddArgs{Text: "A"}, &NoteReply{})
	s.Add(AddArgs{Text: "B"}, &NoteReply{})

	err := s.Clear(EmptyArgs{}, &NoteReply{})
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	if len(s.notes) != 0 {
		t.Errorf("Expected 0 notes after Clear, got %d", len(s.notes))
	}
}

// TestAutoShutdownLogic checks if the daemon correctly prepares to shut down.
// NOTE: We cannot truly test os.Exit(0) in a unit test, so we verify the condition that
// triggers shutdown (the note slice being empty after a deletion).
func TestAutoShutdownLogic(t *testing.T) {
	s := setupTestService()
	s.Add(AddArgs{Text: "A"}, &NoteReply{}) // ID 1

	// Remove the only note. This should trigger checkAutoShutdown.
	var reply NoteReply
	err := s.Remove(IDArgs{IDStr: "1"}, &reply)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	if len(s.notes) != 0 {
		t.Fatalf("Note list should be empty.")
	}
	// In a real run, this completed the process, fulfilling the minimal requirement.
}
