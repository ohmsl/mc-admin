package audit

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
)

type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailed  Outcome = "failed"
	OutcomeDenied  Outcome = "denied"
)

type Entry struct {
	ActorID       string
	ActorEmail    string
	ActorRole     string
	Action        string
	Payload       map[string]any
	Outcome       Outcome
	Error         string
	RequestID     string
	IP            string
	UserAgent     string
	OccurredAtUTC time.Time
}

type Logger struct {
	app            core.App
	collectionName string
}

func NewLogger(app core.App, collectionName string) *Logger {
	return &Logger{
		app:            app,
		collectionName: collectionName,
	}
}

func (l *Logger) Log(entry Entry) (string, error) {
	collection, err := l.app.Dao().FindCollectionByNameOrId(l.collectionName)
	if err != nil {
		return "", fmt.Errorf("find audit collection: %w", err)
	}

	record := models.NewRecord(collection)
	record.Set("actor_id", entry.ActorID)
	record.Set("actor_email", entry.ActorEmail)
	record.Set("actor_role", strings.ToLower(strings.TrimSpace(entry.ActorRole)))
	record.Set("action", entry.Action)
	record.Set("payload_redacted", entry.Payload)
	record.Set("outcome", string(entry.Outcome))
	record.Set("error", entry.Error)
	record.Set("request_id", entry.RequestID)
	record.Set("ip", entry.IP)
	record.Set("user_agent", entry.UserAgent)

	if err := l.app.Dao().SaveRecord(record); err != nil {
		return "", fmt.Errorf("save audit record: %w", err)
	}

	return record.Id, nil
}
