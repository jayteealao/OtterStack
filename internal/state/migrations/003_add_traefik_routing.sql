-- Add Traefik priority-based routing support
-- Migration: 003_add_traefik_routing
-- Created: 2026-01-09

BEGIN TRANSACTION;

-- Add column to enable/disable Traefik routing per project
ALTER TABLE projects ADD COLUMN traefik_routing_enabled BOOLEAN NOT NULL DEFAULT 0;

-- Update schema version
INSERT INTO schema_migrations (version) VALUES (3);

COMMIT;
