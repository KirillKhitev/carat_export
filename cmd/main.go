package main

import (
	"context"
	"fmt"
	"github.com/KirillKhitev/carat_export/internal/config"
	"github.com/KirillKhitev/carat_export/internal/logger"
	"github.com/sirupsen/logrus"
)

// Флаги сборки.
var (
	buildVersion string = "N/A" // Версия сборки
	buildDate    string = "N/A" // Дата сборки
	buildCommit  string = "N/A" // Комментарий сборки
)

func main() {
	printBuildInfo()

	if err := config.Config.Parse(); err != nil {
		panic(err)
	}

	if err := run(); err != nil {
		logger.Log.Logln(logrus.PanicLevel, err)
	}
}

func run() error {
	ctx := context.Background()

	appInstance := newApp()

	if err := appInstance.Bootstrap(); err != nil {
		return fmt.Errorf("bootstrap failed: %w", err)
	}

	logger.Initialize(config.Config.LogLevel)

	logger.Log.WithFields(logrus.Fields{
		"config": config.Config,
	}).Logln(logrus.InfoLevel, "Запустили приложение")

	go appInstance.StartFileServer()
	go appInstance.StartController(ctx)

	return appInstance.CatchTerminateSignal()
}

// printBuildInfo выводит в консоль информацию по сборке.
func printBuildInfo() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}
