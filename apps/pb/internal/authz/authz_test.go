package authz

import (
	"testing"

	"github.com/ohmsl/mc-admin/apps/pb/internal/mc"
)

func TestCanExecuteActionMatrix(t *testing.T) {
	t.Parallel()

	actions := []mc.Action{
		mc.ActionWhitelistAdd,
		mc.ActionWhitelistRemove,
		mc.ActionKick,
		mc.ActionSay,
		mc.ActionSaveWorld,
		mc.ActionRestartServer,
		mc.ActionRawCommand,
	}

	scenarios := []struct {
		name     string
		role     Role
		allowRaw bool
		allowed  map[mc.Action]bool
	}{
		{
			name:     "viewer",
			role:     RoleViewer,
			allowRaw: false,
			allowed:  map[mc.Action]bool{},
		},
		{
			name:     "operator",
			role:     RoleOperator,
			allowRaw: false,
			allowed: map[mc.Action]bool{
				mc.ActionWhitelistAdd:    true,
				mc.ActionWhitelistRemove: true,
				mc.ActionKick:            true,
				mc.ActionSay:             true,
				mc.ActionSaveWorld:       true,
			},
		},
		{
			name:     "owner raw disabled",
			role:     RoleOwner,
			allowRaw: false,
			allowed: map[mc.Action]bool{
				mc.ActionWhitelistAdd:    true,
				mc.ActionWhitelistRemove: true,
				mc.ActionKick:            true,
				mc.ActionSay:             true,
				mc.ActionSaveWorld:       true,
				mc.ActionRestartServer:   true,
			},
		},
		{
			name:     "owner raw enabled",
			role:     RoleOwner,
			allowRaw: true,
			allowed: map[mc.Action]bool{
				mc.ActionWhitelistAdd:    true,
				mc.ActionWhitelistRemove: true,
				mc.ActionKick:            true,
				mc.ActionSay:             true,
				mc.ActionSaveWorld:       true,
				mc.ActionRestartServer:   true,
				mc.ActionRawCommand:      true,
			},
		},
	}

	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			for _, action := range actions {
				expected := scenario.allowed[action]
				actual := CanExecuteAction(scenario.role, action, scenario.allowRaw)
				if actual != expected {
					t.Fatalf("action %s expected allowed=%v got %v", action, expected, actual)
				}
			}
		})
	}
}

func TestNormalizeRole(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		in   string
		want Role
	}{
		{"owner", RoleOwner},
		{"OWNER", RoleOwner},
		{"operator", RoleOperator},
		{"unknown", RoleViewer},
		{"", RoleViewer},
	}

	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario.in, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeRole(scenario.in); got != scenario.want {
				t.Fatalf("expected %s got %s", scenario.want, got)
			}
		})
	}
}
