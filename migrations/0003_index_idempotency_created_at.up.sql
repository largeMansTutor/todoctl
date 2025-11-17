-- Add index to support ordering/cleanup by creation time for idempotency keys.
ALTER TABLE idempotency_keys
    ADD INDEX idx_idempotency_created_at (created_at);
