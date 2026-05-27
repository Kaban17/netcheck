package main

import (
	"context"
	"os"

	"netcheck/internal/infrastructure/network"
	"netcheck/internal/ui/cli"
	"netcheck/internal/usecase"
)

func main() {
	// 1. Dependency Injection
	repo := network.NewNetworkRepository()
	uc := usecase.NewNetworkInteractor(repo)
	presenter := cli.NewPresenter()
	handler := cli.NewHandler(uc, presenter)

	// 2. Execution
	ctx := context.Background()
	handler.Handle(ctx, os.Args[1:])
}
