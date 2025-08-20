-- Fix admin user password hash
-- This migration ensures the admin user exists with the correct password hash for "admin123"

-- First, try to update existing admin user with correct password hash
UPDATE users 
SET password_hash = '$2a$10$N9qo8uLOickgx2ZMRZoMye.IjPeOXe.2p5l/q/FQcre8HdkL6Q262',
    updated_at = CURRENT_TIMESTAMP
WHERE username = 'admin';

-- If admin user doesn't exist, create it
INSERT INTO users (username, email, password_hash, role, status, created_at, updated_at) 
VALUES ('admin', 'admin@qcat.local', '$2a$10$N9qo8uLOickgx2ZMRZoMye.IjPeOXe.2p5l/q/FQcre8HdkL6Q262', 'admin', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (username) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    updated_at = CURRENT_TIMESTAMP;

-- Also ensure we have a test user for development
INSERT INTO users (username, email, password_hash, role, status, created_at, updated_at) 
VALUES ('testuser', 'test@qcat.local', '$2a$10$N9qo8uLOickgx2ZMRZoMye.IjPeOXe.2p5l/q/FQcre8HdkL6Q262', 'user', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (username) DO UPDATE SET
    password_hash = EXCLUDED.password_hash,
    updated_at = CURRENT_TIMESTAMP;

-- Create a demo user with a different password (demo123)
INSERT INTO users (username, email, password_hash, role, status, created_at, updated_at) 
VALUES ('demo', 'demo@qcat.local', '$2a$10$8K1p/a0dhrxiH8Tf4di1HuP4lxvlmOyqjLxYiMyIlSaw1uYwy55jG', 'user', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (username) DO NOTHING;
