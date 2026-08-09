package database

import (
	"time"

	"github.com/aeroxe/docu-flow/backend/internal/config"
	"github.com/nats-io/nats.go"
)

// NC is the NATS connection.
var NC *nats.Conn

// JS is the JetStream context used by the outbox dispatcher.
var JS nats.JetStreamContext

// StreamName is the JetStream stream holding all docuflow events.
const StreamName = "DOCUFLOW"

// DLQStreamName holds events the worker gave up on (dead-letter).
const DLQStreamName = "DOCUFLOW_DLQ"

// eventMaxAge is the retention for the event streams (7 days for the main
// stream, 14 for the dead-letter stream so operators have time to replay).
const eventMaxAge = 7 * 24 * time.Hour

// InitNATS connects and ensures the event streams exist (main + dead-letter).
func InitNATS(cfg *config.Config) error {
	nc, err := nats.Connect(cfg.NATSURL,
		nats.MaxReconnects(10),
		nats.ReconnectWait(nats.DefaultReconnectWait),
	)
	if err != nil {
		return err
	}
	js, err := nc.JetStream()
	if err != nil {
		return err
	}
	if err := ensureStream(js, StreamName, []string{"docuflow.>"}, eventMaxAge); err != nil {
		return err
	}
	// The dead-letter namespace must be disjoint from the main stream's
	// "docuflow.>" — JetStream rejects overlapping stream subjects, and a
	// disjoint prefix also guarantees the worker's own "docuflow.>"
	// subscription can never re-consume parked DLQ messages (feedback loop).
	if err := ensureStream(js, DLQStreamName, []string{"dlq.docuflow.>"}, 2*eventMaxAge); err != nil {
		return err
	}
	NC = nc
	JS = js
	return nil
}

// ensureStream creates a JetStream stream when it does not already exist.
func ensureStream(js nats.JetStreamContext, name string, subjects []string, maxAge time.Duration) error {
	_, err := js.AddStream(&nats.StreamConfig{
		Name:      name,
		Subjects:  subjects,
		Storage:   nats.FileStorage,
		Retention: nats.LimitsPolicy,
		MaxAge:    maxAge,
	})
	if err != nil && !isStreamExists(err) {
		return err
	}
	return nil
}

// CloseNATS drains and closes the connection.
func CloseNATS() {
	if NC != nil {
		_ = NC.Drain()
		NC.Close()
	}
}

func isStreamExists(err error) bool {
	return err == nats.ErrStreamNameAlreadyInUse
}
