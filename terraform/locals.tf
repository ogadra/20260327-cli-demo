locals {
  azs = ["ap-northeast-1a", "ap-northeast-1c", "ap-northeast-1d"]

  public_cidrs  = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  private_cidrs = ["10.0.11.0/24", "10.0.12.0/24", "10.0.13.0/24"]

  ecs_services = {
    nginx  = { port = 8080 }
    broker = { port = 8080 }
    runner = { port = 3000 }
  }

  common_tags = {
    Project   = "Bunshin"
    ManagedBy = "terraform"
  }

  presenter_ws_handlers = {
    connect    = { route_key = "$connect" }
    disconnect = { route_key = "$disconnect" }
    message    = { route_key = "$default" }
  }

  presenter_cf_origin_id = {
    login     = "presenter-login-api"
    websocket = "presenter-websocket-api"
  }
}
