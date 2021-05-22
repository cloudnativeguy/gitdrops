![build](https://github.com/nolancon/gitdrops/actions/workflows/go-build-test.yaml/badge.svg)
![build](https://github.com/nolancon/gitdrops/actions/workflows/go-static-analysis.yaml/badge.svg)

# UNDER CONSTRUCTION 🛠️

# GitDrops
GitOps for DigitalOcean Droplets

## What is GitDrops?
A mechanism for declarative management of [DigitalOcean](https://developers.digitalocean.com/) resources inspired by [GitOps](https://www.weave.works/technologies/gitops/) principles and the [Kubernetes Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/). Manage your DigitalOcean account using a single yaml file as the source of truth. Run GitDrops locally to reconcile your DO account, or leverage Github Actions to produce an automated GitOps pipeline.

## How do I use GitDrops?

### Try GitDrops out locally

* Clone this repo.
* Set an environment vairable `DIGITALOCEAN_TOKEN` with your DigitalOcean account authentication token.
* Edit `gitdrops.yaml` locally.
* Run GitDrops with `make run`

### Run GitDrops from your Github account

* Fork this repo to your own Github account.
* Include Github Actions in forked repo.
* Create a Github secret named `DIGITALOCEAN_TOKEN` with your DigitalOcean account authentication token.
* Edit `gitdrops.yaml` locally.
* Git commit & push your edited `gitdrops.yaml` to your forked repo on the `main` branch.
