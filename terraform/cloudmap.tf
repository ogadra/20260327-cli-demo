resource "aws_service_discovery_private_dns_namespace" "internal" {
  name = "internal"
  vpc  = aws_vpc.main.id

  tags = merge(local.common_tags, {
    Service = "cloudmap"
  })
}

resource "aws_service_discovery_service" "broker" {
  name = "broker"

  dns_config {
    namespace_id = aws_service_discovery_private_dns_namespace.internal.id

    dns_records {
      ttl  = 10
      type = "A"
    }

    routing_policy = "MULTIVALUE"
  }

  health_check_custom_config {
    failure_threshold = 1
  }

  tags = merge(local.common_tags, {
    Service = "broker"
  })
}
