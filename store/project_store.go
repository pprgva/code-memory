package store

import (
	"context"
	"time"
)

// IndexStatus represents the current indexing state of a project.
type IndexStatus string

const (
	IndexStatusPending  IndexStatus = "pending"
	IndexStatusIndexing IndexStatus = "indexing"
	IndexStatusReady    IndexStatus = "ready"
	IndexStatusError    IndexStatus = "error"
)

// Project represents a code project in the system.
type Project struct {
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	LocalPath     string      `json:"local_path,omitempty"`
	Languages     []string    `json:"languages,omitempty"`
	Framework     string      `json:"framework,omitempty"`
	FileCount     int         `json:"file_count"`
	ChunkCount    int         `json:"chunk_count"`
	SymbolCount   int         `json:"symbol_count"`
	IndexStatus   IndexStatus `json:"index_status"`
	LastError     string      `json:"last_error,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	LastIndexedAt *time.Time  `json:"last_indexed_at,omitempty"`
}

// File represents a file within a project.
type File struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	RelativePath string    `json:"relative_path"`
	Language     string    `json:"language,omitempty"`
	ContentHash  string    `json:"content_hash"`
	LineCount    int       `json:"line_count"`
	ModTime      time.Time `json:"mod_time"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProjectStore defines the interface for project management operations.
type ProjectStore interface {
	// CreateProject creates a new project in the store.
	CreateProject(ctx context.Context, p *Project) error

	// GetProject retrieves a project by its name.
	GetProject(ctx context.Context, name string) (*Project, error)

	// GetProjectByID retrieves a project by its ID.
	GetProjectByID(ctx context.Context, id string) (*Project, error)

	// UpdateProject updates an existing project.
	UpdateProject(ctx context.Context, p *Project) error

	// DeleteProject removes a project and all its associated data.
	DeleteProject(ctx context.Context, name string) error

	// ListProjects returns all projects in the store.
	ListProjects(ctx context.Context) ([]*Project, error)

	// UpdateProjectStats recalculates and updates project statistics.
	UpdateProjectStats(ctx context.Context, name string) error

	// SetProjectStatus updates the index status and optional error message.
	SetProjectStatus(ctx context.Context, name string, status IndexStatus, lastError string) error
}

// FileStore defines the interface for file management operations.
type FileStore interface {
	// CreateFile creates a new file record.
	CreateFile(ctx context.Context, f *File) error

	// GetFile retrieves a file by project ID and relative path.
	GetFile(ctx context.Context, projectID, relativePath string) (*File, error)

	// UpdateFile updates an existing file record.
	UpdateFile(ctx context.Context, f *File) error

	// DeleteFile removes a file record.
	DeleteFile(ctx context.Context, projectID, relativePath string) error

	// ListFiles returns all files for a project.
	ListFiles(ctx context.Context, projectID string) ([]*File, error)

	// DeleteFilesByProject removes all files for a project.
	DeleteFilesByProject(ctx context.Context, projectID string) error
}
