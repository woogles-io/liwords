BEGIN; 

CREATE TABLE IF NOT EXISTS game_comments (
    id UUID UNIQUE NOT NULL,
    game_id bigint NOT NULL,
    author_id integer NOT NULL, 
    event_number integer NOT NULL, -- the game event number that the comment is associated with
    created_at timestamptz NOT NULL DEFAULT NOW(),
    edited_at timestamptz NOT NULL DEFAULT NOW(),
    comment TEXT NOT NULL,
    FOREIGN KEY (game_id) REFERENCES games (id),
    FOREIGN KEY (author_id) REFERENCES users (id)
);

COMMIT;