module "network" {
  source              = "../../modules/network"
  name                = var.name
  environment         = var.environment
  cidr_block          = "10.30.0.0/16"
  az_count            = 1
  public_subnet_cidrs = ["10.30.10.0/24", "10.30.11.0/24"]
}

module "backups" {
  source        = "../../modules/s3_backups"
  name          = "${var.name}-backups"
  force_destroy = true
}

module "ecr" {
  source       = "../../modules/ecr"
  name_prefix  = var.name
  repositories = ["controller", "router", "minecraft", "hytale"]
}

module "iam" {
  source            = "../../modules/iam"
  name              = var.name
  backup_bucket_arn = module.backups.bucket_arn
}

module "ecs" {
  source             = "../../modules/ecs_ec2"
  name               = var.name
  vpc_id             = module.network.vpc_id
  public_subnet_ids  = module.network.public_subnet_ids
  allowed_api_cidrs  = var.allowed_api_cidrs
  allowed_game_cidrs = var.allowed_game_cidrs
  router_port        = var.router_port
  minecraft_port     = var.minecraft_host_port
}

module "ecs_services" {
  source = "../../modules/ecs_services"

  name               = var.name
  aws_region         = var.aws_region
  cluster_name       = module.ecs.cluster_name
  execution_role_arn = module.iam.execution_role_arn
  task_role_arn      = module.iam.task_role_arn
  ecr_repo_urls      = module.ecr.repo_urls
  image_tag          = var.image_tag
  backup_bucket_name = module.backups.bucket_name
  backup_prefix      = var.backup_prefix

  router_host_port              = var.router_port
  minecraft_host_port           = var.minecraft_host_port
  minecraft_rcon_port           = var.minecraft_rcon_port
  minecraft_loader              = var.minecraft_loader
  minecraft_version             = var.minecraft_version
  minecraft_server_url          = var.minecraft_server_url
  minecraft_git_bootstrap_repo  = var.minecraft_git_bootstrap_repo
  minecraft_git_bootstrap_ref   = var.minecraft_git_bootstrap_ref
  minecraft_git_bootstrap_path  = var.minecraft_git_bootstrap_path
  minecraft_git_bootstrap_token = var.minecraft_git_bootstrap_token
  controller_git_user_name      = var.controller_git_user_name
  controller_git_user_email     = var.controller_git_user_email
  controller_git_auth_token     = var.controller_git_auth_token
  minecraft_rcon_password       = var.minecraft_rcon_password
}
