package mc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gorcon/rcon"
	"github.com/ohmsl/mc-admin/apps/pb/internal/config"
)

type rconConn interface {
	Execute(command string) (string, error)
	Close() error
}

type rconDialer func(address string, password string) (rconConn, error)

type RCONExecutor struct {
	cfg    config.RCONConfig
	dialer rconDialer
}

func NewRCONExecutor(cfg config.RCONConfig) *RCONExecutor {
	return &RCONExecutor{
		cfg: cfg,
		dialer: func(address string, password string) (rconConn, error) {
			timeout := cfg.Timeout
			if timeout <= 0 {
				timeout = 5 * time.Second
			}
			return rcon.Dial(
				address,
				password,
				rcon.SetDialTimeout(timeout),
				rcon.SetDeadline(timeout),
			)
		},
	}
}

func (r *RCONExecutor) Run(ctx context.Context, command string) (string, error) {
	results, err := r.RunBatch(ctx, []string{command})
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", nil
	}
	return results[0], nil
}

func (r *RCONExecutor) RunBatch(ctx context.Context, commands []string) ([]string, error) {
	if len(commands) == 0 {
		return nil, nil
	}

	if r.cfg.Password == "" {
		return nil, errors.New("MC_RCON_PASSWORD is not configured")
	}

	attempts := r.cfg.RetryCount + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		outputs, err := r.runOnce(ctx, commands)
		if err == nil {
			return outputs, nil
		}

		lastErr = err
		if attempt == attempts {
			break
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}

	return nil, fmt.Errorf("rcon execute failed after %d attempt(s): %w", attempts, lastErr)
}

func (r *RCONExecutor) runOnce(ctx context.Context, commands []string) ([]string, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	conn, err := r.dialer(r.cfg.Address(), r.cfg.Password)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	outputs := make([]string, 0, len(commands))
	for _, command := range commands {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		output, execErr := conn.Execute(command)
		if execErr != nil {
			return nil, fmt.Errorf("execute command %q: %w", command, execErr)
		}
		outputs = append(outputs, output)
	}

	return outputs, nil
}
