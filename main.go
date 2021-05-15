package main

import (
	"context"
	"time"

	"github.com/nolancon/gitdrops/pkg/dolocal"
	"github.com/nolancon/gitdrops/pkg/reconcile"

	"github.com/digitalocean/godo"
)

const (
	shortDuration    = 10 * time.Second
	gitdropsYamlPath = "./gitdrops.yaml"
)

type ReconcileDroplets struct {
	client                     *godo.Client
	activeDroplets             []godo.Droplet
	localDropletCreateRequests []dolocal.LocalDropletCreateRequest
}

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
