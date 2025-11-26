-- Drop the partial unique index
DROP INDEX unique_pending_verification;

-- Restore the old constraint (though this is not ideal, it maintains backward compatibility)
ALTER TABLE verification_requests ADD CONSTRAINT unique_pending_verification UNIQUE (user_id, integration_name, status);
