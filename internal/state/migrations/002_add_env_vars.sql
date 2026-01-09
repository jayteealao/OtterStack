-- Migration 002: Add env_vars column to projects table
-- Stores environment variables as JSON blob

ALTER TABLE projects ADD COLUMN env_vars TEXT DEFAULT '{}';

INSERT OR IGNORE INTO schema_migrations (version) VALUES (2);
