![build](https://github.com/cloudnativedude/gitdrops/actions/workflows/go-build-test.yaml/badge.svg)
![build](https://github.com/cloudnativedude/gitdrops/actions/workflows/go-static-analysis.yaml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/cloudnativedude/gitdrops)](https://goreportcard.com/report/github.com/cloudnativedude/gitdrops)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

![gitdrops-logo-centered](https://user-images.githubusercontent.com/41484746/119481306-726e6880-bd4a-11eb-9a3c-e3f3d849423f.png)

# GitDrops 💧
GitOps for DigitalOcean

## What is GitDrops?
A POC tool for declarative management of [DigitalOcean](https://developers.digitalocean.com/) resources inspired by [GitOps](https://www.weave.works/technologies/gitops/) principles and the [Kubernetes Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/). Manage your DigitalOcean account using a single yaml file as the source of truth. Run GitDrops locally to reconcile your DigitalOcean account, or leverage Github Actions to produce an automated GitOps pipeline.

## Why Use GitDrops?
The use case that inspired this project was simple. I wanted to:
* Manage my DigitalOcean Droplets and Volumes from the command line via an easy to use `yaml` spec. 📝
* Use version control to track changes to my Droplets and Volumes. 🔃
* Most importantly - Schedule my consistent DigitalOcean development environment to spin up every morning before I start work! 🚀

Not to mention it was a fun way to get acquainted with the [DigitalOcean API](https://developers.digitalocean.com/documentation/v2/) and while there are other tools and projects out there to do a lot of this stuff - I felt like making my own! 😎 

## How Do I Use GitDrops?

### Try GitDrops Out Locally

* Clone this repo.
* Set an environment variable `DIGITALOCEAN_TOKEN` with your DigitalOcean account authentication token.
* Edit `gitdrops.yaml` locally. 

**Warning**: Please read editing instructions below before running GitDrops.

* Run GitDrops with `make run`

### Run GitDrops From Your Github Account

* Fork this repo to your own Github account.
* Enable Github Actions on your forked repo.
  * Warning: Before enabling, please review and understand the Github Actions of which there are four.
    1. **[Build and Test](https://github.com/cloudnativedude/gitdrops/blob/main/.github/workflows/go-build-test.yaml)**: This action runs unit tests and builds GitDrops on pushes and pull requests to `main`.
    2. **[Go Static Analysis](https://github.com/cloudnativedude/gitdrops/blob/main/.github/workflows/go-static-analysis.yaml)**: This action runs Go Linting checks on pushes and pull requests to `main`.
    3. **[GitDrops Run](https://github.com/cloudnativedude/gitdrops/blob/main/.github/workflows/gitdrops-run.yaml)**: This is where the magic happens! 🧙‍♂️ This action runs GitDrops on pushes of `gitdrops.yaml` to `main`, reconciling your DigitalOcean account to the desired state of `gitdrops.yaml`.
    4. **[GitDrops Schedule](https://github.com/cloudnativedude/gitdrops/blob/main/.github/workflows/gitdrops-schedule.yaml)**: This action copies `gitdrops-update.yaml` to `gitdrops.yaml` and pushes the updated file to `main`. This is useful if you want to schedule your DigitalOcean resources to spin up at a certain time of the day (e.g every morning while you're grabbing a ☕). See [scheduling instructions](#scheduling-gitdrops) for more details. 
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

See [Droplet](https://github.com/cloudnativedude/gitdrops/blob/main/pkg/gitdrops/types.go#L17) type.

##### Update Capabilities

GitDrops only supports Droplet updates for:
* Image rebuild (i.e. changed `droplet.image` in `gitdrops.yaml`)
* Droplet resize (i.e. changed `drople.size` in `gitdrops.yaml`)

Should you wish to change other details about a Droplet, it is necessary to create a new Droplet with your desired details.

#### Volumes

See [Volume](https://github.com/cloudnativedude/gitdrops/blob/main/pkg/gitdrops/types.go#L33) type.

##### Update Capabilities

GitDrops only supports Volume updates for:
* Volume attach (i.e. changed `droplets.volumes` value in `gitdrops.yaml`)
* Volume detach (i.e. changed `droplets.volumes` value in `gitdrops.yaml`)
* Volume resize (i.e. changed `drople.size` in `gitdrops.yaml`)

Should you wish to change other details about a Volume, it is necessary to create a new Volume with your desired details.

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
  sshKeyFingerprints: ["XX:XX:XX:XX:XX..."]
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

##### Things to Note:
* This `gitdrops.yaml` example only has `create` `privileges`. Upon reconciliation of this file, Gitdrops will attempt to create one Droplet (with `volume-1` attached) and two Volumes on the user's DigitalOcean account. No Droplets or Volumes will be updated or deleted. 
* The Droplet `centos-droplet-1` will be configured with user data stored in [`./couldconfig`](https://github.com/cloudnativedude/gitdrops/blob/main/cloudconfig). You can point to any path in the repo to run a custom script on your new Droplet. This example uses a [cloud-config script](https://www.digitalocean.com/community/tutorials/an-introduction-to-cloud-config-scripting), but you can use bash or whatever you like.
* See sample [`gitdrops.yaml`](https://github.com/cloudnativedude/gitdrops/blob/main/gitdrops.yaml).

### Scheduling GitDrops

#### How It Works
1. 📝 Edit [`gitdrops-update.yaml`](https://github.com/cloudnativedude/gitdrops/blob/main/gitdrops-update.yaml) to represent the desired state for your scheduled update. The `gitdrops-update.yaml` follows the same template as `gitdrops.yaml`, it is basically a "future state" for GitDrops to reconcile at a time you can specify in step 3.
2. ⬆️ Commit and push your `gitdrops-update.yaml` file to your forked repo on the `main` branch.
3. ⏲️ Set the [cron time schedule for GitDrops Schedule workflow](https://github.com/cloudnativedude/gitdrops/blob/main/.github/workflows/gitdrops-schedule.yaml#L5). This workflow creates a Git commit replacing `gitdrops.yaml` with your newly created `gitdrops-update.yaml` and pushes the changes to `main` using the Github Actions Bot. Read more [here](https://docs.github.com/en/actions/reference/events-that-trigger-workflows#schedule) about cron time syntax for Github Actions.
4. 💧 Once the GitDrops Schedule workflow has run to completion, the GitDrops Run workflow will begin. This performs the reconciliation of your DigitalOcean account with the newly committed `gitdrops.yaml` spec (i.e what you originally specified in step 1).
