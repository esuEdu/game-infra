locals {
  controller_env_base = [
    {
      name  = "HTTP_ADDR"
      value = ":${var.controller_container_port}"
    },
    {
      name  = "AWS_REGION"
      value = var.aws_region
    },
    {
      name  = "BACKUP_BUCKET"
      value = var.backup_bucket_name
    },
    {
      name  = "BACKUP_PREFIX"
      value = var.backup_prefix
    },
    {
      name  = "ECS_CLUSTER_NAME"
      value = var.cluster_name
    },
    {
      name  = "ECS_SERVICE_MINECRAFT"
      value = "${var.name}-minecraft"
    },
    {
      name  = "MC_DATA_DIR"
      value = var.controller_minecraft_data_dir
    },
    {
      name  = "GIT_USER_NAME"
      value = var.controller_git_user_name
    },
    {
      name  = "GIT_USER_EMAIL"
      value = var.controller_git_user_email
    }
  ]

  controller_env = concat(
    local.controller_env_base,
    var.controller_git_auth_token != "" ? [
      {
        name  = "GIT_AUTH_TOKEN"
        value = var.controller_git_auth_token
      }
    ] : []
  )

  minecraft_env_base = [
    {
      name  = "EULA"
      value = "true"
    },
    {
      name  = "LOADER"
      value = var.minecraft_loader
    },
    {
      name  = "MC_VERSION"
      value = var.minecraft_version
    },
    {
      name  = "JAVA_XMS"
      value = var.minecraft_java_xms
    },
    {
      name  = "JAVA_XMX"
      value = var.minecraft_java_xmx
    },
    {
      name  = "ENABLE_RCON"
      value = "true"
    },
    {
      name  = "RCON_PORT"
      value = tostring(var.minecraft_rcon_port)
    },
    {
      name  = "RCON_PASSWORD"
      value = var.minecraft_rcon_password
    }
  ]

  minecraft_env = concat(
    local.minecraft_env_base,
    var.minecraft_git_bootstrap_repo != "" ? [
      {
        name  = "GIT_BOOTSTRAP_REPO"
        value = var.minecraft_git_bootstrap_repo
      },
      {
        name  = "GIT_BOOTSTRAP_REF"
        value = var.minecraft_git_bootstrap_ref
      }
    ] : [],
    var.minecraft_git_bootstrap_path != "" ? [
      {
        name  = "GIT_BOOTSTRAP_PATH"
        value = var.minecraft_git_bootstrap_path
      }
    ] : [],
    var.minecraft_git_bootstrap_token != "" ? [
      {
        name  = "GIT_BOOTSTRAP_TOKEN"
        value = var.minecraft_git_bootstrap_token
      }
    ] : [],
    var.minecraft_server_url != "" ? [
      {
        name  = "SERVER_URL"
        value = var.minecraft_server_url
      }
    ] : []
  )
}

resource "aws_cloudwatch_log_group" "app" {
  name              = "/ecs/${var.name}/app"
  retention_in_days = var.log_retention_days
}

resource "aws_cloudwatch_log_group" "minecraft" {
  name              = "/ecs/${var.name}/minecraft"
  retention_in_days = var.log_retention_days
}

resource "aws_ecs_task_definition" "app" {
  family                   = "${var.name}-app"
  network_mode             = "bridge"
  requires_compatibilities = ["EC2"]
  execution_role_arn       = var.execution_role_arn
  task_role_arn            = var.task_role_arn

  volume {
    name      = "minecraft-data"
    host_path = var.minecraft_data_host_path
  }

  container_definitions = jsonencode([
    {
      name              = "controller"
      image             = "${var.ecr_repo_urls["controller"]}:latest"
      essential         = true
      memoryReservation = 256
      environment       = local.controller_env
      mountPoints = [
        {
          sourceVolume  = "minecraft-data"
          containerPath = var.controller_minecraft_data_dir
          readOnly      = false
        }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.app.name
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "controller"
        }
      }
    },
    {
      name              = "router"
      image             = "${var.ecr_repo_urls["router"]}:latest"
      essential         = true
      memoryReservation = 128
      links             = ["controller"]
      dependsOn = [
        {
          containerName = "controller"
          condition     = "START"
        }
      ]
      portMappings = [
        {
          containerPort = 80
          hostPort      = var.router_host_port
          protocol      = "tcp"
        }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.app.name
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "router"
        }
      }
    }
  ])
}

resource "aws_ecs_task_definition" "minecraft" {
  family                   = "${var.name}-minecraft"
  network_mode             = "bridge"
  requires_compatibilities = ["EC2"]
  execution_role_arn       = var.execution_role_arn
  task_role_arn            = var.task_role_arn

  volume {
    name      = "minecraft-data"
    host_path = var.minecraft_data_host_path
  }

  container_definitions = jsonencode([
    {
      name              = "minecraft"
      image             = "${var.ecr_repo_urls["minecraft"]}:latest"
      essential         = true
      memoryReservation = 2048
      environment       = local.minecraft_env
      mountPoints = [
        {
          sourceVolume  = "minecraft-data"
          containerPath = "/data"
          readOnly      = false
        }
      ]
      portMappings = [
        {
          containerPort = var.minecraft_host_port
          hostPort      = var.minecraft_host_port
          protocol      = "tcp"
        },
        {
          containerPort = var.minecraft_rcon_port
          hostPort      = var.minecraft_rcon_port
          protocol      = "tcp"
        }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = aws_cloudwatch_log_group.minecraft.name
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "minecraft"
        }
      }
    }
  ])
}

resource "aws_ecs_service" "app" {
  name                               = "${var.name}-app"
  cluster                            = var.cluster_name
  task_definition                    = aws_ecs_task_definition.app.arn
  desired_count                      = var.controller_desired_count
  launch_type                        = "EC2"
  deployment_maximum_percent         = 100
  deployment_minimum_healthy_percent = 0
  enable_ecs_managed_tags            = true
}

resource "aws_ecs_service" "minecraft" {
  name                               = "${var.name}-minecraft"
  cluster                            = var.cluster_name
  task_definition                    = aws_ecs_task_definition.minecraft.arn
  desired_count                      = var.minecraft_desired_count
  launch_type                        = "EC2"
  deployment_maximum_percent         = 100
  deployment_minimum_healthy_percent = 0
  enable_ecs_managed_tags            = true
}
