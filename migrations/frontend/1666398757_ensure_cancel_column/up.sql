ALTER TABLE webhook_build_jobs ADD COLUMN IF NOT EXISTS cancel BOOLEAN NOT NULL DEFAULT FALSE;