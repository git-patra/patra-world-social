package leaderboard

import "testing"

func TestRankingUsesExactAccuracyThenTimeAndStableResultID(t *testing.T) {
	base := Entry{Score: 100, Hits: 1, Shots: 3, AchievedAt: 100, ResultID: "b"}
	cases := []struct {
		name   string
		entry  Entry
		better bool
	}{
		{"score dominates accuracy", Entry{Score: 101, Shots: 2, AchievedAt: 200, ResultID: "z"}, true},
		{"rational accuracy", Entry{Score: 100, Hits: 80, Shots: 239, AchievedAt: 200, ResultID: "z"}, true},
		{"equal rational earlier", Entry{Score: 100, Hits: 2, Shots: 6, AchievedAt: 99, ResultID: "z"}, true},
		{"equal rational later", Entry{Score: 100, Hits: 2, Shots: 6, AchievedAt: 101, ResultID: "a"}, false},
		{"stable id", Entry{Score: 100, Hits: 2, Shots: 6, AchievedAt: 100, ResultID: "a"}, true},
		{"same result", base, false},
		{"zero shots", Entry{Score: 100, AchievedAt: 99, ResultID: "a"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := better(tc.entry, base); got != tc.better {
				t.Fatalf("ranking = %v want %v", got, tc.better)
			}
		})
	}
	if better(Entry{Score: 0, Shots: 0, AchievedAt: 2}, Entry{Score: 0, Shots: 10, AchievedAt: 1}) {
		t.Fatal("zero shot accuracy must equal zero hits")
	}
}
