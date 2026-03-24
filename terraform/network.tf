# trivy:ignore:AVD-AWS-0178 -- VPC Flow Logs are out of scope for initial deployment
resource "aws_vpc" "main" {
  # checkov:skip=CKV2_AWS_11:VPC Flow Logs are out of scope for initial deployment
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = merge(local.common_tags, {
    Name = "bunshin"
  })
}

# Public subnets
resource "aws_subnet" "public" {
  count = length(local.azs)

  vpc_id            = aws_vpc.main.id
  cidr_block        = local.public_cidrs[count.index]
  availability_zone = local.azs[count.index]

  tags = merge(local.common_tags, {
    Name = "bunshin-public-${local.azs[count.index]}"
  })
}

# Private subnets
resource "aws_subnet" "private" {
  count = length(local.azs)

  vpc_id            = aws_vpc.main.id
  cidr_block        = local.private_cidrs[count.index]
  availability_zone = local.azs[count.index]

  tags = merge(local.common_tags, {
    Name = "bunshin-private-${local.azs[count.index]}"
  })
}

# Internet Gateway
resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = merge(local.common_tags, {
    Name = "bunshin"
  })
}

# Regional NAT Gateway
resource "aws_nat_gateway" "main" {
  vpc_id            = aws_vpc.main.id
  availability_mode = "regional"

  tags = merge(local.common_tags, {
    Name = "bunshin-nat"
  })

  depends_on = [aws_internet_gateway.main]
}

# Public route table
resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = merge(local.common_tags, {
    Name = "bunshin-public"
  })
}

resource "aws_route_table_association" "public" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  count = length(local.azs)

  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

# Private route table
resource "aws_route_table" "private" {
  vpc_id = aws_vpc.main.id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.main.id
  }

  tags = merge(local.common_tags, {
    Name = "bunshin-private"
  })
}

resource "aws_route_table_association" "private" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  count = length(local.azs)

  subnet_id      = aws_subnet.private[count.index].id
  route_table_id = aws_route_table.private.id
}

# VPC Gateway Endpoint for DynamoDB
resource "aws_vpc_endpoint" "dynamodb" {
  vpc_id       = aws_vpc.main.id
  service_name = "com.amazonaws.ap-northeast-1.dynamodb"

  vpc_endpoint_type = "Gateway"
  route_table_ids   = [aws_route_table.private.id]

  tags = merge(local.common_tags, {
    Name = "bunshin-dynamodb"
  })
}

# VPC Interface Endpoints for ECR and CloudWatch Logs
resource "aws_security_group" "ecr_endpoint" {
  name_prefix = "bunshin-ecr-ep-"
  description = "Security group for ECR and CloudWatch Logs VPC endpoints"
  vpc_id      = aws_vpc.main.id

  tags = merge(local.common_tags, {
    Name    = "bunshin-ecr-endpoint"
    Service = "ecr"
  })

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group_rule" "ecr_endpoint_ingress" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  for_each = {
    nginx  = aws_security_group.nginx.id
    broker = aws_security_group.broker.id
    runner = aws_security_group.runner.id
  }

  type                     = "ingress"
  from_port                = 443
  to_port                  = 443
  protocol                 = "tcp"
  source_security_group_id = each.value
  security_group_id        = aws_security_group.ecr_endpoint.id
  description              = "HTTPS from ${each.key}"
}

resource "aws_vpc_endpoint" "ecr_api" {
  vpc_id            = aws_vpc.main.id
  service_name      = "com.amazonaws.ap-northeast-1.ecr.api"
  vpc_endpoint_type = "Interface"

  subnet_ids         = aws_subnet.private[*].id
  security_group_ids = [aws_security_group.ecr_endpoint.id]

  private_dns_enabled = true

  tags = merge(local.common_tags, {
    Name = "bunshin-ecr-api"
  })
}

resource "aws_vpc_endpoint" "ecr_dkr" {
  vpc_id            = aws_vpc.main.id
  service_name      = "com.amazonaws.ap-northeast-1.ecr.dkr"
  vpc_endpoint_type = "Interface"

  subnet_ids         = aws_subnet.private[*].id
  security_group_ids = [aws_security_group.ecr_endpoint.id]

  private_dns_enabled = true

  tags = merge(local.common_tags, {
    Name = "bunshin-ecr-dkr"
  })
}

resource "aws_vpc_endpoint" "logs" {
  vpc_id            = aws_vpc.main.id
  service_name      = "com.amazonaws.ap-northeast-1.logs"
  vpc_endpoint_type = "Interface"

  subnet_ids         = aws_subnet.private[*].id
  security_group_ids = [aws_security_group.ecr_endpoint.id]

  private_dns_enabled = true

  tags = merge(local.common_tags, {
    Name = "bunshin-logs"
  })
}

# S3 Gateway Endpoint for ECR image layer storage
resource "aws_vpc_endpoint" "s3" {
  vpc_id       = aws_vpc.main.id
  service_name = "com.amazonaws.ap-northeast-1.s3"

  vpc_endpoint_type = "Gateway"
  route_table_ids   = [aws_route_table.private.id]

  tags = merge(local.common_tags, {
    Name = "bunshin-s3"
  })
}

# VPC Interface Endpoint for Bedrock Runtime
resource "aws_security_group" "bedrock_endpoint" {
  name_prefix = "bunshin-bedrock-ep-"
  description = "Security group for Bedrock Runtime VPC endpoint"
  vpc_id      = aws_vpc.main.id

  tags = merge(local.common_tags, {
    Name    = "bunshin-bedrock-endpoint"
    Service = "bedrock"
  })

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group_rule" "bedrock_endpoint_ingress_runner" {
  # checkov:skip=CKV_BUNSHIN_1:Resource does not support tags
  type                     = "ingress"
  from_port                = 443
  to_port                  = 443
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.runner.id
  security_group_id        = aws_security_group.bedrock_endpoint.id
  description              = "HTTPS from runner"
}

resource "aws_vpc_endpoint" "bedrock_runtime" {
  vpc_id            = aws_vpc.main.id
  service_name      = "com.amazonaws.ap-northeast-1.bedrock-runtime"
  vpc_endpoint_type = "Interface"

  subnet_ids         = aws_subnet.private[*].id
  security_group_ids = [aws_security_group.bedrock_endpoint.id]

  private_dns_enabled = true

  tags = merge(local.common_tags, {
    Name = "bunshin-bedrock-runtime"
  })
}

# Restrict the default security group to deny all traffic
resource "aws_default_security_group" "default" {
  vpc_id = aws_vpc.main.id

  tags = merge(local.common_tags, {
    Name = "bunshin-default"
  })
}
