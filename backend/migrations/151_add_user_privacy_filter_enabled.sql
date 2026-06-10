-- Add per-user privacy filter opt-in switch.
ALTER TABLE users ADD COLUMN IF NOT EXISTS privacy_filter_enabled BOOLEAN NOT NULL DEFAULT FALSE;
