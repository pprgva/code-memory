-- Migration 002: Full schema with symbols, references, and call graph
-- This migration adds the complete schema from ROADMAP Section 7.

-- ============================================================================
-- MODIFY grepai_chunks: Add file_id reference
-- ============================================================================

-- Add file_id column if it doesn't exist
ALTER TABLE grepai_chunks
ADD COLUMN IF NOT EXISTS file_id UUID REFERENCES grepai_files(id) ON DELETE CASCADE;

-- Create index for file_id lookups
CREATE INDEX IF NOT EXISTS idx_chunks_file_id ON grepai_chunks(file_id);

-- ============================================================================
-- TABLE: grepai_symbols (functions, classes, methods, etc.)
-- ============================================================================

CREATE TABLE IF NOT EXISTS grepai_symbols (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES grepai_projects(id) ON DELETE CASCADE,
    file_id UUID NOT NULL REFERENCES grepai_files(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    kind TEXT NOT NULL,              -- function, method, class, interface, struct, etc.
    line INTEGER NOT NULL,
    end_line INTEGER,                -- End line of the symbol definition
    signature TEXT,                  -- Full function/method signature
    is_exported BOOLEAN DEFAULT false,
    doc_comment TEXT,                -- Documentation comment if present
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for grepai_symbols
CREATE INDEX IF NOT EXISTS idx_symbols_project ON grepai_symbols(project_id);
CREATE INDEX IF NOT EXISTS idx_symbols_file ON grepai_symbols(file_id);
CREATE INDEX IF NOT EXISTS idx_symbols_name ON grepai_symbols(name);
CREATE INDEX IF NOT EXISTS idx_symbols_kind ON grepai_symbols(kind);
CREATE INDEX IF NOT EXISTS idx_symbols_project_name ON grepai_symbols(project_id, name);

-- ============================================================================
-- TABLE: grepai_references (calls, usages)
-- ============================================================================

CREATE TABLE IF NOT EXISTS grepai_references (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES grepai_projects(id) ON DELETE CASCADE,
    file_id UUID NOT NULL REFERENCES grepai_files(id) ON DELETE CASCADE,
    symbol_name TEXT NOT NULL,       -- Name of the symbol being referenced
    symbol_id UUID REFERENCES grepai_symbols(id) ON DELETE SET NULL,
    caller_symbol_id UUID REFERENCES grepai_symbols(id) ON DELETE SET NULL,
    caller_name TEXT,                -- Name of the calling function/method
    line INTEGER NOT NULL,
    column_start INTEGER,            -- Column where reference starts
    column_end INTEGER,              -- Column where reference ends
    context TEXT,                    -- Line of code containing the reference
    ref_type TEXT DEFAULT 'call',    -- call, import, type_usage, field_access
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for grepai_references
CREATE INDEX IF NOT EXISTS idx_references_project ON grepai_references(project_id);
CREATE INDEX IF NOT EXISTS idx_references_file ON grepai_references(file_id);
CREATE INDEX IF NOT EXISTS idx_references_symbol_name ON grepai_references(symbol_name);
CREATE INDEX IF NOT EXISTS idx_references_symbol_id ON grepai_references(symbol_id);
CREATE INDEX IF NOT EXISTS idx_references_caller_id ON grepai_references(caller_symbol_id);
CREATE INDEX IF NOT EXISTS idx_references_project_symbol ON grepai_references(project_id, symbol_name);

-- ============================================================================
-- TABLE: grepai_call_edges (pre-computed call graph)
-- ============================================================================

CREATE TABLE IF NOT EXISTS grepai_call_edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES grepai_projects(id) ON DELETE CASCADE,
    caller_id UUID REFERENCES grepai_symbols(id) ON DELETE CASCADE,
    callee_id UUID REFERENCES grepai_symbols(id) ON DELETE CASCADE,
    caller_name TEXT NOT NULL,       -- Denormalized for fast queries
    callee_name TEXT NOT NULL,       -- Denormalized for fast queries
    file_id UUID REFERENCES grepai_files(id) ON DELETE CASCADE,
    line INTEGER,                    -- Line where the call occurs
    call_type TEXT DEFAULT 'direct', -- direct, method, closure, defer, go
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(project_id, caller_id, callee_id, line)
);

-- Indexes for grepai_call_edges
CREATE INDEX IF NOT EXISTS idx_call_edges_project ON grepai_call_edges(project_id);
CREATE INDEX IF NOT EXISTS idx_call_edges_caller_id ON grepai_call_edges(caller_id);
CREATE INDEX IF NOT EXISTS idx_call_edges_callee_id ON grepai_call_edges(callee_id);
CREATE INDEX IF NOT EXISTS idx_call_edges_caller_name ON grepai_call_edges(caller_name);
CREATE INDEX IF NOT EXISTS idx_call_edges_callee_name ON grepai_call_edges(callee_name);
CREATE INDEX IF NOT EXISTS idx_call_edges_project_caller ON grepai_call_edges(project_id, caller_name);
CREATE INDEX IF NOT EXISTS idx_call_edges_project_callee ON grepai_call_edges(project_id, callee_name);

-- ============================================================================
-- Record this migration
-- ============================================================================

INSERT INTO grepai_migrations (version, name)
VALUES (2, '002_full_schema')
ON CONFLICT (version) DO NOTHING;
