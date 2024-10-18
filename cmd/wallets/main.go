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

	"github.com/flarexio/wallets"
)

func main() {
	app := &cli.App{
		Name: "wallets",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "path",
				EnvVars: []string{"WALLETS_PATH"},
			},
			&cli.IntFlag{
				Name:    "port",
				EnvVars: []string{"WALLETS_SERVICE_PORT"},
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

		path = homeDir + "/.flarex/wallets"
	}

	f, err := os.Open(path + "/config.yaml")
	if err != nil {
		return err
	}
	defer f.Close()

	var cfg wallets.Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return err
	}

	log, err := zap.NewDevelopment()
	if err != nil {
		return err
	}
	defer log.Sync()

	svc := wallets.NewService(cfg)

	r := gin.Default()

	// POST /passkey/start-registration
	{
		endpoint := wallets.StartRegistrationEndpoint(svc)
		r.POST("/passkey/start-registration", wallets.StartRegistrationHandler(endpoint))
	}

	// POST /passkey/finalize-registration
	{
		endpoint := wallets.FinishRegistrationEndpoint(svc)
		r.POST("/passkey/finalize-registration", wallets.FinishRegistrationHandler(endpoint))
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
