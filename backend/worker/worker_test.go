package worker

import "testing"

func TestParseProgress(t *testing.T) {
	tests := []struct {
		line string
		want int
		ok   bool
	}{
		{"[download]  72.3% of   4.50MiB at    1.23MiB/s ETA 00:02", 72, true},
		{"[download] 100% of    4.50MiB at    1.23MiB/s", 100, true},
		{"[download]   0.1% of    4.50MiB at   Unknown speed ETA Unknown", 0, true},
		{"[download]   5.0% of    4.50MiB at    1.23MiB/s ETA 00:04", 5, true},
		{"[ExtractAudio] Destination: title.opus", 0, false},
		{"[download] Destination: title.webm", 0, false},
		{"", 0, false},
		{"some random line", 0, false},
	}

	for _, tt := range tests {
		got, ok := ParseProgress(tt.line)
		if ok != tt.ok {
			t.Errorf("ParseProgress(%q): ok=%v, want %v", tt.line, ok, tt.ok)
			continue
		}
		if ok && got != tt.want {
			t.Errorf("ParseProgress(%q): progress=%d, want %d", tt.line, got, tt.want)
		}
	}
}
