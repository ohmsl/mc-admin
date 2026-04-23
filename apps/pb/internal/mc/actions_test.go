package mc

import (
	"encoding/json"
	"testing"
)

func TestBuildActionPlan_Valid(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name      string
		action    Action
		payload   string
		allowRaw  bool
		wantCount int
	}{
		{name: "whitelist add", action: ActionWhitelistAdd, payload: `{"player":"Steve"}`, wantCount: 1},
		{name: "whitelist remove", action: ActionWhitelistRemove, payload: `{"player":"Alex"}`, wantCount: 1},
		{name: "kick", action: ActionKick, payload: `{"player":"Steve","reason":"AFK"}`, wantCount: 1},
		{name: "say", action: ActionSay, payload: `{"message":"Server restart soon"}`, wantCount: 1},
		{name: "save", action: ActionSaveWorld, payload: `{}`, wantCount: 1},
		{name: "restart", action: ActionRestartServer, payload: `{"message":"Restarting now"}`, wantCount: 3},
		{name: "raw", action: ActionRawCommand, payload: `{"command":"time set day"}`, allowRaw: true, wantCount: 1},
	}

	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			plan, err := BuildActionPlan(scenario.action, json.RawMessage(scenario.payload), scenario.allowRaw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(plan.Commands) != scenario.wantCount {
				t.Fatalf("expected %d commands got %d", scenario.wantCount, len(plan.Commands))
			}
		})
	}
}

func TestBuildActionPlan_Invalid(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name    string
		action  Action
		payload string
	}{
		{name: "invalid player", action: ActionWhitelistAdd, payload: `{"player":"!!!"}`},
		{name: "say empty", action: ActionSay, payload: `{"message":""}`},
		{name: "kick missing player", action: ActionKick, payload: `{"reason":"x"}`},
		{name: "save with payload", action: ActionSaveWorld, payload: `{"x":1}`},
		{name: "raw disabled", action: ActionRawCommand, payload: `{"command":"op me"}`},
		{name: "raw multiline", action: ActionRawCommand, payload: "{\"command\":\"say hi\\nbye\"}"},
	}

	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			_, err := BuildActionPlan(scenario.action, json.RawMessage(scenario.payload), false)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestParseAction(t *testing.T) {
	t.Parallel()

	if _, err := ParseAction("unknown"); err == nil {
		t.Fatal("expected unknown action error")
	}

	action, err := ParseAction("WHITELIST_ADD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if action != ActionWhitelistAdd {
		t.Fatalf("expected %s got %s", ActionWhitelistAdd, action)
	}
}
