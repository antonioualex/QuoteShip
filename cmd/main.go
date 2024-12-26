package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"cargoplot/app"
	"cargoplot/persistence"
	"cargoplot/presentation"
)

// Constants for server timeouts
const (
	defaultAddr            = ":3142"           // Define default http address
	defaultUpdateThreshold = "1000"            //Values to send before each price index retrieval (default 1000)
	readTimeout            = 5 * time.Second   // Define http server read timeout
	writeTimeout           = 10 * time.Second  // Define http server write timeout
	idleTimeout            = 120 * time.Second // Define http server idle timeout
	shutdownTimeout        = 10 * time.Second  // Define http server shutdown timeout
)

func main() {
	// Fetch the server address from an environment variable or use the default value
	addr := getEnv("HTTP_SERVER_ADDR", defaultAddr)

	updateThreshold := getEnv("UPDATE_THRESHOLD", defaultUpdateThreshold)

	// Convert the updateThreshold to an integer
	updateThresholdInt, err := strconv.Atoi(updateThreshold)
	if err != nil {
		slog.Error("failed to convert update threshold to integer", "error", err.Error())
		cleanExit(1)
	}

	// Create a context that listens for SIGINT or SIGTERM signals for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop() // Ensure resources associated with the signal context are released

	if err := run(ctx, addr, updateThresholdInt); err != nil {
		slog.Error("failed to run the application", "error", err.Error())
		// Call a function to cleanly exit
		cleanExit(1)
	}
}

func run(ctx context.Context, addr string, updateThreshold int) error {
	slog.Info("Starting application...")
	slog.Info("http server address", slog.String("addr", addr))
	slog.Info("update threshold value", slog.Int("threshold", updateThreshold))

	// Initialize the shipment repository
	shipmentRepository, err := persistence.NewShipmentOfferRepository(ctx, updateThreshold)
	if err != nil {
		slog.Error("failed to create shipment repository", "error", err.Error())
		return err
	}
	// Initialize the shipment service
	shipmentService, err := app.CreateShipmentService(shipmentRepository)
	if err != nil {
		slog.Error("failed to create shipment service", "error", err.Error())
		return err
	}

	// Create an HTTP request multiplexer (router) and register routes
	mux := http.NewServeMux()

	// Shipment handler is created within the routes registration function
	presentation.RegisterRoutes(mux, shipmentService)

	// Configure the HTTP server with timeouts and base context
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
		BaseContext: func(net.Listener) context.Context {
			// Attach the signal context to the server's lifecycle
			return ctx
		},
	}

	// Start the HTTP server in a separate goroutine
	go func() {
		slog.Info("starting HTTP server", slog.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Log and exit if the server fails to start unexpectedly
			slog.Error("HTTP server shutdown unexpectedly", "error", err.Error())
			cleanExit(1)
		}
		slog.Info("HTTP server shut down gracefully")
	}()

	// Block until a shutdown signal is received (SIGINT, SIGTERM)
	<-ctx.Done()
	slog.Info("shutdown signal received - shutting down service")

	// Create a timeout context for shutting down the server gracefully
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel() // Ensure resources associated with the timeout context are released

	// Attempt to gracefully shut down the HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			// Log a timeout error if shutdown exceeds the allowed time
			slog.Error("server shutdown timed out", "error", err.Error())
		} else {
			// Log other errors during shutdown
			slog.Error("failed to gracefully shutdown HTTP server", "error", err.Error())
		}
		cleanExit(3) // Exit with code 3 for shutdown failure
	}

	slog.Info("server shutdown complete")

	return nil
}

// cleanExit is used to exit the application while ensuring deferred functions are executed
func cleanExit(code int) {
	// Allow deferred functions to run before exiting
	defer os.Exit(code)
}

// getEnv is a helper function to fetch an environment variable or return a default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
