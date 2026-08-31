package main

import (
	"koctz/internal/router"

	"go.uber.org/zap"
)

func main() {
	log, _ := zap.NewDevelopment()

	var srv router.HTTPServer
	if err := srv.Init(log); err != nil {
		log.Fatal("Init server", zap.Error(err))
	}
	if err := srv.Start(); err != nil {
		log.Fatal("failed to start server", zap.Error(err))
	}
}
