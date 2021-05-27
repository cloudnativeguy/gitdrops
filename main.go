package main

import (
	"context"
	"log"

	"github.com/nolancon/gitdrops/pkg/reconcile"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reconcileObjects, err := reconcile.NewReconciler(ctx)
	if err != nil {
		log.Fatalf("failed to create new Reconciler %v", err)
	}
	err = reconcileObjects.Reconcile(ctx)
	if err != nil {
		log.Fatalf("failed to Reconcile %v", err)
	}
}
