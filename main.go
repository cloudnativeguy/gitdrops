package main

import (
	"context"
	"time"

	"github.com/nolancon/gitdrops/pkg/reconcile"
)

const (
	// 15s as there is a 10s sleep between reconciliations to allow updates
	duration = 15 * time.Second
)

func main() {
	//	var ctx context.Context
	//	var ctxCancelFunc context.CancelFunc
	//	var timeUntilContextDeadline = time.Now().Add(duration)
	//
	//	ctx, ctxCancelFunc = context.WithDeadline(context.Background(), timeUntilContextDeadline)
	//	defer ctxCancelFunc()

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
