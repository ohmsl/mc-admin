package config

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/models"
	"github.com/pocketbase/pocketbase/models/schema"
	"github.com/pocketbase/pocketbase/tools/types"
)

const usersCollectionName = "users"

func EnsureCollections(app core.App, cfg Config) error {
	if err := ensureUserRoleField(app, cfg.Permissions.RoleField); err != nil {
		return err
	}

	if err := ensureAuditCollection(app, cfg.Collections.AuditLogs); err != nil {
		return err
	}

	return nil
}

func ensureUserRoleField(app core.App, roleField string) error {
	users, err := app.Dao().FindCollectionByNameOrId(usersCollectionName)
	if err != nil {
		// Some installations start without a default "users" auth collection.
		// In that case, role management can be applied later when the collection exists.
		return nil
	}

	if users.Schema.GetFieldByName(roleField) != nil {
		return nil
	}

	users.Schema.AddField(&schema.SchemaField{
		Name:     roleField,
		Type:     schema.FieldTypeSelect,
		Required: false,
		Options: &schema.SelectOptions{
			MaxSelect: 1,
			Values:    []string{"viewer", "operator", "owner"},
		},
	})

	if err := app.Dao().SaveCollection(users); err != nil {
		return fmt.Errorf("save users collection role field: %w", err)
	}

	return nil
}

func ensureAuditCollection(app core.App, collectionName string) error {
	collection, err := app.Dao().FindCollectionByNameOrId(collectionName)
	if err == nil && collection != nil {
		return nil
	}

	authRule := "@request.auth.id != ''"

	collection = &models.Collection{}
	collection.Name = collectionName
	collection.Type = models.CollectionTypeBase
	collection.ListRule = &authRule
	collection.ViewRule = &authRule
	collection.Schema = schema.NewSchema(
		&schema.SchemaField{
			Name:     "actor_id",
			Type:     schema.FieldTypeText,
			Required: true,
		},
		&schema.SchemaField{
			Name:     "actor_email",
			Type:     schema.FieldTypeText,
			Required: false,
		},
		&schema.SchemaField{
			Name:     "actor_role",
			Type:     schema.FieldTypeSelect,
			Required: true,
			Options: &schema.SelectOptions{
				MaxSelect: 1,
				Values:    []string{"viewer", "operator", "owner"},
			},
		},
		&schema.SchemaField{
			Name:     "action",
			Type:     schema.FieldTypeText,
			Required: true,
		},
		&schema.SchemaField{
			Name:     "payload_redacted",
			Type:     schema.FieldTypeJson,
			Required: false,
			Options: &schema.JsonOptions{MaxSize: 102400},
		},
		&schema.SchemaField{
			Name:     "outcome",
			Type:     schema.FieldTypeSelect,
			Required: true,
			Options: &schema.SelectOptions{
				MaxSelect: 1,
				Values:    []string{"success", "failed", "denied"},
			},
		},
		&schema.SchemaField{
			Name:     "error",
			Type:     schema.FieldTypeText,
			Required: false,
		},
		&schema.SchemaField{
			Name:     "request_id",
			Type:     schema.FieldTypeText,
			Required: false,
		},
		&schema.SchemaField{
			Name:     "ip",
			Type:     schema.FieldTypeText,
			Required: false,
		},
		&schema.SchemaField{
			Name:     "user_agent",
			Type:     schema.FieldTypeText,
			Required: false,
		},
	)
	collection.Indexes = types.JsonArray[string]{
		fmt.Sprintf("CREATE INDEX idx_%s_created ON %s (created)", collectionName, collectionName),
		fmt.Sprintf("CREATE INDEX idx_%s_actor_id ON %s (actor_id)", collectionName, collectionName),
	}

	if err := app.Dao().SaveCollection(collection); err != nil {
		return fmt.Errorf("create %s collection: %w", collectionName, err)
	}

	return nil
}
