-- OtterStack initial schema
-- Migration: 001_initial
-- Created: 2026-01-08

-- Schema version tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Projects table
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    repo_type TEXT NOT NULL CHECK(repo_type IN ('local', 'remote')),
    repo_url TEXT,  -- nullable, only for remote repos
    repo_path TEXT NOT NULL,  -- local path or managed clone path
    compose_file TEXT NOT NULL DEFAULT 'compose.yaml',
    worktree_retention INTEGER NOT NULL DEFAULT 3,
    status TEXT NOT NULL DEFAULT 'ready' CHECK(status IN ('unconfigured', 'cloning', 'ready', 'clone_failed')),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Deployments table
CREATE TABLE IF NOT EXISTS deployments (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    git_sha TEXT NOT NULL,
    git_ref TEXT,  -- nullable, original tag/branch if any
    worktree_path TEXT,
    status TEXT NOT NULL DEFAULT 'deploying' CHECK(status IN ('deploying', 'active', 'inactive', 'failed', 'rolled_back', 'interrupted')),
    error_message TEXT,
    started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    finished_at DATETIME,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Indexes for deployments table
CREATE INDEX IF NOT EXISTS idx_deployments_project_status ON deployments(project_id, status);
CREATE INDEX IF NOT EXISTS idx_deployments_project_started ON deployments(project_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_deployments_git_sha ON deployments(project_id, git_sha);

-- Operation logs table
CREATE TABLE IF NOT EXISTS operation_logs (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    deployment_id TEXT,  -- nullable
    operation TEXT NOT NULL CHECK(operation IN ('clone', 'fetch', 'deploy', 'rollback', 'remove', 'cleanup')),
    status TEXT NOT NULL DEFAULT 'running' CHECK(status IN ('running', 'success', 'failed')),
    log_path TEXT,
    started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    finished_at DATETIME,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (deployment_id) REFERENCES deployments(id) ON DELETE SET NULL
);

-- Index for operation logs
CREATE INDEX IF NOT EXISTS idx_operation_logs_project ON operation_logs(project_id, started_at DESC);

-- Triggers to update updated_at timestamp
CREATE TRIGGER IF NOT EXISTS update_projects_timestamp
    AFTER UPDATE ON projects
    FOR EACH ROW
BEGIN
    UPDATE projects SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;

-- Insert initial migration version
INSERT OR IGNORE INTO schema_migrations (version) VALUES (1);
