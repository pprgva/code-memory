-- Migration 001: Core schema with projects as central entity
-- This migration creates all core tables for grepai.

-- Enable vector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Projects table (central entity)
CREATE TABLE IF NOT EXISTS grepai_projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    local_path TEXT,
    languages TEXT[] DEFAULT '{}',
    framework TEXT,
    file_count INTEGER DEFAULT 0,
    chunk_count INTEGER DEFAULT 0,
    symbol_count INTEGER DEFAULT 0,
    index_status TEXT DEFAULT 'pending',
    last_error TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_indexed_at TIMESTAMPTZ
);

-- Files table (relative paths)
CREATE TABLE IF NOT EXISTS grepai_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES grepai_projects(id) ON DELETE CASCADE,
    relative_path TEXT NOT NULL,
    language TEXT,
    content_hash TEXT NOT NULL,
    line_count INTEGER,
    mod_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(project_id, relative_path)
);

-- Chunks table (code chunks with embeddings)
CREATE TABLE IF NOT EXISTS grepai_chunks (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    file_path TEXT NOT NULL,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    content TEXT NOT NULL,
    vector vector(1024),
    hash TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- Documents table (file metadata)
CREATE TABLE IF NOT EXISTS grepai_documents (
    path TEXT NOT NULL,
    project_id TEXT NOT NULL,
    hash TEXT NOT NULL,
    mod_time TIMESTAMP NOT NULL,
    chunk_ids TEXT[] NOT NULL,
    PRIMARY KEY (project_id, path)
);

-- Indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_projects_name ON grepai_projects(name);
CREATE INDEX IF NOT EXISTS idx_projects_status ON grepai_projects(index_status);
CREATE INDEX IF NOT EXISTS idx_files_project ON grepai_files(project_id);
CREATE INDEX IF NOT EXISTS idx_files_path ON grepai_files(project_id, relative_path);
CREATE INDEX IF NOT EXISTS idx_files_language ON grepai_files(language);
CREATE INDEX IF NOT EXISTS idx_grepai_chunks_project ON grepai_chunks(project_id);
CREATE INDEX IF NOT EXISTS idx_grepai_chunks_file ON grepai_chunks(project_id, file_path);

-- Migration tracking table
CREATE TABLE IF NOT EXISTS grepai_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TIMESTAMPTZ DEFAULT NOW()
);

-- Record this migration
INSERT INTO grepai_migrations (version, name)
VALUES (1, '001_projects')
ON CONFLICT (version) DO NOTHING;
