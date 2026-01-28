-- Migration 001: Projects as central entity
-- This migration creates the projects and files tables for multi-project support.

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

-- Indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_projects_name ON grepai_projects(name);
CREATE INDEX IF NOT EXISTS idx_projects_status ON grepai_projects(index_status);
CREATE INDEX IF NOT EXISTS idx_files_project ON grepai_files(project_id);
CREATE INDEX IF NOT EXISTS idx_files_path ON grepai_files(project_id, relative_path);
CREATE INDEX IF NOT EXISTS idx_files_language ON grepai_files(language);

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
