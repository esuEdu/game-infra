variable "aws_region" {
  type    = string
  default = "us-east-1"
}
variable "name" {
  type    = string
  default = "gamestack-dev"
}

variable "environment" {
  type    = string
  default = "desenv"
}

variable "allowed_game_cidrs" {
  type    = list(string)
  default = ["0.0.0.0/0"]
}

variable "allowed_api_cidrs" {
  type    = list(string)
  default = ["0.0.0.0/0"]
}

variable "router_port" {
  type    = number
  default = 80
}

variable "minecraft_host_port" {
  type    = number
  default = 25565
}

variable "minecraft_rcon_port" {
  type    = number
  default = 25575
}

variable "minecraft_loader" {
  type    = string
  default = "fabric"
}

variable "minecraft_version" {
  type    = string
  default = "1.21.1"
}

variable "minecraft_server_url" {
  type    = string
  default = ""
}

variable "minecraft_rcon_password" {
  type      = string
  sensitive = true
  default   = "devpass"
}
