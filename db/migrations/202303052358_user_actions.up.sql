BEGIN; 

CREATE TABLE IF NOT EXISTS user_actions (
    id BIGSERIAL PRIMARY KEY,
    user_id bigint NOT NULL,
    action_type int NOT NULL,
    start_time timestamptz NOT NULL DEFAULT NOW(),
    end_time timestamptz,
    removed_time timestamptz,
    message_id text,
    applier_id bigint NOT NULL,
    remover_id bigint,
    chat_text text,
    note text,
    removal_note text,
    email_type int NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users (id),
    FOREIGN KEY (applier_id) REFERENCES users (id),
    FOREIGN KEY (remover_id) REFERENCES users (id),
    UNIQUE(user_id, start_time, action_type)
);

COMMIT;