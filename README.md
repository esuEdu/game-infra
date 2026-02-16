# 🎮 GameStack Controller (Minecraft ↔ Hytale)

A **Discord-controlled game server orchestration system** that lets you run **Minecraft or Hytale on AWS**, with automated switching, backups, and command execution — without maintaining two servers at once.

This project is designed to be clean, automatable, and portfolio-worthy: **only one game server runs at a time**, and the controller handles everything.

---

## 🚀 Features

✅ Start/stop Minecraft or Hytale server from Discord  
✅ Switch games safely (Minecraft ⇄ Hytale)  
✅ Automated backups to S3 (versioned)  
✅ Restore world/config from latest backup  
✅ Send in-game commands (`/give`, `/tp`, etc.)  
✅ Clean modular design (Game Adapters)  
✅ Infrastructure-as-code with Terraform  
✅ CI/CD with GitHub Actions + AWS ECR/ECS

---

## 🏗 Architecture

```text
Discord → Bot → API Gateway (Nginx/Kong) → Controller API (Go) → AWS ECS + Docker Game Stack
```

### Core Components

| Component                  | Description                                                       |
| -------------------------- | ----------------------------------------------------------------- |
| **Discord Bot**            | Accepts commands like `!mc on`, `!backup`, `!switch hytale`       |
| **Router / Gateway**       | Public entrypoint (Nginx or Kong) with routing + rate limits      |
| **Controller API (Go)**    | Main brain: tracks state, runs workflows, triggers ECS/S3 actions |
| **Game Server Containers** | Minecraft and Hytale containers (only one runs at a time)         |
| **S3 Backup Storage**      | Stores world backups + version history                            |
| **Terraform**              | Provisions AWS infra (VPC, ECS cluster, ECR, IAM, S3, etc.)       |
| **GitHub Actions**         | Builds + deploys controller and game images automatically         |

---

## 🔥 Why This Exists

Running multiple game servers gets messy fast:

- duplicated infra
- wasted compute
- conflicting ports
- manual backups
- switching worlds becomes painful

This project solves that by enforcing a simple rule:

> **Only one game stack is active at a time.**

When switching games, the controller automatically:

1. stops the running game container
2. backs up the world to S3
3. restores the next game’s config/world
4. starts the new game server

---

## 🔁 Switching Workflow

### Minecraft → Hytale (example)

1. Discord command: `!switch hytale`
2. Controller:
    - Stops Minecraft task
    - Zips world + config
    - Uploads backup to S3
    - Pulls Hytale config (GitHub or S3)
    - Starts Hytale task
3. Discord replies: `Switched to Hytale ✅`

Same process works the other way.

---

## 🧠 Controller Design

The Controller API is built around a **GameAdapter interface** so Minecraft and Hytale act like plug-ins:

```go
type GameAdapter interface {
  Type() GameType
  Start(ctx context.Context) error
  Stop(ctx context.Context) error
  Backup(ctx context.Context) (backupKey string, err error)
  Restore(ctx context.Context, backupKey string) error
  SendCommand(ctx context.Context, command string) error
  Status(ctx context.Context) (map[string]any, error)
}
```

Minecraft uses RCON for commands.  
Hytale support is designed as a stub until official tooling is available.

---

## 🗂 Repo Layout

```text
game-infra/
  infra/
    envs/
      dev/
      prod/
    modules/
      network/
      ecs_ec2/
      iam/
      ecr/
      s3_backups/

  services/
    controller/
    router/
    games/
      minecraft/
      hytale/

  .github/workflows/
    build-and-push.yml
    deploy.yml

  docker-compose.local.yml
  README.md
```

---

## 🌍 AWS Infrastructure

This project is built around **AWS ECS on EC2** (simple + realistic).

### AWS Services Used

- **ECS Cluster (EC2 capacity provider)**
- **EC2 instance (persistent host runtime)**
- **EBS volume** for `/data`
- **S3 bucket** for backups (versioning enabled)
- **ECR** for container images
- **IAM roles** for controller permissions
- Optional:
    - Route53 + ACM for HTTPS
    - Application Load Balancer

---

## 📦 Backup Strategy

Worlds can get huge, so this project avoids storing them directly in Git.

### Recommended pattern:

✅ GitHub stores:

- configs
- plugin list
- metadata
- seed/version info

✅ S3 stores:

- full world backups (`zip`)

S3 versioning + lifecycle policies can automatically prune old backups.

---

## 🔐 Security Notes

This project is designed with security in mind:

- Game ports may be public, but **RCON is never exposed to the internet**
- Controller API should only accept signed requests
- Router can enforce:
    - rate limits
    - API keys
    - IP restrictions

Recommended auth options:

- HMAC signed requests (simple + strong)
- JWT (nice for scaling)
- GitHub Actions OIDC (bonus points)

---

## 📡 Controller API Endpoints

Example REST API:

| Method | Endpoint             | Description                 |
| ------ | -------------------- | --------------------------- |
| POST   | `/v1/server/start`   | Start a game server         |
| POST   | `/v1/server/stop`    | Stop the running server     |
| POST   | `/v1/server/switch`  | Switch active game          |
| POST   | `/v1/server/backup`  | Backup active game world    |
| POST   | `/v1/server/command` | Send command to game server |
| GET    | `/v1/status`         | Server + state status       |

Example request:

```json
{
	"game": "minecraft"
}
```

---

## 🎮 Discord Commands (Planned)

Examples:

```text
!mc on
!mc off
!hytale on
!switch minecraft
!switch hytale
!backup
!restore latest
!give Steve diamond 3
!status
```

---

## 🛠 Local Development

### Requirements

- Docker
- Go 1.24+
- Terraform
- AWS CLI configured

### Run locally (controller + mock setup)

```bash
docker compose -f docker-compose.local.yml up --build
```

---

## 🚢 Deployment

### Manual AWS deploy (current)

This repo now provisions ECS resources for:

- `app` task (`router` + `controller`)
- `minecraft` task

Use this first-run flow:

1. Add `infra/envs/dev/terraform.tfvars` (recommended):

```hcl
minecraft_rcon_password = "change-me-please"
allowed_api_cidrs       = ["0.0.0.0/0"] # tighten to your IP/CIDR
allowed_game_cidrs      = ["0.0.0.0/0"] # tighten if needed
```

2. Create Terraform backend resources once (S3 state + DynamoDB lock):

```bash
export AWS_REGION=us-east-1
export TF_STATE_BUCKET=your-unique-terraform-state-bucket
export TF_LOCK_TABLE=gamestack-terraform-locks

if [ "$AWS_REGION" = "us-east-1" ]; then
  aws s3api create-bucket --bucket "$TF_STATE_BUCKET" --region "$AWS_REGION"
else
  aws s3api create-bucket \
    --bucket "$TF_STATE_BUCKET" \
    --region "$AWS_REGION" \
    --create-bucket-configuration LocationConstraint="$AWS_REGION"
fi

aws s3api put-bucket-versioning \
  --bucket "$TF_STATE_BUCKET" \
  --versioning-configuration Status=Enabled

aws dynamodb create-table \
  --table-name "$TF_LOCK_TABLE" \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region "$AWS_REGION"
```

3. Create base infra and ECR repos first:

```bash
cd infra/envs/dev
terraform init \
  -reconfigure \
  -backend-config="bucket=${TF_STATE_BUCKET}" \
  -backend-config="key=game-infra/dev/terraform.tfstate" \
  -backend-config="region=${AWS_REGION}" \
  -backend-config="dynamodb_table=${TF_LOCK_TABLE}" \
  -backend-config="encrypt=true"
terraform apply \
  -target=module.network \
  -target=module.backups \
  -target=module.ecr \
  -target=module.iam \
  -target=module.ecs
```

4. Build and push images:

```bash
export AWS_REGION=us-east-1
export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
export ECR_PREFIX=gamestack-dev

aws ecr get-login-password --region "$AWS_REGION" | \
  docker login --username AWS --password-stdin "$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com"

docker build -t "$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/$ECR_PREFIX/controller:latest" services/controller
docker build -t "$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/$ECR_PREFIX/router:latest" services/router
docker build -t "$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/$ECR_PREFIX/minecraft:latest" services/games/minecraft

docker push "$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/$ECR_PREFIX/controller:latest"
docker push "$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/$ECR_PREFIX/router:latest"
docker push "$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/$ECR_PREFIX/minecraft:latest"
```

5. Apply full stack (ECS task definitions + services):

```bash
cd infra/envs/dev
terraform apply
```

6. Check service status:

```bash
aws ecs list-services --cluster gamestack-dev-cluster --region us-east-1
aws ecs describe-services \
  --cluster gamestack-dev-cluster \
  --services gamestack-dev-app gamestack-dev-minecraft \
  --region us-east-1
```

7. Get the EC2 public IP and test the API:

```bash
aws ec2 describe-instances \
  --filters "Name=tag:aws:autoscaling:groupName,Values=gamestack-dev-asg" \
  --query "Reservations[].Instances[?State.Name=='running'].PublicIpAddress" \
  --output text \
  --region us-east-1
```

Then call:

```bash
curl http://<EC2_PUBLIC_IP>/healthz
curl http://<EC2_PUBLIC_IP>/v1/status
```

### GitHub Actions (next)

This repo includes workflows for:

- `.github/workflows/ci-test.yml`
  Runs on push for every branch + pull requests.
  Executes Go tests and Terraform fmt/validate.
- `.github/workflows/infra-manual.yml`
  Manual workflow (`Run workflow`) with inputs:
  - `environment`: `dev` or `prod`
  - `operation`: `deploy` or `destroy`
  - `aws_region`: defaults to `us-east-1`
  Uses OIDC with `aws-actions/configure-aws-credentials`.

Required repository/environment secret:

```text
AWS_OIDC_ROLE_ARN
TF_STATE_BUCKET
TF_LOCK_TABLE
```

`infra-manual.yml` uses S3 remote state and DynamoDB locking via:

- key: `game-infra/<environment>/terraform.tfstate`
- lock table: `${TF_LOCK_TABLE}`

If OIDC works only on `main`, your AWS role trust policy is probably restricted to one branch.
Allow your repo refs or environments in `token.actions.githubusercontent.com:sub`, for example:

```json
{
  "StringEquals": {
    "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
  },
  "StringLike": {
    "token.actions.githubusercontent.com:sub": [
      "repo:YOUR_ORG/YOUR_REPO:ref:refs/heads/*",
      "repo:YOUR_ORG/YOUR_REPO:environment:dev",
      "repo:YOUR_ORG/YOUR_REPO:environment:prod"
    ]
  }
}
```

---

## 📌 MVP Roadmap

### Phase 1 (Working Minecraft stack)

- [x] Terraform ECS + EC2
- [x] Minecraft Docker image
- [ ] Controller API start/stop via ECS SDK
- [ ] Backup zip → S3 upload
- [ ] Discord bot integration

### Phase 2 (Switching support)

- [ ] Switch workflow
- [ ] Restore latest backup
- [ ] State tracking (local JSON → DynamoDB)

### Phase 3 (Hytale support)

- [ ] Hytale container template
- [ ] Hytale adapter implementation (when official server tooling exists)

---

## 🧠 Future Ideas

- Web dashboard UI
- Scheduled nightly backups
- Auto-shutdown when nobody is online
- Multi-server support (multiple Minecraft worlds)
- Metrics + logs via CloudWatch
- Auto scaling (if server gets bigger)

---

## 📜 License

MIT License — do whatever you want with it.

---

## 👑 Project Goal

This project is built as a real-world DevOps + backend portfolio piece:

- clean architecture
- automation-first
- cloud deployment
- security-aware
- expandable design
