import yaml
import boto3
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

def prompt_for_parameters(template_name):
    if template_name == "cognito":
        return {
            "UserPoolName": input("Enter the Name of the user pool [default: mud-user-pool]: ") or "mud-user-pool",
            "AppClientName": input("Enter the Name of the app client [default: mud-app-client]: ") or "mud-app-client",
            "CallbackURL": input("Enter the URL of the callback for the app client [default: https://localhost:3000/callback]: ")
            or "https://localhost:3000/callback",
            "SignOutURL": input("Enter the URL of the sign-out page for the app client [default: https://localhost:3000/sign-out]: ")
            or "https://localhost:3000/sign-out",
            "ReplyEmailAddress": input("Enter the email address to send from: "),
        }
    elif template_name == "dynamo":
        return {}
    elif template_name == "codebuild":
        return {
            "GitHubSourceRepo": input("Enter the GitHub repository URL for the source code: "),
            "S3BucketName": input("Enter the name of the S3 bucket where build artifacts will be stored: "),
        }
    elif template_name == "cloudwatch":
        return {
            "LogGroupName": input("Enter the name for the CloudWatch Log Group [default: /mud/game-logs]: ") or "/mud/game-logs",
            "RetentionInDays": input("Enter the number of days to retain logs [default: 30]: ") or "30",
            "MetricNamespace": input("Enter the namespace for CloudWatch Metrics [default: MUD/Application]: ") or "MUD/Application",
        }
    return {}

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
    except ClientError as err:
        print(f"Error in stack operation: {err}")

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

        # Update the configuration
        if "Server" not in config:
            config["Server"] = {"Port": 9050}
        if "Aws" not in config:
            config["Aws"] = {"Region": "us-east-1"}
        if "Game" not in config:
            config["Game"] = {"Balance": 0.25, "AutoSave": 5, "StartingHealth": 10, "StartingEssence": 3}
        if "Logging" not in config:
            config["Logging"] = {"ApplicationName": "mud", "LogLevel": 20}
        if "Cognito" not in config:
            config["Cognito"] = {}

        # Update the configuration structure
        config.setdefault("Server", {})["Port"] = config.get("Port", 9050)
        config.setdefault("Aws", {})["Region"] = config.get("Region", "us-east-1")
        config.setdefault("Cognito", {}).update(config_updates.get("Cognito", {}))
        config.setdefault("Game", {}).update({
            "Balance": config.get("Balance", 0.25),
            "AutoSave": config.get("AutoSave", 5),
            "StartingHealth": config.get("StartingHealth", 10),
            "StartingEssence": config.get("StartingEssence", 3),
        })
        config.setdefault("Logging", {}).update({
            "ApplicationName": "mud",
            "LogLevel": 20,
            "LogGroup": config_updates.get("CloudWatch", {}).get("LogGroupName", "/mud"),
            "LogStream": "application",
            "MetricNamespace": config_updates.get("CloudWatch", {}).get("MetricNamespace", "MUD/Application"),
        })
        config["Cognito"].update({
            "UserPoolId": config_updates.get("UserPoolId", ""),
            "UserPoolClientSecret": config_updates.get("UserPoolClientSecret", ""),
            "UserPoolClientId": config_updates.get("UserPoolClientId", ""),
            "UserPoolDomain": config_updates.get("UserPoolDomain", ""),
            "UserPoolArn": config_updates.get("UserPoolArn", ""),
        })

        with open(CONFIG_PATH, "w", encoding="utf-8") as file:
            yaml.dump(config, file, default_flow_style=False)

        print("Configuration file updated successfully.")
    except Exception as err:
        print(f"An error occurred while updating configuration file: {err}")

def main():
    cloudformation_client = boto3.client("cloudformation")

    # Deploy Cognito stack
    cognito_parameters = prompt_for_parameters("cognito")
    cognito_template = load_template(COGNITO_TEMPLATE_PATH)
    deploy_stack(cloudformation_client, COGNITO_STACK_NAME, cognito_template, cognito_parameters)
    cognito_outputs = get_stack_outputs(cloudformation_client, COGNITO_STACK_NAME)

    # Deploy DynamoDB stack
    dynamo_parameters = prompt_for_parameters("dynamo")
    dynamo_template = load_template(DYNAMO_TEMPLATE_PATH)
    deploy_stack(cloudformation_client, DYNAMO_STACK_NAME, dynamo_template, dynamo_parameters)
    dynamo_outputs = get_stack_outputs(cloudformation_client, DYNAMO_STACK_NAME)

    # Deploy CodeBuild stack
    codebuild_parameters = prompt_for_parameters("codebuild")
    codebuild_parameters.update(cognito_outputs)
    codebuild_parameters.update(dynamo_outputs)
    codebuild_template = load_template(CODEBUILD_TEMPLATE_PATH)
    deploy_stack(cloudformation_client, CODEBUILD_STACK_NAME, codebuild_template, codebuild_parameters)
    codebuild_outputs = get_stack_outputs(cloudformation_client, CODEBUILD_STACK_NAME)

     # Deploy CloudWatch stack
    cloudwatch_parameters = prompt_for_parameters("cloudwatch")
    cloudwatch_template = load_template(CLOUDWATCH_TEMPLATE_PATH)
    deploy_stack(cloudformation_client, CLOUDWATCH_STACK_NAME, cloudwatch_template, cloudwatch_parameters)
    cloudwatch_outputs = get_stack_outputs(cloudformation_client, CLOUDWATCH_STACK_NAME)


    # Update configuration file with outputs from all stacks
    config_updates = {
        "Cognito": cognito_outputs,
        "Dynamo": dynamo_outputs,
        "CodeBuild": codebuild_outputs,
        "CloudWatch": cloudwatch_outputs,
    }
    update_configuration_file(config_updates)

if __name__ == "__main__":
    main()