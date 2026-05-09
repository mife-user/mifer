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

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "serve":
			runServer(app)
			return
		case "chat":
			runCLI(app)
			return
		}
	}

	runDefault(app)
}

func runServer(app *bootstrap.Application) {
	runErr := make(chan error, 1)
	go func() { runErr <- app.Run() }()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-runErr:
		if err != nil {
			log.Fatalf("启动服务器失败: %v", err)
		}
	case <-quit:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		log.Printf("应用关闭失败: %v", err)
	}
}

func runDefault(app *bootstrap.Application) {
	runErr := make(chan error, 1)
	go func() { runErr <- app.Run() }()

	cliDone := make(chan error, 1)
	go func() { cliDone <- app.Clier.Run() }()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-runErr:
		if err != nil {
			log.Fatalf("启动服务器失败: %v", err)
		}
	case <-cliDone:
	case <-quit:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		log.Printf("应用关闭失败: %v", err)
	}
}

func runCLI(app *bootstrap.Application) {
	if err := app.Clier.Run(); err != nil {
		log.Fatal("CLI 运行失败: ", err)
	}
}
