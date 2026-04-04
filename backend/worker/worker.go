package worker

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// ProgressUpdate holds the current state parsed from yt-dlp output.
type ProgressUpdate struct {
	Title    string
	Status   string
	Progress int
	Filename string
}

// ProgressCallback is called whenever yt-dlp reports a state change.
type ProgressCallback func(ProgressUpdate)

var (
	progressRe    = regexp.MustCompile(`\[download\]\s+(\d+(?:\.\d+)?)%`)
	destinationRe = regexp.MustCompile(`\[download\] Destination: (.+)`)
	extractRe     = regexp.MustCompile(`\[ExtractAudio\] Destination: (.+)`)
)

// Run executes yt-dlp and streams progress updates via callback.
// It blocks until the download is complete or the context is cancelled.
func Run(ctx context.Context, ytdlpPath, audioFormat, outputDir, url string, callback ProgressCallback) error {
	args := []string{
		"-x",
		"--audio-quality", "0",
		"--newline",
		"-o", outputDir + "/%(title)s.%(ext)s",
		"--no-playlist",
	}
	if audioFormat != "" {
		args = append(args, "--audio-format", audioFormat)
	}
	args = append(args, url)

	cmd := exec.CommandContext(ctx, ytdlpPath, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	slog.DebugContext(ctx, "Starting yt-dlp", "cmd", cmd.String())
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start yt-dlp: %w", err)
	}
	slog.InfoContext(ctx, "yt-dlp started", "url", url, "outputDir", outputDir, "audioFormat", audioFormat)

	// Drain stderr and log each line as a warning.
	go func() {
		s := bufio.NewScanner(stderr)
		for s.Scan() {
			slog.WarnContext(ctx, "yt-dlp stderr", "line", s.Text())
		}
	}()

	var title, filename string
	status := "downloading"

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case progressRe.MatchString(line):
			m := progressRe.FindStringSubmatch(line)
			pct, _ := strconv.ParseFloat(m[1], 64)
			callback(ProgressUpdate{
				Title:    title,
				Status:   status,
				Progress: int(pct),
				Filename: filename,
			})

		case destinationRe.MatchString(line):
			m := destinationRe.FindStringSubmatch(line)
			dest := strings.TrimSpace(m[1])
			base := dest
			if idx := strings.LastIndex(base, "/"); idx >= 0 {
				base = base[idx+1:]
			}
			if idx := strings.LastIndex(base, "."); idx >= 0 {
				title = base[:idx]
			}
			slog.InfoContext(ctx, "download destination detected", "title", title, "dest", dest)

		case extractRe.MatchString(line):
			m := extractRe.FindStringSubmatch(line)
			dest := strings.TrimSpace(m[1])
			if idx := strings.LastIndex(dest, "/"); idx >= 0 {
				filename = dest[idx+1:]
			} else {
				filename = dest
			}
			status = "converting"
			slog.InfoContext(ctx, "converting audio", "title", title, "filename", filename)
			callback(ProgressUpdate{
				Title:    title,
				Status:   status,
				Progress: 99,
				Filename: filename,
			})
		}
	}

	if err := cmd.Wait(); err != nil {
		slog.ErrorContext(ctx, "yt-dlp failed", "url", url, "err", err)
		return fmt.Errorf("yt-dlp exited with error: %w", err)
	}

	slog.InfoContext(ctx, "download completed", "url", url, "title", title, "filename", filename)
	callback(ProgressUpdate{
		Title:    title,
		Status:   "completed",
		Progress: 100,
		Filename: filename,
	})
	return nil
}

// ParseProgress parses a single yt-dlp output line and returns the progress percentage.
// Returns (progress, true) if the line contains a progress update, otherwise (0, false).
// Exported for unit testing.
func ParseProgress(line string) (int, bool) {
	m := progressRe.FindStringSubmatch(line)
	if len(m) < 2 {
		return 0, false
	}
	pct, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, false
	}
	return int(pct), true
}
