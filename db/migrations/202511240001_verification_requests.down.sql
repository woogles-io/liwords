-- Remove role permissions
DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE name = 'user_verifier');

-- Remove role
DELETE FROM roles WHERE name = 'user_verifier';

-- Remove permission
DELETE FROM permissions WHERE code = 'can_verify_user_identities';

-- Drop indexes
DROP INDEX IF EXISTS idx_verification_requests_user_id;
DROP INDEX IF EXISTS idx_verification_requests_status;

-- Drop table
DROP TABLE IF EXISTS verification_requests;
