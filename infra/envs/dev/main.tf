module "network" {
    source = "../../modules/network"
    name = var.name
    environment = var.environment
    cidr_block = "10.30.0.0/16"
    az_count = 1
    public_subnet_cidrs = ["10.30.10.0/24", "10.30.11.0/24" ]
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
  source            = "../../modules/ecs_ec2"
  name              = var.name
  vpc_id            = module.network.vpc_id
  public_subnet_ids  = module.network.public_subnet_ids
  allowed_game_cidrs = var.allowed_game_cidrs
}