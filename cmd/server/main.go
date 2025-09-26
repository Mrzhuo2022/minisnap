package main

import (
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	"minisnap/internal/config"
	"minisnap/internal/content"
	"minisnap/internal/server"
)

func main() {
	_ = godotenv.Load()

	bindFlag := flag.String("bind", "", "override bind address, e.g. :9090")
	contentFlag := flag.String("content-dir", "", "override content directory path")
	passwordFlag := flag.String("admin-password", "", "override admin password (for development only)")
	flag.Parse()

	if *passwordFlag != "" {
		_ = os.Setenv("ADMIN_PASSWORD", *passwordFlag)
	}
	if *bindFlag != "" {
		_ = os.Setenv("BIND_ADDR", *bindFlag)
	}
	if *contentFlag != "" {
		_ = os.Setenv("CONTENT_DIR", *contentFlag)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	store, err := content.NewStore(cfg.ContentDir)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("get wd: %v", err)
	}

	tplDir := filepath.Join(cwd, "templates")
	s, err := server.New(cfg, store, tplDir)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}

	slog.Info("starting server", "addr", cfg.BindAddr)
	if err := http.ListenAndServe(cfg.BindAddr, s); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
