# Source Code Manager operator for Kubernetes

A Kubernetes operator built with [Operator SDK](https://sdk.operatorframework.io/) that manages GitLab resources declaratively through Custom Resources. This project was created as a demo for a talk on building Kubernetes operators in Go, presented at the [Golang Lyon Meetup](https://www.meetup.com/golang-lyon/).

## What it does

This operator introduces three Custom Resources:

- **`GitlabGroup`**: mirrors a GitLab group and syncs its metadata (path, visibility, members, subgroups, projects) into the CR status.
- **`GitlabProject`**: creates and tracks a GitLab project under a parent group, including last commit and pipeline status on a tracked branch.
- **`Notifier`**: sends notifications (Discord, with Slack/Telegram stubs) on pipeline and commit events for linked projects.

## Prerequisites

You need the following installed on your machine **before** cloning this repository:

| Tool | Purpose | Install |
|------|---------|---------|
| **Docker** | Container runtime (required by Kind and for building images) | [docs.docker.com](https://docs.docker.com/get-docker/) |
| **Determinate Nix** | Reproducible dev shell providing Go, kubectl, Kind, Helm, operator-sdk, etc. | [docs.determinate.systems](https://docs.determinate.systems/) |
| **direnv** | Automatically loads the Nix dev shell when you enter the project directory | [direnv.net/docs/installation](https://direnv.net/docs/installation.html) |

Once all three are installed, clone the repo, `cd` into it, and run:

```bash
direnv allow
```

This will activate the Nix dev shell and make all required tools (Go, kubectl, Kind, Helm, operator-sdk, gcloud) available in your PATH.

## Quick start

### 1. Set up the demo cluster

```bash
make setup-cluster
```

This creates a Kind cluster, installs cert-manager and External Secrets Operator, and configures a `ClusterSecretStore` for GCP Secret Manager.

> **Note:** Before running this, update `config/eso/cluster-secret-store.yaml` with your GCP project ID and update `demo.Makefile` with the path to your GCP service account credentials file.

### 2. Build and deploy the operator

```bash
make deploy
```

### 3. Apply sample resources

```bash
kubectl apply -k config/samples
```

### 4. Check status

```bash
kubectl get ggroups
kubectl get gprojects
kubectl get notifiers
```

## Configuration

Before deploying, you need to provide your own values for the following placeholders in the config files:

| Placeholder | File | Description |
|-------------|------|-------------|
| `YOUR-GCP-PROJECT-ID` | `config/eso/cluster-secret-store.yaml` | Your Google Cloud project ID |
| `YOUR-GITLAB-GROUP-ID` | `config/samples/go-lyon-corp-group.yaml` | The numeric ID of your GitLab group |
| `PATH-TO-YOUR-SERVICE-ACCOUNT-CREDENTIALS` | `demo.Makefile` | Path to your GCP service account JSON key file |

The operator also expects the following Kubernetes Secrets (managed via External Secrets Operator):

- **`gitlab-token`**: GitLab API token (key: `token`): `config/samples/ext-secrets/gitlab-token.yaml`
- **`discord-webhook`**: Discord webhook URL (key: `webhookUrl`): `config/samples/ext-secrets/discord-webhook.yaml`

## Clean up

```bash
# Remove CRs and operator from the cluster
make undeploy

# Delete the Kind cluster
make clean
```

