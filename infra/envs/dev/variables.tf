variable "aws_region" { 
    type = string 
    default = "us-east-1"
}
variable "name"       { 
    type = string 
    default = "gamestack-dev"
}

variable "environment" {
    type = string
    default = "desenv"
}

variable "allowed_game_cidrs" { 
    type = list(string) 
    default = ["0.0.0.0/0"] 
}
