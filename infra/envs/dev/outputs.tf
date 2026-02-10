output "ecr_repo_urls" { value = module.ecr.repo_urls }
output "backup_bucket" { value = module.backups.bucket_name }
output "ecs_cluster"   { value = module.ecs.cluster_name }
