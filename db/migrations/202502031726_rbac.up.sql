BEGIN;

CREATE TABLE permissions (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,  -- e.g., "can_create_tournaments"
    description TEXT NOT NULL
);

CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,  -- e.g., "Tournament Organizer"
    description TEXT NOT NULL
);

-- Junction table linking roles to permissions
CREATE TABLE role_permissions (
    role_id INT REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INT REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- Junction table linking users to roles
CREATE TABLE user_roles (
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    role_id INT REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);


INSERT INTO permissions (code, description)
VALUES
    ('admin_all_access', 'Bypasses all permission checks'),
    ('can_create_tournaments', 'Create new tournaments'),
    ('can_bypass_elitebot_paywall', 'Can play EliteBot without subscription'),
    ('can_moderate_users', 'Can moderate users (mute, some bans)'),
    ('can_modify_announcements', 'Can modify announcements on the homepage'),
    ('can_draw_more_blanks_than_average', 'This is just a joke, guys, dont at me'),
    ('can_manage_tournaments', 'Can manage (start, assign pairings, etc) any tournaments'),
    ('can_create_puzzles', 'Can run puzzle generation / etc jobs'),
    ('can_reset_and_delete_accounts', 'Can reset and delete accounts');

INSERT INTO roles(name, description)
VALUES
    ('Admin', 'Site administrator - has all access'),
    ('Moderator', 'Site moderator'),
    ('Tournament Creator', 'Tournament creator'),
    ('Special Access Player', 'Player with special access to some features'),
    ('Tournament Manager', 'Tournament manager');

INSERT INTO role_permissions (role_id, permission_id)
VALUES
    ((SELECT id FROM roles WHERE name = 'Admin'),
     (SELECT id FROM permissions WHERE code = 'admin_all_access')),
    ((SELECT id FROM roles WHERE name = 'Moderator'),
     (SELECT id FROM permissions WHERE code = 'can_moderate_users')),
    ((SELECT id FROM roles WHERE name = 'Moderator'),
     (SELECT id FROM permissions WHERE code = 'can_modify_announcements')),
    ((SELECT id FROM roles WHERE name = 'Tournament Creator'),
     (SELECT id FROM permissions WHERE code = 'can_create_tournaments')),
    ((SELECT id FROM roles WHERE name = 'Special Access Player'),
     (SELECT id FROM permissions WHERE code = 'can_bypass_elitebot_paywall')),
    ((SELECT id FROM roles WHERE name = 'Tournament Manager'),
     (SELECT id FROM permissions WHERE code = 'can_manage_tournaments'));

COMMIT;