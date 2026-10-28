ALTER TABLE users ADD COLUMN deleted_at timestamptz;
ALTER TABLE users ADD COLUMN settings jsonb NOT NULL DEFAULT '{"audio":true,"controls":"both","reducedMotion":false}';
