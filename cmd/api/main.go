package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"arca-invoice-proxy/internal/application/authentication"
	"arca-invoice-proxy/internal/application/invoice"
	"arca-invoice-proxy/internal/config"
	"arca-invoice-proxy/internal/domain/credential"
	"arca-invoice-proxy/internal/infrastructure/postgres"
	"arca-invoice-proxy/internal/interfaces/http/handlers"
	"arca-invoice-proxy/internal/interfaces/http/middleware"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := postgres.NewPool(ctx, &cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	userRepo := postgres.NewUserRepository(pool)
	apiKeyRepo := postgres.NewAPIKeyRepository(pool)
	credentialRepo := postgres.NewARCACredentialRepository(pool)
	invoiceRepo := postgres.NewInvoiceRepository(pool)

	authService := authentication.NewService(apiKeyRepo, cfg.Auth.APIKeyPepper)

	arcaClient := newMockARCAClient()
	environment := credential.Environment(cfg.ARCA.Environment)
	invoiceService := invoice.NewService(invoiceRepo, userRepo, credentialRepo, arcaClient, environment)

	invoiceHandler := handlers.NewInvoiceHandler(invoiceService)
	apiKeyHandler := handlers.NewAPIKeyHandler(authService)
	healthHandler := handlers.NewHealthHandler()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", healthHandler.Health)
	mux.HandleFunc("GET /ready", healthHandler.Ready)

	protected := http.NewServeMux()
	protected.HandleFunc("POST /v1/facturar", invoiceHandler.IssueInvoice)
	protected.HandleFunc("POST /v1/api-keys", apiKeyHandler.CreateAPIKey)
	protected.HandleFunc("GET /v1/api-keys", apiKeyHandler.ListAPIKeys)
	protected.HandleFunc("DELETE /v1/api-keys/{id}", apiKeyHandler.RevokeAPIKey)

	mux.Handle("/v1/", middleware.RequestID()(middleware.Authentication(authService)(protected)))

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

type mockARCAClient struct{}

func newMockARCAClient() *mockARCAClient {
	return &mockARCAClient{}
}

func (m *mockARCAClient) GetLastVoucher(ctx context.Context, cuit, invoiceType string, pointOfSale int) (int64, error) {
	return 0, nil
}

func (m *mockARCAClient) CreateVoucher(ctx context.Context, request credential.VoucherRequest) (*credential.VoucherResponse, error) {
	return &credential.VoucherResponse{
		CAE:               "12345678901234",
		CAEExpirationDate: time.Now().UTC().AddDate(0, 0, 10).Format("20060102"),
		VoucherNumber:     1,
		VoucherType:       11,
		Result:            "A",
		Observations:      []credential.VoucherObservation{},
	}, nil
}
