package commands

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tokamak-network/trh-sdk/server/api"
	"github.com/urfave/cli/v3"
)

func ActionStartServer() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		port := cmd.String("port")
		
		// Initialize Gin router
		router := gin.Default()
		
		// Setup API routes
		api.SetupRoutes(router)

		// List all registered routes
		routes := router.Routes()
		fmt.Println("\nAvailable API Routes:")
		fmt.Println("====================")
		for _, route := range routes {
			fmt.Printf("%s %s\n", route.Method, route.Path)
		}
		fmt.Println("====================\n")

		srv := &http.Server{
			Addr:    fmt.Sprintf(":%s", port),
			Handler: router,
		}

		// Start server in goroutine
		go func() {
			fmt.Printf("Server starting on port %s...\n", port)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start server: %v", err)
			}
		}()

		// Wait for interrupt signal
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		fmt.Println("\nShutting down server...")

		// Graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			return fmt.Errorf("server forced to shutdown: %v", err)
		}

		fmt.Println("Server stopped gracefully")
		return nil
	}
}
