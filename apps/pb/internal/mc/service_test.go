package mc

import "testing"

func TestParseListResponse(t *testing.T) {
	t.Parallel()

	scenarios := []struct {
		name            string
		response        string
		wantPlayers     []string
		wantOnlineCount int
		wantMax         int
		wantDegraded    bool
	}{
		{
			name:            "no players",
			response:        "There are 0 of a max of 20 players online:",
			wantPlayers:     []string{},
			wantOnlineCount: 0,
			wantMax:         20,
			wantDegraded:    false,
		},
		{
			name:            "players listed",
			response:        "There are 2 of a max of 20 players online: Steve, Alex",
			wantPlayers:     []string{"Steve", "Alex"},
			wantOnlineCount: 2,
			wantMax:         20,
			wantDegraded:    false,
		},
		{
			name:            "unparseable",
			response:        "weird response",
			wantPlayers:     []string{},
			wantOnlineCount: 0,
			wantMax:         0,
			wantDegraded:    true,
		},
		{
			name:            "count mismatch",
			response:        "There are 2 of a max of 20 players online: Steve",
			wantPlayers:     []string{"Steve"},
			wantOnlineCount: 2,
			wantMax:         20,
			wantDegraded:    true,
		},
	}

	for _, scenario := range scenarios {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			players, onlineCount, maxPlayers, degraded := parseListResponse(scenario.response)
			if onlineCount != scenario.wantOnlineCount {
				t.Fatalf("expected online count %d got %d", scenario.wantOnlineCount, onlineCount)
			}
			if degraded != scenario.wantDegraded {
				t.Fatalf("expected degraded=%v got %v", scenario.wantDegraded, degraded)
			}
			if len(players) != len(scenario.wantPlayers) {
				t.Fatalf("expected %d players got %d", len(scenario.wantPlayers), len(players))
			}
			for i := range players {
				if players[i] != scenario.wantPlayers[i] {
					t.Fatalf("player[%d] expected %s got %s", i, scenario.wantPlayers[i], players[i])
				}
			}

			if scenario.wantMax == 0 {
				if maxPlayers != nil {
					t.Fatalf("expected nil max players got %d", *maxPlayers)
				}
				return
			}

			if maxPlayers == nil {
				t.Fatal("expected max players value")
			}
			if *maxPlayers != scenario.wantMax {
				t.Fatalf("expected max players %d got %d", scenario.wantMax, *maxPlayers)
			}
		})
	}
}
