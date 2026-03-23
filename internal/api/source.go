package api

import (
	"fmt"
	"os"
	"path/filepath"
)

// Source describes a Lua script payload before it enters the frontend pipeline.
type Source struct {
	Name    string
	Content string
}

// NewStringSource creates a named in-memory Lua source payload.
func NewStringSource(name, content string) Source {
	sourceName := name
	if sourceName == "" {
		sourceName = "<memory>"
	}

	return Source{
		Name:    sourceName,
		Content: content,
	}
}

// NewFileSource loads a Lua source payload from a file on disk.
func NewFileSource(path string) (Source, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Source{}, fmt.Errorf("read lua source file %q: %w", path, err)
	}

	return Source{
		Name:    filepath.Clean(path),
		Content: string(content),
	}, nil
}
