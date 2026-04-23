package mc

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var listResponsePattern = regexp.MustCompile(`There are\s+(\d+)\s+of a max of\s+(\d+)\s+players online:?\s*(.*)$`)

type Executor interface {
	Run(ctx context.Context, command string) (string, error)
	RunBatch(ctx context.Context, commands []string) ([]string, error)
}

type Service struct {
	executor Executor
}

func NewService(executor Executor) *Service {
	return &Service{executor: executor}
}

type Status struct {
	ServerReachable bool      `json:"serverReachable"`
	OnlinePlayers   []string  `json:"onlinePlayers"`
	OnlineCount     int       `json:"onlineCount"`
	MaxPlayers      *int      `json:"maxPlayers"`
	CheckedAt       time.Time `json:"checkedAt"`
	Degraded        bool      `json:"degraded"`
	Message         string    `json:"message,omitempty"`
}

type ExecuteResult struct {
	Action     Action    `json:"action"`
	Message    string    `json:"message"`
	Output     []string  `json:"output,omitempty"`
	ExecutedAt time.Time `json:"executedAt"`
}

func (s *Service) Status(ctx context.Context) (Status, error) {
	checkedAt := time.Now().UTC()
	response, err := s.executor.Run(ctx, "list")
	if err != nil {
		return Status{
			ServerReachable: false,
			OnlinePlayers:   []string{},
			OnlineCount:     0,
			CheckedAt:       checkedAt,
			Degraded:        false,
			Message:         "minecraft server unreachable",
		}, err
	}

	onlinePlayers, onlineCount, maxPlayers, degraded := parseListResponse(response)

	status := Status{
		ServerReachable: true,
		OnlinePlayers:   onlinePlayers,
		OnlineCount:     onlineCount,
		MaxPlayers:      maxPlayers,
		CheckedAt:       checkedAt,
		Degraded:        degraded,
	}

	if degraded {
		status.Message = "status response parsed partially"
	}

	return status, nil
}

func (s *Service) ExecuteAction(ctx context.Context, plan ActionPlan) (ExecuteResult, error) {
	outputs, err := s.executor.RunBatch(ctx, plan.Commands)
	if err != nil {
		return ExecuteResult{}, err
	}

	message := fmt.Sprintf("%s executed", plan.Action)
	if len(outputs) > 0 {
		lastOutput := strings.TrimSpace(outputs[len(outputs)-1])
		if lastOutput != "" {
			message = lastOutput
		}
	}

	return ExecuteResult{
		Action:     plan.Action,
		Message:    message,
		Output:     outputs,
		ExecutedAt: time.Now().UTC(),
	}, nil
}

func parseListResponse(response string) ([]string, int, *int, bool) {
	trimmed := strings.TrimSpace(response)
	matches := listResponsePattern.FindStringSubmatch(trimmed)
	if len(matches) != 4 {
		return []string{}, 0, nil, true
	}

	onlineCount, onlineErr := strconv.Atoi(matches[1])
	maxPlayers, maxErr := strconv.Atoi(matches[2])
	if onlineErr != nil || maxErr != nil {
		return []string{}, 0, nil, true
	}

	playersSection := strings.TrimSpace(matches[3])
	players := []string{}
	if playersSection != "" {
		parts := strings.Split(playersSection, ",")
		players = make([]string, 0, len(parts))
		for _, p := range parts {
			name := strings.TrimSpace(p)
			if name == "" {
				continue
			}
			players = append(players, name)
		}
	}

	if len(players) != onlineCount {
		// keep the parsed values but mark degraded so UI can surface uncertainty.
		return players, onlineCount, &maxPlayers, true
	}

	return players, onlineCount, &maxPlayers, false
}
