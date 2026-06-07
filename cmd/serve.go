package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/syawalqi/arahin/config"
	"github.com/syawalqi/arahin/web"
)

// Serve starts the ARAHIN web server.
func Serve(cfg *config.Config) error {
	srv := web.NewServer(cfg)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("server: %v", err)
		}
	}()

	fmt.Fprintf(os.Stderr, "[ARAHIN] Web server running on http://0.0.0.0:%s\n", cfg.Route.Port)
	fmt.Fprintf(os.Stderr, "[ARAHIN] Press Ctrl+C to stop\n")

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}
