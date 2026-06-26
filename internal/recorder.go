package internal

import (
	"encoding/json"
	"os"
	"sync"
)

// Recorder appends every executed command to a newline-delimited JSON file so
// sessions can be replayed later for reproducing bugs or seeding CI.
type Recorder struct {
	mu sync.Mutex
	f  *os.File
}

// NewRecorder opens (or creates) the given file for appending recorded commands.
func NewRecorder(path string) (*Recorder, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &Recorder{f: f}, nil
}

// Record writes a single command and its arguments as one JSON line.
func (r *Recorder) Record(command string, args []interface{}) {
	if r == nil {
		return
	}
	entry := make([]interface{}, 0, len(args)+1)
	entry = append(entry, command)
	entry = append(entry, args...)

	b, err := json.Marshal(entry)
	if err != nil {
		return
	}
	r.mu.Lock()
	r.f.Write(append(b, '\n'))
	r.mu.Unlock()
}

// Close flushes and closes the underlying file.
func (r *Recorder) Close() error {
	if r == nil || r.f == nil {
		return nil
	}
	return r.f.Close()
}
