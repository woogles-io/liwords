BEGIN;

CREATE TABLE IF NOT EXISTS broadcasts (
    id SERIAL PRIMARY KEY,
    uuid UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    slug TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT,

    -- Data feed (format-agnostic)
    broadcast_url TEXT NOT NULL,
    broadcast_url_format TEXT NOT NULL DEFAULT 'tsh_newt_json',
    poll_interval_seconds INTEGER NOT NULL DEFAULT 120,
    poll_start_time TIMESTAMPTZ,
    poll_end_time TIMESTAMPTZ,

    -- Game defaults
    lexicon TEXT NOT NULL DEFAULT 'CSW24',
    board_layout TEXT NOT NULL DEFAULT 'CrosswordGame',
    letter_distribution TEXT NOT NULL DEFAULT 'english',
    challenge_rule INTEGER NOT NULL DEFAULT 0,

    last_polled_at TIMESTAMPTZ,

    -- Ownership
    creator_id INTEGER NOT NULL REFERENCES users(id),

    -- Status
    active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS broadcast_games (
    id SERIAL PRIMARY KEY,
    broadcast_id INTEGER NOT NULL REFERENCES broadcasts(id) ON DELETE CASCADE,
    game_uuid TEXT NOT NULL,
    division TEXT NOT NULL DEFAULT '',
    round INTEGER NOT NULL,
    table_number INTEGER NOT NULL,
    player1_name TEXT NOT NULL,
    player2_name TEXT NOT NULL,
    annotator_user_id INTEGER REFERENCES users(id),
    claimed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(broadcast_id, division, round, table_number),
    UNIQUE(game_uuid)
);

CREATE TABLE IF NOT EXISTS broadcast_annotators (
    broadcast_id INTEGER NOT NULL REFERENCES broadcasts(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id),
    PRIMARY KEY (broadcast_id, user_id)
);

CREATE TABLE IF NOT EXISTS broadcast_directors (
    broadcast_id INTEGER NOT NULL REFERENCES broadcasts(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id),
    PRIMARY KEY (broadcast_id, user_id)
);

CREATE INDEX idx_broadcasts_active ON broadcasts(active) WHERE active = true;
CREATE INDEX idx_broadcasts_slug ON broadcasts(slug);
CREATE INDEX idx_broadcast_games_broadcast_round ON broadcast_games(broadcast_id, division, round);

INSERT INTO permissions (code, description)
VALUES ('can_create_broadcasts', 'Can create and configure broadcast events.');

INSERT INTO roles (name, description)
VALUES ('Broadcast Creator', 'Can create and manage live tournament broadcast events.');

INSERT INTO role_permissions (role_id, permission_id)
VALUES
    ((SELECT id FROM roles WHERE name = 'Admin'),
     (SELECT id FROM permissions WHERE code = 'can_create_broadcasts')),
    ((SELECT id FROM roles WHERE name = 'Broadcast Creator'),
     (SELECT id FROM permissions WHERE code = 'can_create_broadcasts'));

COMMIT;
