-- Add player_on_turn column to track which player's turn it is
ALTER TABLE games ADD COLUMN player_on_turn INT DEFAULT NULL;

-- For ongoing games where player0 went first (typical case), set to calculate from history
-- We'll let the backend populate this on next update, so leaving NULL for now
