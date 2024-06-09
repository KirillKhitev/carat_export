package main

import (
	"context"
	"github.com/KirillKhitev/carat_export/internal/config"
	"github.com/KirillKhitev/carat_export/internal/controller"
	"github.com/KirillKhitev/carat_export/internal/logger"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
)

type app struct {
	c *controller.Controller
}

func newApp() *app {
	instance := &app{
		c: controller.NewController(),
	}

	return instance
}

func (a *app) StartController(ctx context.Context) {
	a.c.Start(ctx)
}

// Bootstrap создает необходимые папки
func (a *app) Bootstrap() error {
	if _, err := os.Stat(config.Config.ImagesDir); err == nil {
		return nil
	}

	if err := os.Mkdir(config.Config.ImagesDir, 0644); err != nil {
		return err
	}

	if _, err := os.Stat(config.Config.LogDir); err == nil {
		return nil
	}

	if err := os.Mkdir(config.Config.LogDir, 0644); err != nil {
		return err
	}

	return nil
}

func (a *app) Close() error {
	if err := a.c.Close(); err != nil {
		return err
	}

	return nil
}

func (a *app) CatchTerminateSignal() error {
	terminateSignals := make(chan os.Signal, 1)

	signal.Notify(terminateSignals, syscall.SIGINT, syscall.SIGTERM)

	<-terminateSignals

	if err := a.Close(); err != nil {
		return err
	}

	logger.Log.Logln(logrus.InfoLevel, "Приложение успешно остановлено")

	return nil
}
