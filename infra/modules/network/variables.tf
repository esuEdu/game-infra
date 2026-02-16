variable "name" {
  type        = string
  description = "Name for the VPC"
}

variable "environment" {
  type        = string
  description = "Deploy environment"
}

variable "cidr_block" {
  type        = string
  description = "CIDR block for the VPC"
}

variable "az_count" {
  type        = number
  description = "Avalible zone counter"
}

variable "public_subnet_cidrs" {
  type        = list(string)
  description = "Must match az_count length"
}