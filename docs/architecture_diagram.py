"""Generate the Bunshin AWS architecture diagram including deploy flow."""

from pathlib import Path

from diagrams import Cluster, Diagram, Edge
from diagrams.aws.compute import ECR, ECS, Lambda
from diagrams.aws.database import Dynamodb
from diagrams.aws.ml import SagemakerModel
from diagrams.aws.network import APIGateway, CloudFront, ELB, Endpoint
from diagrams.aws.security import SecretsManager
from diagrams.aws.storage import S3
from diagrams.onprem.ci import GithubActions
from diagrams.onprem.client import Users

CLUSTER_FONT = {"fontsize": "20", "fontname": "Sans-Serif Bold"}

GRAPH_ATTR = {
    "fontsize": "32",
    "bgcolor": "white",
    "pad": "0.8",
    "nodesep": "1.0",
    "ranksep": "1.5",
    "compound": "true",
    **CLUSTER_FONT,
}

NODE_ATTR = {
    "fontsize": "16",
    "fontname": "Sans-Serif Bold",
    "labelloc": "b",
    "imagepos": "tc",
}

EDGE_ATTR = {
    "fontsize": "16",
    "fontname": "Sans-Serif Bold",
}

OUTPUT_FILE = str(Path(__file__).with_name("bunshin_architecture"))


def main() -> None:
    """Generate the architecture diagram as a PNG file."""
    with Diagram(
        "Bunshin - AWS Architecture",
        show=False,
        filename=OUTPUT_FILE,
        outformat="png",
        direction="TB",
        graph_attr=GRAPH_ATTR,
        node_attr=NODE_ATTR,
        edge_attr=EDGE_ATTR,
    ):
        users = Users("Clients")
        cf = CloudFront("CloudFront")

        users >> cf

        # --- Runtime layer ---
        with Cluster("VPC 10.0.0.0/16", graph_attr={**CLUSTER_FONT, "margin": "8"}):
            with Cluster("Public Subnets (1a / 1c / 1d)",
                         graph_attr={**CLUSTER_FONT, "margin": "24"}):
                alb = ELB("ALB")

            with Cluster("Private Subnets (1a / 1c / 1d)",
                         graph_attr={**CLUSTER_FONT, "margin": "8"}):
                with Cluster("ECS Cluster: bunshin",
                             graph_attr={**CLUSTER_FONT, "margin": "24"}):
                    nginx = ECS("NGINX\n3 tasks / ARM64")
                    broker = ECS("Broker\n6 tasks\nARM64")
                    runner = ECS("Runner\n250 tasks / x86_64")

                vpce_gw = Endpoint("VPCE Gateway\nDynamoDB / S3")
                vpce_if = Endpoint("VPCE Interface\nECR / Logs\nBedrock")
                vpce_gw >> Edge(style="invis") >> vpce_if

        s3_front = S3("S3\nFront Assets")

        with Cluster("Presenter - Serverless", graph_attr={**CLUSTER_FONT, "margin": "24"}):
            apigw_ws = APIGateway("API GW\nWebSocket")
            apigw_http = APIGateway("API GW\nHTTP")
            lmb_ws = Lambda("ws-connect\nws-disconnect\nws-message")
            lmb_login = Lambda("login")

            apigw_ws >> lmb_ws
            apigw_http >> lmb_login

        # CloudFront routing
        cf >> Edge(xlabel="PATH: /") >> s3_front
        cf >> Edge(xlabel="PATH: /api/*") >> alb
        cf >> Edge(xlabel="PATH: /ws") >> apigw_ws
        cf >> Edge(xlabel="PATH: /login") >> apigw_http

        # ECS data flow
        alb >> nginx
        nginx >> Edge(label="proxy") >> runner
        nginx >> Edge(label="auth_request") >> broker
        broker >> Edge(label="register") << runner

        # --- Data stores ---
        ddb_runners = Dynamodb("bunshin-runners")
        broker >> ddb_runners

        with Cluster("Presenter DB", graph_attr={**CLUSTER_FONT, "margin": "24"}):
            ddb_ws = Dynamodb("ws-connections\npoll-votes\nroom-state")
            ddb_sessions = Dynamodb("sessions")

        lmb_ws >> ddb_ws
        lmb_ws >> Edge(xlabel="session check") >> ddb_sessions
        lmb_login >> ddb_sessions

        secrets = SecretsManager("Secrets Manager\nPassword Hash")
        lmb_login >> secrets
        ddb_sessions - Edge(style="invis") - secrets

        # Bedrock
        bedrock = SagemakerModel("Bedrock Runtime\nClaude Sonnet 4")
        runner >> Edge(xlabel="LLM validation") >> bedrock

        # --- CI/CD Deploy flow ---
        with Cluster("CI/CD - GitHub Actions", graph_attr={**CLUSTER_FONT, "margin": "24"}):
            gh = GithubActions("GitHub Actions\nOIDC")

        with Cluster("ECR", graph_attr={**CLUSTER_FONT, "margin": "24"}):
            ecr_runner = ECR("runner")
            ecr_broker = ECR("broker")
            ecr_nginx = ECR("nginx")

        s3_lambda = S3("S3\nLambda Deploy")

        gh >> Edge(style="dashed", color="gray") >> ecr_nginx
        gh >> Edge(style="dashed", color="gray") >> ecr_broker
        gh >> Edge(style="dashed", color="gray") >> ecr_runner
        gh >> Edge(xlabel="S3 Sync", style="dashed", color="gray") >> s3_front
        gh >> Edge(style="dashed", color="gray") >> s3_lambda
        s3_lambda >> Edge(style="dashed", color="gray") >> lmb_ws
        s3_lambda >> Edge(style="dashed", color="gray") >> lmb_login

        # ECR -> ECS image pull
        ecr_nginx >> Edge(xlabel="image pull", style="dotted") >> nginx
        ecr_broker >> Edge(style="dotted") >> broker
        ecr_runner >> Edge(style="dotted") >> runner


if __name__ == "__main__":
    main()
