"""Generate the Bunshin AWS architecture diagram including deploy flow."""

from diagrams import Cluster, Diagram, Edge
from diagrams.aws.compute import ECR, ECS, Lambda
from diagrams.aws.database import Dynamodb
from diagrams.aws.ml import SagemakerModel
from diagrams.aws.network import APIGateway, CloudFront, CloudMap, ELB, Endpoint
from diagrams.aws.security import SecretsManager
from diagrams.aws.storage import S3
from diagrams.onprem.ci import GithubActions
from diagrams.onprem.client import Users

GRAPH_ATTR = {
    "fontsize": "24",
    "bgcolor": "white",
    "pad": "0.5",
    "nodesep": "0.8",
    "ranksep": "1.2",
    "compound": "true",
}

with Diagram(
    "Bunshin - AWS Architecture",
    show=False,
    filename="bunshin_architecture",
    outformat="png",
    direction="TB",
    graph_attr=GRAPH_ATTR,
):
    users = Users("Clients")
    cf = CloudFront("CloudFront")

    users >> cf

    # --- Runtime layer ---
    with Cluster("VPC 10.0.0.0/16"):
        with Cluster("Public Subnets"):
            alb = ELB("ALB")

        with Cluster("Private Subnets"):
            with Cluster("ECS Cluster: bunshin"):
                nginx = ECS("NGINX\n6 tasks / ARM64")
                broker = ECS("Broker\n6 tasks / ARM64")
                runner = ECS("Runner\nN tasks / x86_64")

            vpce = Endpoint("VPC Endpoints\nDynamoDB / S3 / ECR\nLogs / Bedrock")
            cloud_map = CloudMap("Cloud Map\nbroker.internal")

    s3_front = S3("S3\nFront Assets")

    with Cluster("Presenter - Serverless"):
        apigw_ws = APIGateway("API GW\nWebSocket")
        apigw_http = APIGateway("API GW\nHTTP")
        lmb_ws = Lambda("ws-connect\nws-disconnect\nws-message")
        lmb_login = Lambda("login")

        apigw_ws >> lmb_ws
        apigw_http >> lmb_login

    # CloudFront routing
    cf >> Edge(label="/") >> s3_front
    cf >> Edge(label="/api/*") >> alb
    cf >> Edge(label="/ws") >> apigw_ws
    cf >> Edge(label="/login") >> apigw_http

    # ECS data flow
    alb >> nginx
    nginx >> Edge(label="auth_request") >> broker
    nginx >> Edge(label="proxy") >> runner
    runner >> Edge(label="register") >> broker

    # --- Data stores ---
    ddb_runners = Dynamodb("bunshin-runners")
    broker >> ddb_runners

    with Cluster("Presenter DB"):
        ddb_ws = Dynamodb("ws-connections\npoll-votes\nroom-state")
        ddb_sessions = Dynamodb("sessions")

    lmb_ws >> ddb_ws
    lmb_login >> ddb_sessions

    secrets = SecretsManager("Secrets Manager\nPassword Hash")
    lmb_login >> secrets

    # Bedrock
    bedrock = SagemakerModel("Bedrock Runtime\nClaude Sonnet 4")
    runner >> Edge(label="LLM validation") >> bedrock

    # --- CI/CD Deploy flow ---
    with Cluster("CI/CD - GitHub Actions"):
        gh = GithubActions("GitHub Actions\nOIDC")

    with Cluster("ECR"):
        ecr_nginx = ECR("nginx")
        ecr_broker = ECR("broker")
        ecr_runner = ECR("runner")

    s3_lambda = S3("S3\nLambda Deploy")

    gh >> Edge(style="dashed", color="gray") >> ecr_nginx
    gh >> Edge(style="dashed", color="gray") >> ecr_broker
    gh >> Edge(style="dashed", color="gray") >> ecr_runner
    gh >> Edge(label="S3 Sync", style="dashed", color="gray") >> s3_front
    gh >> Edge(style="dashed", color="gray") >> s3_lambda
    s3_lambda >> Edge(style="dashed", color="gray") >> lmb_ws
    s3_lambda >> Edge(style="dashed", color="gray") >> lmb_login
