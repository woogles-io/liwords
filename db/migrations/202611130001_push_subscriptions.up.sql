BEGIN;

-- One row per browser that asked for web push turn notifications. A push
-- service gives each browser its own endpoint, so an endpoint belongs to
-- whoever subscribed it last. updated_at is when it was last subscribed.
CREATE TABLE push_subscriptions (
    endpoint text PRIMARY KEY,
    user_id integer NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    p256dh text NOT NULL,
    auth text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX push_subscriptions_user_id_idx ON push_subscriptions (user_id, updated_at);

COMMIT;
