output "app_service_name" {
  value = aws_ecs_service.app.name
}

output "minecraft_service_name" {
  value = aws_ecs_service.minecraft.name
}

output "app_task_definition_arn" {
  value = aws_ecs_task_definition.app.arn
}

output "minecraft_task_definition_arn" {
  value = aws_ecs_task_definition.minecraft.arn
}
