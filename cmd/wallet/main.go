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

	repo, err := persistence.NewWalletRepository(cfg.Persistences)
	if err != nil {
		return err
	}
	defer repo.Close()

	svc, err := wallet.NewService(repo, cfg)
	if err != nil {
		return err
	}
	defer svc.Close()

	r := gin.Default()

	// POST /wallets
	{
		endpoint := wallet.WalletEndpoint(svc)
		r.POST("/wallets", wallet.WalletHandler(endpoint))
	}

	r.StaticFS("/app", gin.Dir("./app/dist/app/browser", false))
	r.NoRoute(func(c *gin.Context) {
		c.File("./app/dist/app/browser/index.html")
	})

	port := cli.Int("port")
	go r.Run(":" + strconv.Itoa(port))

	// Setup signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sign := <-quit // Wait for a termination signal

	log.Info("graceful shutdown", zap.String("singal", sign.String()))
	return nil
}
