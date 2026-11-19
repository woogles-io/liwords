-- Lets IsEmailOfSuspendedAccount find accounts by a past address. The table
-- is small and rarely written, so a plain CREATE INDEX holds its lock only
-- briefly.
CREATE INDEX IF NOT EXISTS idx_email_changes_old_email_key ON email_changes (email_key(old_email));
CREATE INDEX IF NOT EXISTS idx_email_changes_new_email_key ON email_changes (email_key(new_email));
