package main

import (
	"context"
	"log"
	"net/http"
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

	f, err := os.Open(path + "/config.yaml")
	if err != nil {
		return err
	}
	defer f.Close()

	var cfg wallet.Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return err
	}

	log, err := zap.NewDevelopment()
	if err != nil {
		return err
	}
	defer log.Sync()

	svc := wallet.NewService(cfg)

	r := gin.Default()

	// GET /.well-known/webauthn
	r.GET("/.well-known/webauthn", func(c *gin.Context) {
		origins := struct {
			Origins []string `json:"origins"`
		}{Origins: cfg.PasskeysConfig.Origins}

		c.JSON(http.StatusOK, origins)
	})

	api := r.Group("/api")
	{
		// POST /passkeys/registration/initialize
		{
			endpoint := wallet.InitializeRegistrationEndpoint(svc)
			api.POST("/passkeys/registration/initialize", wallet.InitializeRegistrationHandler(endpoint))
		}

		// POST /passkeys/registration/finalize
		{
			endpoint := wallet.FinalizeRegistrationEndpoint(svc)
			api.POST("/passkeys/registration/finalize", wallet.FinalizeRegistrationHandler(endpoint))
		}

		// POST /passkeys/login/initialize
		{
			endpoint := wallet.InitializeLoginEndpoint(svc)
			api.POST("/passkeys/login/initialize", wallet.InitializeLoginHandler(endpoint))
		}

		// POST /passkeys/login/finalize
		{
			endpoint := wallet.FinalizeLoginEndpoint(svc)
			api.POST("/passkeys/login/finalize", wallet.FinalizeLoginHandler(endpoint))
		}
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
