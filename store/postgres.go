package store

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type PostgresStore struct {
	pool       *pgxpool.Pool
	projectID  string
	dimensions int
}

func NewPostgresStore(ctx context.Context, dsn string, projectUUID string, vectorDimensions int) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Run migrations first
	if err := RunMigrations(ctx, pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	store := &PostgresStore{
		pool:       pool,
		projectID:  projectUUID,
		dimensions: vectorDimensions,
	}

	if err := store.ensureSchema(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return store, nil
}

func (s *PostgresStore) ensureSchema(ctx context.Context) error {
	// Create vector extension (required for pgvector)
	if _, err := s.pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS vector`); err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	// Adjust vector dimension if needed (tables are created by migrations)
	if _, err := s.pool.Exec(ctx, buildEnsureVectorSQL(s.dimensions)); err != nil {
		return fmt.Errorf("failed to adjust vector dimension: %w", err)
	}

	return nil
}

func (s *PostgresStore) SaveChunks(ctx context.Context, chunks []Chunk) error {
	batch := &pgx.Batch{}

	for _, chunk := range chunks {
		vec := pgvector.NewVector(chunk.Vector)
		content := chunk.Content
		if !utf8.ValidString(content) {
			content = strings.ToValidUTF8(content, "\uFFFD")
		}
		batch.Queue(
			`INSERT INTO grepai_chunks (id, project_id, file_path, start_line, end_line, content, vector, hash, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO UPDATE SET
				file_path = EXCLUDED.file_path,
				start_line = EXCLUDED.start_line,
				end_line = EXCLUDED.end_line,
				content = EXCLUDED.content,
				vector = EXCLUDED.vector,
				hash = EXCLUDED.hash,
				updated_at = EXCLUDED.updated_at`,
			chunk.ID, s.projectID, chunk.FilePath, chunk.StartLine, chunk.EndLine,
			content, vec, chunk.Hash, chunk.UpdatedAt,
		)
	}

	results := s.pool.SendBatch(ctx, batch)
	defer results.Close()

	for range chunks {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("failed to save chunk: %w", err)
		}
	}

	return nil
}

func (s *PostgresStore) DeleteByFile(ctx context.Context, filePath string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM grepai_chunks WHERE project_id = $1 AND file_path = $2`,
		s.projectID, filePath,
	)
	if err != nil {
		return fmt.Errorf("failed to delete chunks: %w", err)
	}
	return nil
}

func (s *PostgresStore) Search(ctx context.Context, queryVector []float32, limit int) ([]SearchResult, error) {
	vec := pgvector.NewVector(queryVector)

	rows, err := s.pool.Query(ctx,
		`SELECT id, file_path, start_line, end_line, content, vector, hash, updated_at,
			1 - (vector <=> $1) as score
		FROM grepai_chunks
		WHERE project_id = $2
		ORDER BY vector <=> $1
		LIMIT $3`,
		vec, s.projectID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var chunk Chunk
		var vec pgvector.Vector
		var score float32

		if err := rows.Scan(
			&chunk.ID, &chunk.FilePath, &chunk.StartLine, &chunk.EndLine,
			&chunk.Content, &vec, &chunk.Hash, &chunk.UpdatedAt, &score,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		chunk.Vector = vec.Slice()
		results = append(results, SearchResult{
			Chunk: chunk,
			Score: score,
		})
	}

	return results, rows.Err()
}

func (s *PostgresStore) GetDocument(ctx context.Context, filePath string) (*Document, error) {
	var doc Document
	var modTime time.Time

	err := s.pool.QueryRow(ctx,
		`SELECT path, hash, mod_time, chunk_ids FROM grepai_documents WHERE project_id = $1 AND path = $2`,
		s.projectID, filePath,
	).Scan(&doc.Path, &doc.Hash, &modTime, &doc.ChunkIDs)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	doc.ModTime = modTime
	return &doc, nil
}

func (s *PostgresStore) SaveDocument(ctx context.Context, doc Document) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO grepai_documents (path, project_id, hash, mod_time, chunk_ids)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (project_id, path) DO UPDATE SET
			hash = EXCLUDED.hash,
			mod_time = EXCLUDED.mod_time,
			chunk_ids = EXCLUDED.chunk_ids`,
		doc.Path, s.projectID, doc.Hash, doc.ModTime, doc.ChunkIDs,
	)
	if err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}
	return nil
}

func (s *PostgresStore) DeleteDocument(ctx context.Context, filePath string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM grepai_documents WHERE project_id = $1 AND path = $2`,
		s.projectID, filePath,
	)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	return nil
}

func (s *PostgresStore) ListDocuments(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT path FROM grepai_documents WHERE project_id = $1`,
		s.projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("failed to scan path: %w", err)
		}
		paths = append(paths, path)
	}

	return paths, rows.Err()
}

func (s *PostgresStore) Load(ctx context.Context) error {
	// No-op for Postgres, data is already persistent
	return nil
}

func (s *PostgresStore) Persist(ctx context.Context) error {
	// No-op for Postgres, data is already persistent
	return nil
}

func (s *PostgresStore) Close() error {
	s.pool.Close()
	return nil
}

func (s *PostgresStore) GetStats(ctx context.Context) (*IndexStats, error) {
	var stats IndexStats

	// Get file count
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM grepai_documents WHERE project_id = $1`,
		s.projectID,
	).Scan(&stats.TotalFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}

	// Get chunk count and last updated
	err = s.pool.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(MAX(updated_at), '1970-01-01'::timestamp) FROM grepai_chunks WHERE project_id = $1`,
		s.projectID,
	).Scan(&stats.TotalChunks, &stats.LastUpdated)
	if err != nil {
		return nil, fmt.Errorf("failed to count chunks: %w", err)
	}

	// IndexSize not applicable for Postgres (data stored remotely)
	stats.IndexSize = 0

	return &stats, nil
}

func (s *PostgresStore) ListFilesWithStats(ctx context.Context) ([]FileStats, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT path, mod_time, array_length(chunk_ids, 1) FROM grepai_documents WHERE project_id = $1`,
		s.projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}
	defer rows.Close()

	var files []FileStats
	for rows.Next() {
		var f FileStats
		var chunkCount *int
		if err := rows.Scan(&f.Path, &f.ModTime, &chunkCount); err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		if chunkCount != nil {
			f.ChunkCount = *chunkCount
		}
		files = append(files, f)
	}

	return files, rows.Err()
}

func (s *PostgresStore) GetChunksForFile(ctx context.Context, filePath string) ([]Chunk, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, file_path, start_line, end_line, content, hash, updated_at
		FROM grepai_chunks WHERE project_id = $1 AND file_path = $2
		ORDER BY start_line`,
		s.projectID, filePath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get chunks: %w", err)
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.ID, &c.FilePath, &c.StartLine, &c.EndLine, &c.Content, &c.Hash, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}
		chunks = append(chunks, c)
	}

	return chunks, rows.Err()
}

func (s *PostgresStore) GetAllChunks(ctx context.Context) ([]Chunk, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, file_path, start_line, end_line, content, hash, updated_at
		FROM grepai_chunks WHERE project_id = $1`,
		s.projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get all chunks: %w", err)
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.ID, &c.FilePath, &c.StartLine, &c.EndLine, &c.Content, &c.Hash, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan chunk: %w", err)
		}
		chunks = append(chunks, c)
	}

	return chunks, rows.Err()
}

// buildEnsureVectorSQL returns a SQL block that alters the "grepai_chunks.vector" column
// only if its current dimension differs from the specified one.
func buildEnsureVectorSQL(dim int) string {
	return fmt.Sprintf(`
DO $$
DECLARE
	current_length int;
BEGIN
	SELECT atttypmod - 4
	INTO current_length
	FROM pg_attribute
	WHERE attrelid = 'grepai_chunks'::regclass
	  AND attname = 'vector';

	IF current_length IS DISTINCT FROM %d THEN
		RAISE NOTICE 'Altering vector size from %% to %d', current_length;
		EXECUTE 'ALTER TABLE grepai_chunks ALTER COLUMN vector TYPE vector(%d)';
	ELSE
		RAISE NOTICE 'Vector size already %d, skipping ALTER';
	END IF;
END$$;
`, dim, dim, dim, dim)
}

// Pool returns the underlying connection pool for use by other stores.
func (s *PostgresStore) Pool() *pgxpool.Pool {
	return s.pool
}

// NewPostgresPool creates a new connection pool without initializing a full store.
// This is useful for operations that need database access before store creation.
func NewPostgresPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}
	return pool, nil
}

// GetOrCreateProjectWithPool finds or creates a project using a raw pool connection.
// This is useful when you need to register a project before creating a PostgresStore.
// If projectID is provided, it will be used for new projects instead of generating one.
func GetOrCreateProjectWithPool(ctx context.Context, pool *pgxpool.Pool, name string, localPath string, projectID string) (string, error) {
	var projectUUID string
	var existingPath *string

	err := pool.QueryRow(ctx,
		`SELECT id, local_path FROM grepai_projects WHERE name = $1`,
		name,
	).Scan(&projectUUID, &existingPath)

	if err == pgx.ErrNoRows {
		if projectID != "" {
			// Use provided UUID
			_, err = pool.Exec(ctx,
				`INSERT INTO grepai_projects (id, name, local_path, index_status)
				VALUES ($1, $2, $3, 'pending')`,
				projectID, name, localPath,
			)
			if err != nil {
				return "", fmt.Errorf("failed to create project with ID: %w", err)
			}
			return projectID, nil
		}
		// Generate new UUID
		err = pool.QueryRow(ctx,
			`INSERT INTO grepai_projects (name, local_path, index_status)
			VALUES ($1, $2, 'pending')
			RETURNING id`,
			name, localPath,
		).Scan(&projectUUID)
		if err != nil {
			return "", fmt.Errorf("failed to create project: %w", err)
		}
		return projectUUID, nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to query project: %w", err)
	}

	if existingPath == nil || *existingPath != localPath {
		_, err = pool.Exec(ctx,
			`UPDATE grepai_projects SET local_path = $1, updated_at = NOW() WHERE id = $2`,
			localPath, projectUUID,
		)
		if err != nil {
			return "", fmt.Errorf("failed to update project path: %w", err)
		}
	}

	return projectUUID, nil
}

// GetOrCreateProject finds a project by name or creates it if not found.
// If the project exists but local_path differs, it updates the path.
// If projectID is provided, it will be used for new projects instead of generating one.
// Returns the project UUID.
func (s *PostgresStore) GetOrCreateProject(ctx context.Context, name string, localPath string, projectID string) (string, error) {
	var projectUUID string
	var existingPath *string

	// Try to find existing project by name
	err := s.pool.QueryRow(ctx,
		`SELECT id, local_path FROM grepai_projects WHERE name = $1`,
		name,
	).Scan(&projectUUID, &existingPath)

	if err == pgx.ErrNoRows {
		// Project doesn't exist, create it
		if projectID != "" {
			// Use provided UUID
			_, err = s.pool.Exec(ctx,
				`INSERT INTO grepai_projects (id, name, local_path, index_status)
				VALUES ($1, $2, $3, 'pending')`,
				projectID, name, localPath,
			)
			if err != nil {
				return "", fmt.Errorf("failed to create project with ID: %w", err)
			}
			return projectID, nil
		}
		// Generate new UUID
		err = s.pool.QueryRow(ctx,
			`INSERT INTO grepai_projects (name, local_path, index_status)
			VALUES ($1, $2, 'pending')
			RETURNING id`,
			name, localPath,
		).Scan(&projectUUID)
		if err != nil {
			return "", fmt.Errorf("failed to create project: %w", err)
		}
		return projectUUID, nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to query project: %w", err)
	}

	// Project exists, update local_path if different
	if existingPath == nil || *existingPath != localPath {
		_, err = s.pool.Exec(ctx,
			`UPDATE grepai_projects SET local_path = $1, updated_at = NOW() WHERE id = $2`,
			localPath, projectUUID,
		)
		if err != nil {
			return "", fmt.Errorf("failed to update project path: %w", err)
		}
	}

	return projectUUID, nil
}

// UpdateProjectStats updates file_count, chunk_count, and sets status to 'ready'.
func (s *PostgresStore) UpdateProjectStats(ctx context.Context, projectUUID string) error {
	// Note: grepai_chunks/documents.project_id is TEXT, grepai_projects.id is UUID
	// So we cast appropriately in the query
	_, err := s.pool.Exec(ctx, `
		UPDATE grepai_projects SET
			file_count = (SELECT COUNT(*) FROM grepai_documents WHERE project_id = $1::text),
			chunk_count = (SELECT COUNT(*) FROM grepai_chunks WHERE project_id = $1::text),
			index_status = 'ready',
			updated_at = NOW()
		WHERE id = $1::uuid`,
		projectUUID,
	)
	if err != nil {
		return fmt.Errorf("failed to update project stats: %w", err)
	}
	return nil
}

// RunMigrations executes all pending SQL migrations in order.
// Migrations are embedded from the migrations/ directory.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Ensure migrations tracking table exists
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS grepai_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMPTZ DEFAULT NOW()
		)`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get list of applied migrations
	rows, err := pool.Query(ctx, `SELECT version FROM grepai_migrations`)
	if err != nil {
		return fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return fmt.Errorf("failed to scan migration version: %w", err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to iterate migrations: %w", err)
	}

	// Read migration files from embedded filesystem
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Sort migration files by name to ensure order
	var migrationFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			migrationFiles = append(migrationFiles, entry.Name())
		}
	}
	sort.Strings(migrationFiles)

	// Execute pending migrations
	for _, filename := range migrationFiles {
		// Extract version number from filename (e.g., "001_projects.sql" -> 1)
		var version int
		if _, err := fmt.Sscanf(filename, "%03d_", &version); err != nil {
			continue // Skip files that don't match the pattern
		}

		if applied[version] {
			continue // Skip already applied migrations
		}

		// Read and execute migration
		content, err := migrationsFS.ReadFile("migrations/" + filename)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", filename, err)
		}

		// Execute migration in a transaction
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", filename, err)
		}

		_, err = tx.Exec(ctx, string(content))
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("failed to execute migration %s: %w", filename, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", filename, err)
		}
	}

	return nil
}

