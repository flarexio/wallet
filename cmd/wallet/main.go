package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/flarexio/core/policy"
	"github.com/flarexio/identity/passkeys"
	"github.com/flarexio/wallet"
	"github.com/flarexio/wallet/conf"
	"github.com/flarexio/wallet/persistence"
	"github.com/flarexio/wallet/transport/http"
)

func main() {
	cmd := &cli.Command{
		Name: "wallet",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				Sources: cli.EnvVars("WALLET_PATH"),
			},
			&cli.IntFlag{
				Name:    "port",
				Sources: cli.EnvVars("WALLET_SERVICE_PORT"),
				Value:   8080,
			},
		},
		Action: run,
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := cmd.Run(ctx, os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	path := cmd.String("path")
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		path = homeDir + "/.flarex/wallet"
	}

	conf.Path = path

	f, err := os.Open(conf.Path + "/config.yaml")
	if err != nil {
		return err
	}

	var cfg conf.Config
	err = yaml.NewDecoder(f).Decode(&cfg)
	if err != nil {
		return err
	}

	log, err := zap.NewDevelopment()
	if err != nil {
		return err
	}
	defer log.Sync()

	passkeysSvc, err := passkeys.NewService(cfg.Passkeys)
	if err != nil {
		return err
	}

	repo, err := persistence.NewAccountRepository(cfg.Persistence)
	if err != nil {
		return err
	}
	defer repo.Close()

	svc, err := wallet.NewService(repo, passkeysSvc, cfg)
	if err != nil {
		return err
	}
	defer svc.Close()

	r := gin.Default()

	http.Init(ctx, cfg.JWT)

	permissionsPath := filepath.Join(path, "permissions.json")
	policy, err := policy.NewRegoPolicy(ctx, permissionsPath)
	if err != nil {
		return err
	}

	auth := http.JWTAuthorizator(policy)

	api := r.Group("/wallet/v1")
	{
		// GET /health
		api.GET("/health", http.HealthHandler)

		// GET /accounts/:user
		{
			endpoint := wallet.WalletEndpoint(svc)
			api.GET("/accounts/:user", auth("wallet::accounts.get", http.Owner),
				http.WalletHandler(endpoint))
		}

		// POST /accounts/:user/message-signatures
		{
			endpoint := wallet.InitializeSignMessageEndpoint(svc)
			api.POST("/accounts/:user/message-signatures", auth("wallet::accounts.get", http.Owner),
				http.InitializeSignMessageHandler(endpoint))
		}

		// PUT /accounts/:user/message-signatures
		{
			endpoint := wallet.FinalizeSignMessageEndpoint(svc)
			api.PUT("/accounts/:user/message-signatures", auth("wallet::accounts.get", http.Owner),
				http.FinalizeSignMessageHandler(endpoint))
		}

		// POST /accounts/:user/transaction-signatures
		{
			endpoint := wallet.InitializeSignTransactionEndpoint(svc)
			api.POST("/accounts/:user/transaction-signatures", auth("wallet::accounts.get", http.Owner),
				http.InitializeSignTransactionHandler(endpoint))
		}

		// PUT /accounts/:user/transaction-signatures
		{
			endpoint := wallet.FinalizeSignTransactionEndpoint(svc)
			api.PUT("/accounts/:user/transaction-signatures", auth("wallet::accounts.get", http.Owner),
				http.FinalizeSignTransactionHandler(endpoint))
		}

		// POST /sessions
		{
			endpoint := wallet.CreateSessionEndpoint(svc)
			api.POST("/sessions", http.CreateSessionHandler(endpoint))
		}

		// GET /sessions/:session
		{
			endpoint := wallet.SessionDataEndpoint(svc)
			api.GET("/sessions/:session", http.SessionDataHandler(endpoint))
		}

		// POST /sessions/:session/ack
		{
			endpoint := wallet.AckSessionEndpoint(svc)
			api.POST("/sessions/:session/ack", http.AckSessionHandler(endpoint))
		}
	}

	port := cmd.Int("port")
	go r.Run(":" + strconv.Itoa(port))

	// Setup signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sign := <-quit // Wait for a termination signal

	log.Info("graceful shutdown", zap.String("singal", sign.String()))
	return nil
}
