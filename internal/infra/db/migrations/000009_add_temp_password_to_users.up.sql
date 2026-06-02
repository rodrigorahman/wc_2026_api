ALTER TABLE users ADD COLUMN temp_password_hash TEXT;
ALTER TABLE users ADD COLUMN temp_password_expires_at TIMESTAMP;
