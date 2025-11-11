BEGIN;

-- Add new permissions
INSERT INTO permissions (code, description)
VALUES
    ('can_play_leagues', 'Can register for and play in leagues.'),
    ('can_invite_to_leagues', 'Can invite other users to play in leagues.');

-- Add new roles
INSERT INTO roles (name, description)
VALUES
    ('League Player', 'Players who can register for and participate in leagues.'),
    ('League Promoter', 'Users who can invite others to play in leagues.');

-- Associate permissions with roles
INSERT INTO role_permissions (role_id, permission_id)
VALUES
    -- League Player role gets can_play_leagues permission
    ((SELECT id FROM roles WHERE name = 'League Player'),
     (SELECT id FROM permissions WHERE code = 'can_play_leagues')),

    -- League Promoter role gets can_invite_to_leagues permission
    ((SELECT id FROM roles WHERE name = 'League Promoter'),
     (SELECT id FROM permissions WHERE code = 'can_invite_to_leagues')),

    -- League Promoter also gets can_play_leagues (promoters can also play)
    ((SELECT id FROM roles WHERE name = 'League Promoter'),
     (SELECT id FROM permissions WHERE code = 'can_play_leagues'));

COMMIT;
