package main

import (
	"context"

	"github.com/nolancon/gitdrops/pkg/reconcile"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
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
