package main

import (
	"context"
	"time"

	"github.com/nolancon/gitdrops/pkg/reconcile"
)

const (
	shortDuration = 10 * time.Second
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), shortDuration)
	defer cancel()
	reconcileDroplets, err := reconcile.NewReconciler(ctx)
	if err != nil {
		return
	}
	err = reconcileDroplets.Reconcile(ctx)
	if err != nil {
		return
	}
}
