package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"media-downloader/config"
	"media-downloader/db"
	"media-downloader/worker"

	"github.com/google/uuid"
)

type Service struct {
	queries  *db.Queries
	channels map[string]config.ChannelConfig
	ytdlp    config.YtdlpConfig
}

func New(queries *db.Queries, cfg *config.Config) *Service {
	channelMap := make(map[string]config.ChannelConfig, len(cfg.Channels))
	for _, ch := range cfg.Channels {
		channelMap[ch.Secret] = ch
	}
	return &Service{
		queries:  queries,
		channels: channelMap,
		ytdlp:    cfg.Ytdlp,
	}
}

// GetChannel returns the channel config for the given secret, or false if not found.
func (s *Service) GetChannel(secret string) (config.ChannelConfig, bool) {
	ch, ok := s.channels[secret]
	return ch, ok
}

// GetDownload returns a single download by ID.
func (s *Service) GetDownload(ctx context.Context, id string) (db.Download, error) {
	return s.queries.GetDownloadByID(ctx, id)
}

// ListDownloads returns all downloads for the given channel secret.
func (s *Service) ListDownloads(ctx context.Context, secret string) ([]db.Download, error) {
	return s.queries.GetDownloadsByChannel(ctx, secret)
}

// CreateDownload inserts a new download record and starts the download in a goroutine.
func (s *Service) CreateDownload(ctx context.Context, secret, url string) (db.Download, error) {
	ch, ok := s.channels[secret]
	if !ok {
		return db.Download{}, fmt.Errorf("channel not found: %s", secret)
	}

	download, err := s.queries.CreateDownload(ctx, db.CreateDownloadParams{
		ID:      uuid.New().String(),
		Channel: secret,
		Url:     url,
	})
	if err != nil {
		return db.Download{}, fmt.Errorf("create download record: %w", err)
	}

	slog.InfoContext(ctx, "Register download queue", "channel", secret, "url", url, "download_id", download.ID)

	go s.runDownload(ch, download)

	return download, nil
}

func (s *Service) runDownload(ch config.ChannelConfig, d db.Download) {
	ctx := context.Background()

	err := worker.Run(ctx, s.ytdlp.Path, s.ytdlp.AudioFormat, ch.OutputDir, d.Url,
		func(u worker.ProgressUpdate) {
			err := s.queries.UpdateDownloadStatus(ctx, db.UpdateDownloadStatusParams{
				ID:          d.ID,
				Status:      u.Status,
				Progress:    int64(u.Progress),
				Title:       nullString(u.Title),
				Filename:    nullString(u.Filename),
				Error:       sql.NullString{},
				CompletedAt: completedAt(u.Status),
			})
			if err != nil {
				slog.ErrorContext(ctx, "failed to update download status", "download_id", d.ID, "err", err)
			}
		})

	if err != nil {
		err := s.queries.UpdateDownloadStatus(ctx, db.UpdateDownloadStatusParams{
			ID:          d.ID,
			Status:      "error",
			Progress:    0,
			Error:       sql.NullString{String: err.Error(), Valid: true},
			CompletedAt: completedAt("error"),
		})
		if err != nil {
			slog.ErrorContext(ctx, "failed to update download status", "download_id", d.ID, "err", err)
		}
	}
}

func completedAt(status string) sql.NullTime {
	if status == "completed" || status == "error" {
		return sql.NullTime{Time: time.Now(), Valid: true}
	}
	return sql.NullTime{}
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
