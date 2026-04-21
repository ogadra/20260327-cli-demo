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

  # Destination regions for APAC cross-region inference profile from ap-northeast-1
  apac_cris_destination_regions = [
    "ap-northeast-1",
    "ap-northeast-2",
    "ap-northeast-3",
    "ap-south-1",
    "ap-south-2",
    "ap-southeast-1",
    "ap-southeast-2",
    "ap-southeast-4",
  ]
}
