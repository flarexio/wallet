package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v2"
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
	app := &cli.App{
		Name: "wallet",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				EnvVars: []string{"WALLET_PATH"},
			},
			&cli.IntFlag{
				Name:    "port",
				EnvVars: []string{"WALLET_SERVICE_PORT"},
				Value:   8080,
			},
		},
		Action: run,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3000*time.Millisecond)
	defer cancel()

	<-ctx.Done()
}

func run(cli *cli.Context) error {
	path := cli.String("path")
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

	ctx := context.Background()

	http.Init(cfg.JWT)

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

	port := cli.Int("port")
	go r.Run(":" + strconv.Itoa(port))

	// Setup signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sign := <-quit // Wait for a termination signal

	log.Info("graceful shutdown", zap.String("singal", sign.String()))
	return nil
}
