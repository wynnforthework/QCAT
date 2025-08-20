-- Rollback admin password fix
-- Remove the test users created in this migration

DELETE FROM users WHERE username IN ('testuser', 'demo');

-- Note: We don't rollback the admin password hash change as it might break existing functionality
-- If you need to rollback the admin password, you would need to manually update it
