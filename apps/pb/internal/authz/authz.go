package authz

import (
	"strings"

	"github.com/ohmsl/mc-admin/apps/pb/internal/mc"
	"github.com/pocketbase/pocketbase/models"
)

type Role string

const (
	RoleViewer   Role = "viewer"
	RoleOperator Role = "operator"
	RoleOwner    Role = "owner"
)

func NormalizeRole(raw string) Role {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(RoleOwner):
		return RoleOwner
	case string(RoleOperator):
		return RoleOperator
	default:
		return RoleViewer
	}
}

func RoleFromRecord(record *models.Record, fieldName string) Role {
	if record == nil {
		return RoleViewer
	}

	return NormalizeRole(record.GetString(fieldName))
}

func CanViewStatus(role Role) bool {
	switch role {
	case RoleViewer, RoleOperator, RoleOwner:
		return true
	default:
		return false
	}
}

func CanExecuteAction(role Role, action mc.Action, allowRaw bool) bool {
	switch role {
	case RoleOwner:
		if action == mc.ActionRawCommand {
			return allowRaw
		}
		return true
	case RoleOperator:
		switch action {
		case mc.ActionWhitelistAdd,
			mc.ActionWhitelistRemove,
			mc.ActionKick,
			mc.ActionSay,
			mc.ActionSaveWorld:
			return true
		default:
			return false
		}
	case RoleViewer:
		return false
	default:
		return false
	}
}
