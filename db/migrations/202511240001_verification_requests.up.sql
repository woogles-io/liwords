-- Create verification_requests table for manual identity verification
CREATE TABLE verification_requests (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    integration_name TEXT NOT NULL,
    member_id TEXT NOT NULL,
    full_name TEXT NOT NULL,
    image_url TEXT NOT NULL,
    title TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    submitted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    notes TEXT,
    CONSTRAINT unique_pending_verification UNIQUE (user_id, integration_name, status)
);

-- Add index for efficient querying of pending verifications
CREATE INDEX idx_verification_requests_status ON verification_requests(status, submitted_at);
CREATE INDEX idx_verification_requests_user_id ON verification_requests(user_id);

-- Add can_verify_user_identities permission
INSERT INTO permissions (code, description)
VALUES ('can_verify_user_identities', 'Can verify user identity claims for organization memberships');

-- Create user_verifier role with this permission
INSERT INTO roles (name, description)
VALUES ('user_verifier', 'Can verify user identity claims');

-- Link the permission to the role
INSERT INTO role_permissions (role_id, permission_id)
SELECT
    (SELECT id FROM roles WHERE name = 'user_verifier'),
    (SELECT id FROM permissions WHERE code = 'can_verify_user_identities');
