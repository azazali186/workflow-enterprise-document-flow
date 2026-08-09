-- 000002_outbox_trace_id.up.sql
-- Carry the originating HTTP request id on outbox rows so the worker can
-- correlate event processing back to the API call that produced it.
ALTER TABLE outbox_messages ADD COLUMN IF NOT EXISTS trace_id varchar(64);
CREATE INDEX IF NOT EXISTS idx_outbox_messages_trace_id ON outbox_messages (trace_id);
