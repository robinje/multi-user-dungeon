"""
Multi User Dungeon Deployment Script
"""

import traceback

import boto3
import yaml
from botocore.exceptions import ClientError

# Constants for stack names
COGNITO_STACK_NAME = "MUD-Cognito-Stack"
DYNAMO_STACK_NAME = "MUD-DynamoDB-Stack"
CODEBUILD_STACK_NAME = "MUD-CodeBuild-Stack"
CLOUDWATCH_STACK_NAME = "MUD-CloudWatch-Stack"

# Paths to the CloudFormation templates
COGNITO_TEMPLATE_PATH = "../cloudformation/cognito.yml"
DYNAMO_TEMPLATE_PATH = "../cloudformation/dynamo.yml"
CODEBUILD_TEMPLATE_PATH = "../cloudformation/codebuild.yml"
CLOUDWATCH_TEMPLATE_PATH = "../cloudformation/cloudwatch.yml"

# Configuration file path
CONFIG_PATH = "../ssh_server/config.yml"


def validate_s3_bucket(bucket_name, region="us-east-1"):
    s3_client = boto3.client("s3", region_name=region)
    try:
        s3_client.head_bucket(Bucket=bucket_name)
        print(f"S3 bucket '{bucket_name}' exists and is accessible")
        return True
    except ClientError as e:
        print(f"Error accessing S3 bucket '{bucket_name}': {e}")
        return False


def load_template(template_path):
    with open(template_path, "r", encoding="utf-8") as file:
        return file.read()


def deploy_stack(client, stack_name, template_body, parameters):
    cf_parameters = [{"ParameterKey": k, "ParameterValue": v} for k, v in parameters.items()]
    try:
        if stack_exists(client, stack_name):
            print(f"Updating existing stack: {stack_name}")
            client.update_stack(
                StackName=stack_name,
                TemplateBody=template_body,
                Parameters=cf_parameters,
                Capabilities=["CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"],
            )
        else:
            print(f"Creating new stack: {stack_name}")
            client.create_stack(
                StackName=stack_name,
                TemplateBody=template_body,
                Parameters=cf_parameters,
                Capabilities=["CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"],
            )
        wait_for_stack_completion(client, stack_name)
        return True
    except ClientError as err:
        print(f"Error in stack operation for {stack_name}: {err}")
        if not stack_exists(client, stack_name):
            print(f"Stack {stack_name} was not created due to an error.")
        else:
            print(f"Attempting to delete stack {stack_name} due to error...")
            try:
                client.delete_stack(StackName=stack_name)
                print(f"Stack {stack_name} deletion initiated.")
            except ClientError as delete_err:
                print(f"Error deleting stack {stack_name}: {delete_err}")
        return False


def stack_exists(client, stack_name):
    try:
        client.describe_stacks(StackName=stack_name)
        return True
    except client.exceptions.ClientError:
        return False


def wait_for_stack_completion(client, stack_name):
    print(f"Waiting for stack {stack_name} to complete...")
    waiter = client.get_waiter("stack_create_complete")
    waiter.wait(StackName=stack_name)
    print("Stack operation completed.")


def get_stack_outputs(client, stack_name):
    stack = client.describe_stacks(StackName=stack_name)
    outputs = stack["Stacks"][0]["Outputs"]
    return {output["OutputKey"]: output["OutputValue"] for output in outputs}


def update_configuration_file(config_updates):
    try:
        with open(CONFIG_PATH, "r", encoding="utf-8") as file:
            config = yaml.safe_load(file) or {}

        # Ensure top-level keys exist
        for key in ["Server", "Aws", "Cognito", "Game", "Logging"]:
            if key not in config or config[key] is None:
                config[key] = {}

        # Update Server configuration
        config["Server"]["Port"] = 9050

        # Update Aws configuration
        config["Aws"]["Region"] = "us-east-1"

        # Update Game configuration
        config["Game"].update(
            {
                "Balance": 0.25,
                "AutoSave": 5,
                "StartingHealth": 10,
                "StartingEssence": 3,
            }
        )

        # Update Logging configuration
        config["Logging"].update(
            {
                "ApplicationName": "mud",
                "LogLevel": 20,
                "LogGroup": config_updates.get("CloudWatch", {}).get("LogGroupName", "/mud/game-logs"),
                "LogStream": "application",
                "MetricNamespace": config_updates.get("CloudWatch", {}).get("MetricNamespace", "MUD/Application"),
            }
        )

        # Update Cognito configuration
        cognito_updates = config_updates.get("Cognito", {})
        config["Cognito"].update(
            {
                "UserPoolId": cognito_updates.get("UserPoolId", ""),
                "UserPoolClientSecret": cognito_updates.get("UserPoolClientSecret", ""),
                "UserPoolClientId": cognito_updates.get("UserPoolClientId", ""),
                "UserPoolDomain": cognito_updates.get("UserPoolDomain", ""),
                "UserPoolArn": cognito_updates.get("UserPoolArn", ""),
            }
        )

        with open(CONFIG_PATH, "w", encoding="utf-8") as file:
            yaml.dump(config, file, default_flow_style=False)

        print("Configuration file updated successfully.")
    except Exception as err:
        print(f"An error occurred while updating configuration file: {err}")
        print("Current config_updates:", config_updates)
        print("Current config:", config)


def gather_all_parameters():
    parameters = {}

    # Cognito parameters
    parameters["cognito"] = {
        "UserPoolName": input("Enter the Name of the user pool [default: mud-user-pool]: ") or "mud-user-pool",
        "AppClientName": input("Enter the Name of the app client [default: mud-app-client]: ") or "mud-app-client",
        "CallbackURL": input("Enter the URL of the callback for the app client [default: https://localhost:3000/callback]: ")
        or "https://localhost:3000/callback",
        "SignOutURL": input("Enter the URL of the sign-out page for the app client [default: https://localhost:3000/sign-out]: ")
        or "https://localhost:3000/sign-out",
        "ReplyEmailAddress": input("Enter the email address to send from: "),
    }

    # DynamoDB parameters (empty for now)
    parameters["dynamo"] = {}

    # CodeBuild parameters
    parameters["codebuild"] = {
        "GitHubSourceRepo": input(
            "Enter the GitHub repository URL for the source code [default: https://github.com/robinje/multi-user-dungeon]: "
        )
        or "https://github.com/robinje/multi-user-dungeon",
        "S3BucketName": input("Enter the name of the existing S3 bucket for build artifacts: "),
    }

    # CloudWatch parameters
    parameters["cloudwatch"] = {
        "LogGroupName": input("Enter the name for the CloudWatch Log Group [default: /mud/game-logs]: ") or "/mud/game-logs",
        "MetricNamespace": input("Enter the namespace for CloudWatch Metrics [default: MUD/Application]: ") or "MUD/Application",
    }

    return parameters


def main():
    cloudformation_client = boto3.client("cloudformation")

    try:
        # Gather all parameters upfront
        all_parameters = gather_all_parameters()

        # Validate S3 bucket
        s3_bucket_name = all_parameters["codebuild"]["S3BucketName"]
        if not validate_s3_bucket(s3_bucket_name):
            print("Invalid or inaccessible S3 bucket. Exiting...")
            return

        # Deploy Cognito stack
        cognito_template = load_template(COGNITO_TEMPLATE_PATH)
        if not deploy_stack(cloudformation_client, COGNITO_STACK_NAME, cognito_template, all_parameters["cognito"]):
            print("Deployment failed at Cognito stack. Exiting...")
            return

        cognito_outputs = get_stack_outputs(cloudformation_client, COGNITO_STACK_NAME)

        # Deploy DynamoDB stack
        dynamo_template = load_template(DYNAMO_TEMPLATE_PATH)
        if not deploy_stack(cloudformation_client, DYNAMO_STACK_NAME, dynamo_template, all_parameters["dynamo"]):
            print("Deployment failed at DynamoDB stack. Exiting...")
            return

        dynamo_outputs = get_stack_outputs(cloudformation_client, DYNAMO_STACK_NAME)

        # Update CodeBuild parameters with Cognito outputs
        all_parameters["codebuild"].update(
            {"UserPoolId": cognito_outputs.get("UserPoolId", ""), "ClientId": cognito_outputs.get("UserPoolClientId", "")}
        )

        # Deploy CodeBuild stack
        codebuild_template = load_template(CODEBUILD_TEMPLATE_PATH)
        if not deploy_stack(cloudformation_client, CODEBUILD_STACK_NAME, codebuild_template, all_parameters["codebuild"]):
            print("Deployment failed at CodeBuild stack. Exiting...")
            return

        codebuild_outputs = get_stack_outputs(cloudformation_client, CODEBUILD_STACK_NAME)

        # Deploy CloudWatch stack
        cloudwatch_template = load_template(CLOUDWATCH_TEMPLATE_PATH)
        if not deploy_stack(cloudformation_client, CLOUDWATCH_STACK_NAME, cloudwatch_template, all_parameters["cloudwatch"]):
            print("Deployment failed at CloudWatch stack. Exiting...")
            return

        cloudwatch_outputs = get_stack_outputs(cloudformation_client, CLOUDWATCH_STACK_NAME)

        # Update configuration file with outputs from all stacks
        config_updates = {
            "Cognito": cognito_outputs,
            "Dynamo": dynamo_outputs,
            "CodeBuild": codebuild_outputs,
            "CloudWatch": cloudwatch_outputs,
        }
        update_configuration_file(config_updates)

        print("Deployment completed successfully.")
    except Exception as e:
        print(f"An unexpected error occurred during deployment: {e}")
        traceback.print_exc()


if __name__ == "__main__":
    main()
