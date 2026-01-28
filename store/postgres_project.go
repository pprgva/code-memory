package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresProjectStore implements ProjectStore and FileStore for PostgreSQL.
type PostgresProjectStore struct {
	pool *pgxpool.Pool
}

// NewPostgresProjectStore creates a new PostgresProjectStore.
func NewPostgresProjectStore(pool *pgxpool.Pool) *PostgresProjectStore {
	return &PostgresProjectStore{pool: pool}
}

// Ensure PostgresProjectStore implements both interfaces.
var _ ProjectStore = (*PostgresProjectStore)(nil)
var _ FileStore = (*PostgresProjectStore)(nil)

// CreateProject creates a new project in the store.
func (s *PostgresProjectStore) CreateProject(ctx context.Context, p *Project) error {
	query := `
		INSERT INTO grepai_projects (name, local_path, languages, framework, file_count, chunk_count, symbol_count, index_status, last_error, last_indexed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, updated_at`

	err := s.pool.QueryRow(ctx, query,
		p.Name,
		nullableString(p.LocalPath),
		p.Languages,
		nullableString(p.Framework),
		p.FileCount,
		p.ChunkCount,
		p.SymbolCount,
		string(p.IndexStatus),
		nullableString(p.LastError),
		p.LastIndexedAt,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	return nil
}

// GetProject retrieves a project by its name.
func (s *PostgresProjectStore) GetProject(ctx context.Context, name string) (*Project, error) {
	query := `
		SELECT id, name, local_path, languages, framework, file_count, chunk_count, symbol_count,
		       index_status, last_error, created_at, updated_at, last_indexed_at
		FROM grepai_projects
		WHERE name = $1`

	return s.scanProject(ctx, query, name)
}

// GetProjectByID retrieves a project by its ID.
func (s *PostgresProjectStore) GetProjectByID(ctx context.Context, id string) (*Project, error) {
	query := `
		SELECT id, name, local_path, languages, framework, file_count, chunk_count, symbol_count,
		       index_status, last_error, created_at, updated_at, last_indexed_at
		FROM grepai_projects
		WHERE id = $1`

	return s.scanProject(ctx, query, id)
}

func (s *PostgresProjectStore) scanProject(ctx context.Context, query string, arg interface{}) (*Project, error) {
	var p Project
	var localPath, framework, lastError *string
	var indexStatus string

	err := s.pool.QueryRow(ctx, query, arg).Scan(
		&p.ID,
		&p.Name,
		&localPath,
		&p.Languages,
		&framework,
		&p.FileCount,
		&p.ChunkCount,
		&p.SymbolCount,
		&indexStatus,
		&lastError,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.LastIndexedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	p.LocalPath = derefString(localPath)
	p.Framework = derefString(framework)
	p.LastError = derefString(lastError)
	p.IndexStatus = IndexStatus(indexStatus)

	return &p, nil
}

// UpdateProject updates an existing project.
func (s *PostgresProjectStore) UpdateProject(ctx context.Context, p *Project) error {
	query := `
		UPDATE grepai_projects
		SET name = $2, local_path = $3, languages = $4, framework = $5,
		    file_count = $6, chunk_count = $7, symbol_count = $8,
		    index_status = $9, last_error = $10, last_indexed_at = $11, updated_at = NOW()
		WHERE id = $1`

	result, err := s.pool.Exec(ctx, query,
		p.ID,
		p.Name,
		nullableString(p.LocalPath),
		p.Languages,
		nullableString(p.Framework),
		p.FileCount,
		p.ChunkCount,
		p.SymbolCount,
		string(p.IndexStatus),
		nullableString(p.LastError),
		p.LastIndexedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("project not found: %s", p.ID)
	}

	return nil
}

// DeleteProject removes a project and all its associated data.
func (s *PostgresProjectStore) DeleteProject(ctx context.Context, name string) error {
	// CASCADE will handle associated files
	result, err := s.pool.Exec(ctx,
		`DELETE FROM grepai_projects WHERE name = $1`,
		name,
	)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("project not found: %s", name)
	}

	return nil
}

// ListProjects returns all projects in the store.
func (s *PostgresProjectStore) ListProjects(ctx context.Context) ([]*Project, error) {
	query := `
		SELECT id, name, local_path, languages, framework, file_count, chunk_count, symbol_count,
		       index_status, last_error, created_at, updated_at, last_indexed_at
		FROM grepai_projects
		ORDER BY name`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []*Project
	for rows.Next() {
		var p Project
		var localPath, framework, lastError *string
		var indexStatus string

		err := rows.Scan(
			&p.ID,
			&p.Name,
			&localPath,
			&p.Languages,
			&framework,
			&p.FileCount,
			&p.ChunkCount,
			&p.SymbolCount,
			&indexStatus,
			&lastError,
			&p.CreatedAt,
			&p.UpdatedAt,
			&p.LastIndexedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}

		p.LocalPath = derefString(localPath)
		p.Framework = derefString(framework)
		p.LastError = derefString(lastError)
		p.IndexStatus = IndexStatus(indexStatus)

		projects = append(projects, &p)
	}

	return projects, rows.Err()
}

// UpdateProjectStats recalculates and updates project statistics.
func (s *PostgresProjectStore) UpdateProjectStats(ctx context.Context, name string) error {
	query := `
		UPDATE grepai_projects p
		SET
			file_count = COALESCE((SELECT COUNT(*) FROM grepai_files f WHERE f.project_id = p.id), 0),
			chunk_count = COALESCE((SELECT COUNT(*) FROM grepai_chunks c WHERE c.project_id = p.id::text), 0),
			updated_at = NOW()
		WHERE p.name = $1`

	result, err := s.pool.Exec(ctx, query, name)
	if err != nil {
		return fmt.Errorf("failed to update project stats: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("project not found: %s", name)
	}

	return nil
}

// SetProjectStatus updates the index status and optional error message.
func (s *PostgresProjectStore) SetProjectStatus(ctx context.Context, name string, status IndexStatus, lastError string) error {
	var query string
	var args []interface{}

	if status == IndexStatusReady {
		query = `
			UPDATE grepai_projects
			SET index_status = $2, last_error = $3, last_indexed_at = NOW(), updated_at = NOW()
			WHERE name = $1`
		args = []interface{}{name, string(status), nullableString(lastError)}
	} else {
		query = `
			UPDATE grepai_projects
			SET index_status = $2, last_error = $3, updated_at = NOW()
			WHERE name = $1`
		args = []interface{}{name, string(status), nullableString(lastError)}
	}

	result, err := s.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to set project status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("project not found: %s", name)
	}

	return nil
}

// CreateFile creates a new file record.
func (s *PostgresProjectStore) CreateFile(ctx context.Context, f *File) error {
	query := `
		INSERT INTO grepai_files (project_id, relative_path, language, content_hash, line_count, mod_time)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at`

	err := s.pool.QueryRow(ctx, query,
		f.ProjectID,
		f.RelativePath,
		nullableString(f.Language),
		f.ContentHash,
		f.LineCount,
		f.ModTime,
	).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	return nil
}

// GetFile retrieves a file by project ID and relative path.
func (s *PostgresProjectStore) GetFile(ctx context.Context, projectID, relativePath string) (*File, error) {
	query := `
		SELECT id, project_id, relative_path, language, content_hash, line_count, mod_time, created_at, updated_at
		FROM grepai_files
		WHERE project_id = $1 AND relative_path = $2`

	var f File
	var language *string

	err := s.pool.QueryRow(ctx, query, projectID, relativePath).Scan(
		&f.ID,
		&f.ProjectID,
		&f.RelativePath,
		&language,
		&f.ContentHash,
		&f.LineCount,
		&f.ModTime,
		&f.CreatedAt,
		&f.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file: %w", err)
	}

	f.Language = derefString(language)
	return &f, nil
}

// UpdateFile updates an existing file record.
func (s *PostgresProjectStore) UpdateFile(ctx context.Context, f *File) error {
	query := `
		UPDATE grepai_files
		SET language = $3, content_hash = $4, line_count = $5, mod_time = $6, updated_at = NOW()
		WHERE project_id = $1 AND relative_path = $2`

	result, err := s.pool.Exec(ctx, query,
		f.ProjectID,
		f.RelativePath,
		nullableString(f.Language),
		f.ContentHash,
		f.LineCount,
		f.ModTime,
	)

	if err != nil {
		return fmt.Errorf("failed to update file: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("file not found: %s/%s", f.ProjectID, f.RelativePath)
	}

	return nil
}

// DeleteFile removes a file record.
func (s *PostgresProjectStore) DeleteFile(ctx context.Context, projectID, relativePath string) error {
	result, err := s.pool.Exec(ctx,
		`DELETE FROM grepai_files WHERE project_id = $1 AND relative_path = $2`,
		projectID, relativePath,
	)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("file not found: %s/%s", projectID, relativePath)
	}

	return nil
}

// ListFiles returns all files for a project.
func (s *PostgresProjectStore) ListFiles(ctx context.Context, projectID string) ([]*File, error) {
	query := `
		SELECT id, project_id, relative_path, language, content_hash, line_count, mod_time, created_at, updated_at
		FROM grepai_files
		WHERE project_id = $1
		ORDER BY relative_path`

	rows, err := s.pool.Query(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer rows.Close()

	var files []*File
	for rows.Next() {
		var f File
		var language *string

		err := rows.Scan(
			&f.ID,
			&f.ProjectID,
			&f.RelativePath,
			&language,
			&f.ContentHash,
			&f.LineCount,
			&f.ModTime,
			&f.CreatedAt,
			&f.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}

		f.Language = derefString(language)
		files = append(files, &f)
	}

	return files, rows.Err()
}

// DeleteFilesByProject removes all files for a project.
func (s *PostgresProjectStore) DeleteFilesByProject(ctx context.Context, projectID string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM grepai_files WHERE project_id = $1`,
		projectID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete files: %w", err)
	}

	return nil
}

// Helper functions for nullable strings
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// GetOrCreateProject retrieves a project by name or creates it if it doesn't exist.
func (s *PostgresProjectStore) GetOrCreateProject(ctx context.Context, name, localPath string) (*Project, error) {
	// Try to get existing project first
	project, err := s.GetProject(ctx, name)
	if err != nil {
		return nil, err
	}

	if project != nil {
		// Update local path if it changed
		if project.LocalPath != localPath && localPath != "" {
			project.LocalPath = localPath
			project.UpdatedAt = time.Now()
			if err := s.UpdateProject(ctx, project); err != nil {
				return nil, err
			}
		}
		return project, nil
	}

	// Create new project
	project = &Project{
		Name:        name,
		LocalPath:   localPath,
		Languages:   []string{},
		IndexStatus: IndexStatusPending,
	}

	if err := s.CreateProject(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}
