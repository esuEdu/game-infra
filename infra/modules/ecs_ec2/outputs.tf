output "cluster_name" { value = aws_ecs_cluster.this.name }
output "host_sg_id"   { value = aws_security_group.host.id }
