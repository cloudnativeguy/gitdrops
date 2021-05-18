package main

import (
	"context"
	"time"

	"github.com/nolancon/gitdrops/pkg/reconcile"
)

const (
	shortDuration    = 10 * time.Second
	gitdropsYamlPath = "./gitdrops.yaml"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), shortDuration)
	defer cancel()
	reconcileDroplets, err := reconcile.NewReconcileDroplets(ctx)
	if err != nil {
		return
	}
	err = reconcileDroplets.Reconcile(ctx)
	if err != nil {
		return
	}
}
