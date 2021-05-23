![build](https://github.com/nolancon/gitdrops/actions/workflows/go-build-test.yaml/badge.svg)
![build](https://github.com/nolancon/gitdrops/actions/workflows/go-static-analysis.yaml/badge.svg)

# UNDER CONSTRUCTION 🛠️

# GitDrops
GitOps for DigitalOcean Droplets

## What is GitDrops?
A mechanism for declarative management of [DigitalOcean](https://developers.digitalocean.com/) resources inspired by [GitOps](https://www.weave.works/technologies/gitops/) principles and the [Kubernetes Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/). Manage your DigitalOcean account using a single yaml file as the source of truth. Run GitDrops locally to reconcile your DO account, or leverage Github Actions to produce an automated GitOps pipeline.

## How do I use GitDrops?

### Try GitDrops Out Locally

* Clone this repo.
* Set an environment vairable `DIGITALOCEAN_TOKEN` with your DigitalOcean account authentication token.
* Edit `gitdrops.yaml` locally.
* Run GitDrops with `make run`

### Run GitDrops From Your Github Account

* Fork this repo to your own Github account.
* Enable Github Actions in your forked repo.
  * Warning: Before enabling, please review and understand the Github Actions of which there are four.
    1. Build and Test: This action runs unit tests and builds GitDrops on pushes and pull requests to `main`.
    2. Go Static Analysis: This action runs Go Linting checks on pushes and pull requests to `main`.
    3. Gitdrops Run: This is where the magic happens! 🔥 This action runs GitDrops on pushes of `gitdrops.yaml` to `main`, reconciling with your DigitalOcean account.
  * On your forked GitDrops repo, click the Actions tab and enable Github Actions
  
![github-actions-enable](https://user-images.githubusercontent.com/41484746/119275751-ac3a5480-bc0e-11eb-92aa-92a2f85d3e7b.PNG)

* Create a Github secret named `DIGITALOCEAN_TOKEN` with your DigitalOcean account authentication token.
* Edit `gitdrops.yaml` locally.
* Git commit & push your edited `gitdrops.yaml` to your forked repo on the `main` branch.
