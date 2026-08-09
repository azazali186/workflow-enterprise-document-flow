-- 000002_outbox_trace_id.down.sql
DROP INDEX IF EXISTS idx_outbox_messages_trace_id;
ALTER TABLE outbox_messages DROP COLUMN IF EXISTS trace_id;
