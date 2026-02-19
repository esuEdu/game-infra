variable "name" {
  type = string
}

variable "aws_region" {
  type = string
}

variable "cluster_name" {
  type = string
}

variable "execution_role_arn" {
  type = string
}

variable "task_role_arn" {
  type = string
}

variable "ecr_repo_urls" {
  type = map(string)
}

variable "backup_bucket_name" {
  type = string
}

variable "backup_prefix" {
  type    = string
  default = "backups"
}

variable "controller_desired_count" {
  type    = number
  default = 1
}

variable "minecraft_desired_count" {
  type    = number
  default = 0
}

variable "router_host_port" {
  type    = number
  default = 80
}

variable "controller_container_port" {
  type    = number
  default = 8080
}

variable "minecraft_host_port" {
  type    = number
  default = 25565
}

variable "minecraft_rcon_port" {
  type    = number
  default = 25575
}

variable "minecraft_data_host_path" {
  type    = string
  default = "/var/lib/gamestack/minecraft"
}

variable "controller_minecraft_data_dir" {
  type    = string
  default = "/srv/minecraft-data"
}

variable "controller_git_user_name" {
  type    = string
  default = "GameStack Bot"
}

variable "controller_git_user_email" {
  type    = string
  default = "gamestack-bot@example.com"
}

variable "controller_git_auth_token" {
  type      = string
  sensitive = true
  default   = ""
}

variable "minecraft_loader" {
  type    = string
  default = "fabric"
}

variable "minecraft_version" {
  type    = string
  default = "1.21.1"
}

variable "minecraft_java_xms" {
  type    = string
  default = "1G"
}

variable "minecraft_java_xmx" {
  type    = string
  default = "2G"
}

variable "minecraft_server_url" {
  type    = string
  default = ""
}

variable "minecraft_git_bootstrap_repo" {
  type    = string
  default = ""
}

variable "minecraft_git_bootstrap_ref" {
  type    = string
  default = "main"
}

variable "minecraft_git_bootstrap_path" {
  type    = string
  default = ""
}

variable "minecraft_git_bootstrap_token" {
  type      = string
  sensitive = true
  default   = ""
}

variable "minecraft_rcon_password" {
  type      = string
  sensitive = true
}

variable "log_retention_days" {
  type    = number
  default = 14
}
