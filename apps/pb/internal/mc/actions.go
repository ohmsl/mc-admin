package mc

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type Action string

const (
	ActionWhitelistAdd    Action = "whitelist_add"
	ActionWhitelistRemove Action = "whitelist_remove"
	ActionKick            Action = "kick"
	ActionSay             Action = "say"
	ActionSaveWorld       Action = "save_world"
	ActionRestartServer   Action = "restart_server"
	ActionRawCommand      Action = "raw_command"
)

var (
	errInvalidAction  = errors.New("invalid action")
	errInvalidPayload = errors.New("invalid payload")

	playerNamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,16}$`)
)

type ActionPlan struct {
	Action       Action
	Commands     []string
	AuditPayload map[string]any
}

type payloadWhitelist struct {
	Player string `json:"player"`
}

type payloadKick struct {
	Player string `json:"player"`
	Reason string `json:"reason"`
}

type payloadSay struct {
	Message string `json:"message"`
}

type payloadRestart struct {
	Message string `json:"message"`
}

type payloadRaw struct {
	Command string `json:"command"`
}

func ParseAction(raw string) (Action, error) {
	action := Action(strings.TrimSpace(strings.ToLower(raw)))
	if !action.IsKnown() {
		return "", fmt.Errorf("%w: %s", errInvalidAction, raw)
	}
	return action, nil
}

func (a Action) IsKnown() bool {
	switch a {
	case ActionWhitelistAdd,
		ActionWhitelistRemove,
		ActionKick,
		ActionSay,
		ActionSaveWorld,
		ActionRestartServer,
		ActionRawCommand:
		return true
	default:
		return false
	}
}

func BuildActionPlan(action Action, rawPayload json.RawMessage, allowRaw bool) (ActionPlan, error) {
	switch action {
	case ActionWhitelistAdd:
		payload := payloadWhitelist{}
		if err := decodePayload(rawPayload, &payload); err != nil {
			return ActionPlan{}, err
		}
		player, err := validatePlayer(payload.Player)
		if err != nil {
			return ActionPlan{}, err
		}
		return ActionPlan{
			Action:   action,
			Commands: []string{fmt.Sprintf("whitelist add %s", player)},
			AuditPayload: map[string]any{
				"player": player,
			},
		}, nil
	case ActionWhitelistRemove:
		payload := payloadWhitelist{}
		if err := decodePayload(rawPayload, &payload); err != nil {
			return ActionPlan{}, err
		}
		player, err := validatePlayer(payload.Player)
		if err != nil {
			return ActionPlan{}, err
		}
		return ActionPlan{
			Action:   action,
			Commands: []string{fmt.Sprintf("whitelist remove %s", player)},
			AuditPayload: map[string]any{
				"player": player,
			},
		}, nil
	case ActionKick:
		payload := payloadKick{}
		if err := decodePayload(rawPayload, &payload); err != nil {
			return ActionPlan{}, err
		}
		player, err := validatePlayer(payload.Player)
		if err != nil {
			return ActionPlan{}, err
		}
		reason, err := validateReason(payload.Reason)
		if err != nil {
			return ActionPlan{}, err
		}

		command := fmt.Sprintf("kick %s", player)
		if reason != "" {
			command = fmt.Sprintf("%s %s", command, reason)
		}

		return ActionPlan{
			Action:   action,
			Commands: []string{command},
			AuditPayload: map[string]any{
				"player": player,
				"reason": reason,
			},
		}, nil
	case ActionSay:
		payload := payloadSay{}
		if err := decodePayload(rawPayload, &payload); err != nil {
			return ActionPlan{}, err
		}
		message, err := validateMessage(payload.Message)
		if err != nil {
			return ActionPlan{}, err
		}

		return ActionPlan{
			Action:   action,
			Commands: []string{fmt.Sprintf("say %s", message)},
			AuditPayload: map[string]any{
				"message": message,
			},
		}, nil
	case ActionSaveWorld:
		if err := ensureEmptyPayload(rawPayload); err != nil {
			return ActionPlan{}, err
		}
		return ActionPlan{
			Action:       action,
			Commands:     []string{"save-all"},
			AuditPayload: map[string]any{},
		}, nil
	case ActionRestartServer:
		payload := payloadRestart{}
		if err := decodePayload(rawPayload, &payload); err != nil {
			return ActionPlan{}, err
		}
		message, err := validateReason(payload.Message)
		if err != nil {
			return ActionPlan{}, err
		}

		commands := []string{"save-all", "stop"}
		if message != "" {
			commands = []string{fmt.Sprintf("say %s", message), "save-all", "stop"}
		}

		return ActionPlan{
			Action:   action,
			Commands: commands,
			AuditPayload: map[string]any{
				"message": message,
			},
		}, nil
	case ActionRawCommand:
		if !allowRaw {
			return ActionPlan{}, fmt.Errorf("%w: raw command is disabled", errInvalidAction)
		}

		payload := payloadRaw{}
		if err := decodePayload(rawPayload, &payload); err != nil {
			return ActionPlan{}, err
		}

		command := strings.TrimSpace(payload.Command)
		if command == "" {
			return ActionPlan{}, fmt.Errorf("%w: command is required", errInvalidPayload)
		}
		if len(command) > 256 {
			return ActionPlan{}, fmt.Errorf("%w: command must be 256 chars or fewer", errInvalidPayload)
		}
		if strings.ContainsAny(command, "\r\n") {
			return ActionPlan{}, fmt.Errorf("%w: command cannot contain line breaks", errInvalidPayload)
		}

		return ActionPlan{
			Action:   action,
			Commands: []string{command},
			AuditPayload: map[string]any{
				"command": command,
			},
		}, nil
	default:
		return ActionPlan{}, fmt.Errorf("%w: %s", errInvalidAction, action)
	}
}

func AllowedActions(allowRaw bool) []Action {
	actions := []Action{
		ActionWhitelistAdd,
		ActionWhitelistRemove,
		ActionKick,
		ActionSay,
		ActionSaveWorld,
		ActionRestartServer,
	}
	if allowRaw {
		actions = append(actions, ActionRawCommand)
	}
	sort.Slice(actions, func(i, j int) bool {
		return actions[i] < actions[j]
	})
	return actions
}

func decodePayload(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		raw = []byte("{}")
	}

	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("%w: %s", errInvalidPayload, err)
	}

	return nil
}

func ensureEmptyPayload(raw json.RawMessage) error {
	if len(raw) == 0 {
		return nil
	}

	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "{}" || trimmed == "null" {
		return nil
	}

	return fmt.Errorf("%w: action does not accept payload", errInvalidPayload)
}

func validatePlayer(raw string) (string, error) {
	player := strings.TrimSpace(raw)
	if !playerNamePattern.MatchString(player) {
		return "", fmt.Errorf("%w: invalid player name", errInvalidPayload)
	}
	return player, nil
}

func validateReason(raw string) (string, error) {
	reason := strings.TrimSpace(raw)
	if reason == "" {
		return "", nil
	}
	if len(reason) > 120 {
		return "", fmt.Errorf("%w: reason must be 120 chars or fewer", errInvalidPayload)
	}
	if strings.ContainsAny(reason, "\r\n") {
		return "", fmt.Errorf("%w: reason cannot contain line breaks", errInvalidPayload)
	}
	return reason, nil
}

func validateMessage(raw string) (string, error) {
	message := strings.TrimSpace(raw)
	if message == "" {
		return "", fmt.Errorf("%w: message is required", errInvalidPayload)
	}
	if len(message) > 200 {
		return "", fmt.Errorf("%w: message must be 200 chars or fewer", errInvalidPayload)
	}
	if strings.ContainsAny(message, "\r\n") {
		return "", fmt.Errorf("%w: message cannot contain line breaks", errInvalidPayload)
	}
	return message, nil
}
