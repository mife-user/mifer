package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mifer/cmd/bootstrap"
)

func main() {
	app, err := bootstrap.NewApplication()
	if err != nil {
		log.Fatalf("应用初始化失败: %v", err)
	}

	runErr := make(chan error, 1)
	go func() {
		runErr <- app.Run()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-runErr:
		log.Fatalf("启动服务器失败: %v", err)
	case <-quit:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		log.Printf("应用关闭失败: %v", err)
	}
}
