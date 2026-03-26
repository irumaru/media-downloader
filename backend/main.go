package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"media-downloader/api"
	"media-downloader/config"
	"media-downloader/db"
	"media-downloader/handler"
	"media-downloader/service"

	_ "modernc.org/sqlite"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Ensure the DB directory exists.
	if err := os.MkdirAll(filepath.Dir(cfg.Database.Path), 0o755); err != nil {
		log.Fatalf("create db dir: %v", err)
	}

	sqlDB, err := sql.Open("sqlite", cfg.Database.Path)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer sqlDB.Close()

	// Enable WAL mode for better concurrent read performance.
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Fatalf("set wal mode: %v", err)
	}

	// Run schema migrations (idempotent due to IF NOT EXISTS).
	schemaBytes, err := os.ReadFile("db/schema.sql")
	if err != nil {
		log.Fatalf("read schema: %v", err)
	}
	if _, err := sqlDB.Exec(string(schemaBytes)); err != nil {
		log.Fatalf("run schema: %v", err)
	}

	queries := db.New(sqlDB)
	svc := service.New(queries, cfg)
	h := handler.New(svc)

	srv, err := api.NewServer(h)
	if err != nil {
		log.Fatalf("create api server: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", srv)
	mux.Handle("/", spaHandler(cfg.Server.StaticDir))

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

// spaHandler serves static files and falls back to index.html for SPA routing.
func spaHandler(staticDir string) http.Handler {
	fs := http.FileServer(http.Dir(staticDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(staticDir, filepath.Clean(r.URL.Path))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
}
