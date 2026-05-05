package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/klausklausen/external-dns-webhook-technitium/internal/technitium"
	"github.com/klausklausen/external-dns-webhook-technitium/internal/webhook"
	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/external-dns/endpoint"
)

var (
	// Technitium configuration
	technitiumURL      = flag.String("technitium-url", getEnv("TECHNITIUM_URL", "http://localhost:5380"), "Technitium DNS server URL")
	technitiumUser     = flag.String("technitium-user", getEnv("TECHNITIUM_USER", "admin"), "Technitium username")
	technitiumPassword = flag.String("technitium-password", getEnv("TECHNITIUM_PASSWORD", "admin"), "Technitium password")
	technitiumToken    = flag.String("technitium-token", getEnv("TECHNITIUM_TOKEN", ""), "Technitium API token (if set, user/password are ignored)")

	// Webhook configuration
	webhookAddr = flag.String("webhook-addr", getEnv("WEBHOOK_ADDR", ":8888"), "Webhook server listen address")

	// DNS configuration
	domainFilter = flag.String("domain-filter", getEnv("DOMAIN_FILTER", ""), "Comma-separated list of domain filters")
	dryRun       = flag.Bool("dry-run", getEnv("DRY_RUN", "false") == "true", "Run in dry-run mode (no changes will be made)")

	// Logging
	logLevel  = flag.String("log-level", getEnv("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
	logFormat = flag.String("log-format", getEnv("LOG_FORMAT", "text"), "Log format (text, json)")
)

func main() {
	flag.Parse()

	// Setup logging
	setupLogging(*logLevel, *logFormat)

	log.Info("Starting external-dns-webhook-technitium")
	log.Infof("Technitium URL: %s", *technitiumURL)
	log.Infof("Webhook address: %s", *webhookAddr)
	log.Infof("Dry run: %v", *dryRun)

	// Create Technitium client
	var client *technitium.Client
	var err error

	if *technitiumToken != "" {
		log.Info("Using API token authentication")
		client = technitium.NewClient(*technitiumURL, *technitiumToken)
	} else {
		log.Info("Using username/password authentication")
		client, err = technitium.NewClientWithAuth(*technitiumURL, *technitiumUser, *technitiumPassword)
		if err != nil {
			log.Fatalf("Failed to authenticate with Technitium: %v", err)
		}
	}

	// Verify connection
	if err := client.HealthCheck(); err != nil {
		log.Fatalf("Failed to connect to Technitium: %v", err)
	}
	log.Info("Successfully connected to Technitium DNS server")

	// Create domain filter
	filter := createDomainFilter(*domainFilter)
	if len(filter.Filters) > 0 {
		log.Infof("Domain filter: %v", filter.Filters)
	} else {
		log.Info("No domain filter configured (all domains will be managed)")
	}

	// Create provider
	provider, err := webhook.NewTechnitiumProvider(client, filter, *dryRun)
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	// Create webhook server
	server := webhook.NewServer(provider, *webhookAddr)

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		log.Infof("Received signal: %v", sig)
	case err := <-errChan:
		log.Errorf("Server error: %v", err)
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Errorf("Error during shutdown: %v", err)
		os.Exit(1)
	}

	log.Info("Shutdown complete")
}

// setupLogging configures the logging system
func setupLogging(level, format string) {
	// Set log level
	switch strings.ToLower(level) {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn", "warning":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}

	// Set log format
	if strings.ToLower(format) == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp: true,
		})
	}
}

// createDomainFilter creates a domain filter from a comma-separated string
func createDomainFilter(filterStr string) endpoint.DomainFilter {
	if filterStr == "" {
		return endpoint.DomainFilter{}
	}

	filters := strings.Split(filterStr, ",")
	var trimmedFilters []string
	for _, f := range filters {
		if trimmed := strings.TrimSpace(f); trimmed != "" {
			trimmedFilters = append(trimmedFilters, trimmed)
		}
	}

	return endpoint.NewDomainFilter(trimmedFilters)
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
