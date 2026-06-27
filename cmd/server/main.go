package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

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

	// 设置超时，缓解慢速连接（slowloris 类）攻击。
	httpServer := &http.Server{
		Addr:              cfg.BindAddr,
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// 监听中断与 SIGTERM 信号，实现优雅关停。
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("starting server", "addr", cfg.BindAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server exited: %v", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutdown signal received, draining connections...")

	// 给在途请求最多 30 秒完成。
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	slog.Info("server stopped")
}
