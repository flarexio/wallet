package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/flarexio/wallet"
	"github.com/flarexio/wallet/conf"
	"github.com/flarexio/wallet/persistence"

	"github.com/flarexio/identity/passkeys"
	"github.com/flarexio/identity/policy"

	identityHTTP "github.com/flarexio/identity/transport/http"
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

	identityHTTP.Init(
		cfg.Identity.BaseURL,
		cfg.Identity.JWT.Audiences[0],
		cfg.Identity.JWT.Secret,
	)

	ctx := context.Background()
	policy, err := policy.NewRegoPolicy(ctx, conf.Path)
	if err != nil {
		return err
	}

	auth := identityHTTP.Authorizator(policy)

	{
		// GET /wallets/:user
		{
			endpoint := wallet.WalletEndpoint(svc)
			r.GET("/wallets/:user", auth("wallet::accounts.get", identityHTTP.Owner),
				wallet.WalletHandler(endpoint))
		}

		// POST /wallets/:user/signature/initialize
		{
			endpoint := wallet.InitializeSignatureEndpoint(svc)
			r.POST("/wallets/:user/signature/initialize", auth("wallet::accounts.get", identityHTTP.Owner),
				wallet.InitializeSignatureHandler(endpoint))
		}

		// POST /wallets/:user/signature/finalize
		{
			endpoint := wallet.FinalizeSignatureEndpoint(svc)
			r.POST("/wallets/:user/signature/finalize", auth("wallet::accounts.get", identityHTTP.Owner),
				wallet.FinalizeSignatureHandler(endpoint))
		}

		// POST /wallets/:user/transaction/initialize
		{
			endpoint := wallet.InitializeTransactionEndpoint(svc)
			r.POST("/wallets/:user/transaction/initialize", auth("wallet::accounts.get", identityHTTP.Owner),
				wallet.InitializeTransactionHandler(endpoint))
		}

		// POST /wallets/:user/transaction/finalize
		{
			endpoint := wallet.FinalizeTransactionEndpoint(svc)
			r.POST("/wallets/:user/transaction/finalize", auth("wallet::accounts.get", identityHTTP.Owner),
				wallet.FinalizeTransactionHandler(endpoint))
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
