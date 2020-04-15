package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	"github.com/vkg/tetris"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type config struct {
	Address string `envconfig:"ADDRESS"`
	HostKey string `envconfig:"HOST_KEY"`
	Debug   bool   `envconfig:"DEBUG" default:"false"`
}

func main() {
	var conf config
	if err := envconfig.Process("", &conf); err != nil {
		log.Fatal("failed to parse env config", err)
	}

	logc := zap.NewProductionConfig()
	if conf.Debug {
		logc.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	} else {
		logc.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	logc.DisableStacktrace = true
	logger, err := logc.Build()
	if err != nil {
		log.Fatal("failed to build logger", err)
	}
	defer logger.Sync()

	logger.Info("initializing server...", zap.String("addr", conf.Address))

	ctx, cancel := context.WithCancel(context.Background())
	keyRegister := tetris.NewGithubKeyRegister(logger)
	server, err := tetris.NewServer(logger, conf.Address, []byte(conf.HostKey), keyRegister)
	if err != nil {
		logger.Error("failed to new server", zap.Error(err))
		os.Exit(1)
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return server.Serve(ctx)
	})

	// Waiting for SIGTERM or Interrupt signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, os.Interrupt)
	select {
	case <-sigCh:
		logger.Debug("received SIGTERM, exiting server gracefully")
	case <-ctx.Done():
	}

	cancel()
	server.Close()

	if err := eg.Wait(); err != nil {
		logger.Error("server results in failure", zap.Error(err))
	}

	logger.Info("server closed", zap.Error(err))
}
