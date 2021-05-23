![build](https://github.com/nolancon/gitdrops/actions/workflows/go-build-test.yaml/badge.svg)
![build](https://github.com/nolancon/gitdrops/actions/workflows/go-static-analysis.yaml/badge.svg)

# UNDER CONSTRUCTION 🛠️

# GitDrops 💧
GitOps for DigitalOcean Droplets

## What is GitDrops?
A POC tool for declarative management of [DigitalOcean](https://developers.digitalocean.com/) resources inspired by [GitOps](https://www.weave.works/technologies/gitops/) principles and the [Kubernetes Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/). Manage your DigitalOcean account using a single yaml file as the source of truth. Run GitDrops locally to reconcile your DO account, or leverage Github Actions to produce an automated GitOps pipeline.

## How do I use GitDrops?

### Try GitDrops Out Locally

* Clone this repo.
* Set an environment variable `DIGITALOCEAN_TOKEN` with your DigitalOcean account authentication token.
* Edit `gitdrops.yaml` locally.
* Run GitDrops with `make run`

### Run GitDrops From Your Github Account

* Fork this repo to your own Github account.
* Enable Github Actions on your forked repo.
  * Warning: Before enabling, please review and understand the Github Actions of which there are four.
    1. **[Build and Test](https://github.com/cloudnativedude/gitdrops/blob/main/.github/workflows/go-build-test.yaml)**: This action runs unit tests and builds GitDrops on pushes and pull requests to `main`.
    2. **[Go Static Analysis](https://github.com/cloudnativedude/gitdrops/blob/main/.github/workflows/go-static-analysis.yaml)**: This action runs Go Linting checks on pushes and pull requests to `main`.
    3. **[GitDrops Run](https://github.com/cloudnativedude/gitdrops/blob/main/.github/workflows/gitdrops-run.yaml)**: This is where the magic happens! 🔥 This action runs GitDrops on pushes of `gitdrops.yaml` to `main`, reconciling your DigitalOcean account to the desired state of `gitdrops.yaml`.
    4. **[GitDrops Schedule](https://github.com/cloudnativedude/gitdrops/blob/main/.github/workflows/gitdrops-schedule.yaml)**: This action copies `gitdrops-schedule.yaml` to `gitdrops.yaml` and pushes the updated file to `main`. This is useful if you want to schedule your DigitalOcean resources to spin up at a certain time of the day (e.g every morning while you're grabbing a ☕). See scheduling instructions for more details. 
  * On your forked GitDrops repo, click the Actions tab and enable Github Actions
  
![github-actions-enable](https://user-images.githubusercontent.com/41484746/119275751-ac3a5480-bc0e-11eb-92aa-92a2f85d3e7b.PNG)

* Create a Github secret named `DIGITALOCEAN_TOKEN` with your DigitalOcean account authentication token.
* Edit `gitdrops.yaml` locally.
* Git commit & push your edited `gitdrops.yaml` to your forked repo on the `main` branch.

### Editing `gitdrops.yaml`

This yaml file will represent the desired state for your DigitalOcean account (initial support for Droplets and Volumes only).

#### Privileges

GitDrops can be configured with `true` or `false` `privileges` for `create`, `update` and `delete` on Droplets and Volumes.

**Warning**: Should `gitdrops.yaml` be afforded `delete` `privileges`, Droplets and Volumes not listed in `gitdrops.yaml` but running on DigitalOcean will be deleted upon reconciliation.

#### Droplets

See [Droplet](https://github.com/cloudnativedude/gitdrops/blob/30fa3b45c68baf99524dcb1be4cdd81819717276/pkg/gitdrops/types.go#L17) type.

#### Volumes

See [Volume](https://github.com/cloudnativedude/gitdrops/blob/30fa3b45c68baf99524dcb1be4cdd81819717276/pkg/gitdrops/types.go#L33) type.

#### Example

```yaml
privileges:
  create: true
  update: false
  delete: false
droplets:
- name: centos-droplet-1
  region: nyc3
  size: s-1vcpu-1gb
  image: centos-8-x64         
  backups: false
  ipv6: false
  volumes: ["volume-1"]
  sshKeyFingerprint: "XX:XX:XX:XX:XX..."
  userData:
    path: cloudconfig
volumes:
- name: volume-1
  region: nyc3
  sizeGigaBytes: 100
- name: volume-2
  region: nyc3
  sizeGigaBytes: 100
```

This `gitdrops.yaml` only has `create` `privileges`. Upon reconciliation of this file, Gitdrops will attempt to create one Droplet (with `volume-1` attached) and two Volumes on the user's DigitalOcean account. The Droplet `centos-droplet-1` will be configured with user data stored in `./couldconfig`. See sample [`gitdrops.yaml`](https://github.com/cloudnativedude/gitdrops/blob/main/gitdrops.yaml).

### Scheduling GitDrops

Currently, the simplest way to schedule GitDrops is by setting [cron times](https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions#onschedule) for [GitDrops Schedule](https://github.com/cloudnativedude/gitdrops/blob/30fa3b45c68baf99524dcb1be4cdd81819717276/.github/workflows/gitdrops-schedule.yaml#L5) and [GitDrops Run](https://github.com/cloudnativedude/gitdrops/blob/30fa3b45c68baf99524dcb1be4cdd81819717276/.github/workflows/gitdrops-run.yaml#L5) workflows.

#### How It Works
1. Edit [`gitdrops-schedule.yaml`](https://github.com/cloudnativedude/gitdrops/blob/main/gitdrops-schedule.yaml) to represent the desired state for your scheduled update.
2. Set the [cron time schedule for GitDrops Schedule workflow](https://github.com/cloudnativedude/gitdrops/blob/30fa3b45c68baf99524dcb1be4cdd81819717276/.github/workflows/gitdrops-schedule.yaml#L5). Note: This workflow must run **before** GitDrops Run in order for the updated `gitdrops.yaml` to be reconciled. Also, although a push to `main` is performed by this workflow, it will not trigger a GitDrops Run workflow because separate Github Actions cannot trigger each other (this is expected Github Actions behaviour).
3. Set the [cron time schedule for GitDrops Run workflow](https://github.com/cloudnativedude/gitdrops/blob/30fa3b45c68baf99524dcb1be4cdd81819717276/.github/workflows/gitdrops-run.yaml#L5). Note: Github Actions running on schedule are often delayed. To ensure Gitdrops Schedule workflow has already run and `gitdrops.yaml` has been updated, allow up to 1 hour between GitDrops Schedule and GitDrops Run workflows.
