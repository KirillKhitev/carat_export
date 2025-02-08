package main

import (
	"context"
	"fmt"
	"github.com/KirillKhitev/carat_export/internal/config"
	"github.com/KirillKhitev/carat_export/internal/controller"
	"github.com/KirillKhitev/carat_export/internal/logger"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

type app struct {
	c      *controller.Controller
	server http.Server
}

func newApp() *app {
	instance := &app{
		c: controller.NewController(),
	}

	return instance
}

// StartFileServer запускает файловый сервер для отдачи изображений товаров.
func (a *app) StartFileServer() {
	fs := http.FileServer(http.Dir(config.Config.ImagesPath))

	mux := http.NewServeMux()
	mux.Handle("/images/", http.StripPrefix("/images/", fs))
	mux.HandleFunc("/products.xml", func(w http.ResponseWriter, r *http.Request) {
		logger.Log.Log(logrus.InfoLevel, "Пришли за файлом авито:")

		s := ""
		for h, values := range r.Header {
			s += h + ":\r\n"
			for i, v := range values {
				s += fmt.Sprintf("\t%d: %s", i, v) + "\r\n"
			}
		}

		logger.Log.Log(logrus.InfoLevel, s)
		http.ServeFile(w, r, config.Config.AvitoFilePath)
	})

	a.server = http.Server{
		Addr:    config.Config.ServerURL,
		Handler: mux,
	}

	if err := a.server.ListenAndServe(); err != nil {
		logger.Log.WithFields(logrus.Fields{
			"error": err,
		}).Logln(logrus.FatalLevel, "Ошибка вебсервера при отдаче файла товаров")
	}
}

func (a *app) StartController(ctx context.Context) {
	a.c.Start(ctx)
}

// Bootstrap создает необходимые папки
func (a *app) Bootstrap() error {
	if _, err := os.Stat(config.Config.ImagesPath); err == nil {
		return nil
	}

	if err := os.Mkdir(config.Config.ImagesPath, 0755); err != nil {
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
	if err := a.shutdownServer(); err != nil {
		return err
	}

	if err := a.c.Close(); err != nil {
		return err
	}

	return nil
}

func (a *app) shutdownServer() error {
	shutdownCtx, shutdownRelease := context.WithCancel(context.TODO())
	defer shutdownRelease()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("ошибка при остановке HTTP сервера: %w", err)
	}

	logger.Log.Log(logrus.InfoLevel, "HTTP сервер успешно остановлен.")

	return nil
}

func (a *app) CatchTerminateSignal() error {
	terminateSignals := make(chan os.Signal, 1)

	signal.Notify(terminateSignals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL)

	<-terminateSignals

	if err := a.Close(); err != nil {
		return err
	}

	logger.Log.Logln(logrus.InfoLevel, "Приложение успешно остановлено")

	return nil
}
