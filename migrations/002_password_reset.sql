-- Password reset tokens
ALTER TABLE users ADD COLUMN reset_token_hash TEXT;
ALTER TABLE users ADD COLUMN reset_token_expires_at TEXT;

-- Email verification tokens
ALTER TABLE users ADD COLUMN email_verification_token_hash TEXT;
ALTER TABLE users ADD COLUMN email_verification_token_expires_at TEXT;
