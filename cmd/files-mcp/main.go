package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agynio/files-mcp/internal/config"
	"github.com/agynio/files-mcp/internal/filesclient"
	"github.com/agynio/files-mcp/internal/mcpserver"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("files-mcp: %v", err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}

	conn, err := filesclient.Dial(cfg.GatewayAddress, cfg.APIToken)
	if err != nil {
		return fmt.Errorf("dial gateway: %w", err)
	}
	defer conn.Close()

	server := mcpserver.New(filesclient.New(conn), mcpserver.Options{MaxFileSize: cfg.MaxFileSize})

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.MCPPort),
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("http shutdown: %v", err)
		}
	}()

	log.Printf("files-mcp listening on :%d", cfg.MCPPort)
	if err := httpServer.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve http: %w", err)
	}
	return nil
}
