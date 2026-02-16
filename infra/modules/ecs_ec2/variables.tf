variable "name" { type = string }
variable "vpc_id" { type = string }
variable "public_subnet_ids" { type = list(string) }

variable "instance_type" {
  type    = string
  default = "t3.medium"
}
variable "key_name" {
  type    = string
  default = null
}

variable "allowed_game_cidrs" {
  type        = list(string)
  description = "Who can connect to Minecraft/Hytale ports"
  default     = ["0.0.0.0/0"]
}

variable "allowed_api_cidrs" {
  type        = list(string)
  description = "Who can connect to the router/API port"
  default     = ["0.0.0.0/0"]
}

variable "router_port" {
  type    = number
  default = 80
}

variable "minecraft_port" {
  type    = number
  default = 25565
}
variable "hytale_port" {
  type    = number
  default = 25566
}
