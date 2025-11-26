-- Drop the old constraint that applies to all statuses
ALTER TABLE verification_requests DROP CONSTRAINT unique_pending_verification;

-- Create a partial unique index that only applies to pending requests
-- This allows multiple rejected/approved requests for the same user+integration,
-- but still prevents multiple pending requests
CREATE UNIQUE INDEX unique_pending_verification ON verification_requests (user_id, integration_name) WHERE status = 'pending';
