data "aws_iam_policy_document" "ecs_task_assume" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

# Execution role (ECR pull, logs)
resource "aws_iam_role" "execution" {
  name               = "${var.name}-ecs-exec"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume.json
}

resource "aws_iam_role_policy_attachment" "execution" {
  role       = aws_iam_role.execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# Task role (controller permissions)
resource "aws_iam_role" "task" {
  name               = "${var.name}-task"
  assume_role_policy = data.aws_iam_policy_document.ecs_task_assume.json
}

data "aws_iam_policy_document" "controller" {
  statement {
    sid = "ECSControl"
    actions = [
      "ecs:RunTask", "ecs:StopTask", "ecs:DescribeTasks",
      "ecs:UpdateService", "ecs:DescribeServices",
      "ecs:ListTasks", "ecs:DescribeTaskDefinition"
    ]
    resources = ["*"]
  }

  statement {
    sid     = "S3Backups"
    actions = ["s3:PutObject", "s3:GetObject", "s3:ListBucket"]
    resources = [
      var.backup_bucket_arn,
      "${var.backup_bucket_arn}/*"
    ]
  }

  statement {
    sid       = "PassRoles"
    actions   = ["iam:PassRole"]
    resources = ["*"]
  }
}

resource "aws_iam_policy" "controller" {
  name   = "${var.name}-controller"
  policy = data.aws_iam_policy_document.controller.json
}

resource "aws_iam_role_policy_attachment" "controller" {
  role       = aws_iam_role.task.name
  policy_arn = aws_iam_policy.controller.arn
}
