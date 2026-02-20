resource "aws_ecr_repository" "this" {
  for_each     = toset(var.repositories)
  name         = "${var.name_prefix}/${each.value}"
  force_delete = true

  image_scanning_configuration { scan_on_push = true }
}
