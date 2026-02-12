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
- Go 1.21+
- Terraform
- AWS CLI configured

### Run locally (controller + mock setup)

```bash
docker compose -f docker-compose.local.yml up --build
```

---

## 🚢 Deployment

### Terraform (AWS Infra)

```bash
cd infra/envs/dev
terraform init
terraform apply
```

### GitHub Actions

This repo includes workflows for:

- building Docker images
- pushing to AWS ECR
- deploying to ECS

Once configured, deployment becomes:

```text
git push origin main
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
