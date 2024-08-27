"""
Script for Deploying the AWS Components.

This project is licensed under the Apache 2.0 License. See the LICENSE file for more details.
"""

import yaml

import boto3

# Constants for stack names
COGNITO_STACK_NAME = "MUD-Cognito-Stack"
DYNAMO_STACK_NAME = "MUD-DynamoDB-Stack"
CODEBUILD_STACK_NAME = "MUD-CodeBuild-Stack"

# Paths to the CloudFormation templates
COGNITO_TEMPLATE_PATH = "../cloudformation/cognito.yml"
DYNAMO_TEMPLATE_PATH = "../cloudformation/dynamo.yml"
CODEBUILD_TEMPLATE_PATH = "../cloudformation/codebuild.yml"

# Configuration file path
CONFIG_PATH = "../mud/config.yml"


def prompt_for_parameters(template_name) -> dict:
    """
    Prompts the user for parameters based on the specified template name.
    """
    if template_name == "cognito":
        parameters: dict = {
            "UserPoolName": input("Enter the Name of the user pool [default: mud-user-pool]: ") or "mud-user-pool",
            "AppClientName": input("Enter the Name of the app client [default: mud-app-client]: ") or "mud-app-client",
            "CallbackURL": input("Enter the URL of the callback for the app client [default: https://localhost:3000/callback]: ") or "https://localhost:3000/callback",
            "SignOutURL": input("Enter the URL of the sign-out page for the app client [default: https://localhost:3000/sign-out]: ") or "https://localhost:3000/sign-out",
            "ReplyEmailAddress": input("Enter the email address to send from: "),
        }
    elif template_name == "dynamo":
        # DynamoDB template doesn't require any parameters
        parameters = {}
    elif template_name == "codebuild":
        parameters = {
            "GitHubSourceRepo": input("Enter the GitHub repository URL for the source code: "),
            "S3BucketName": input("Enter the name of the S3 bucket where build artifacts will be stored: "),
        }
    return parameters


def load_template(template_path) -> str:
    """
    Loads a CloudFormation template from the specified path.
    """
    with open(template_path, "r", encoding="utf-8") as file:
        return file.read()


def deploy_stack(client, stack_name, template_body, parameters) -> None:
    """
    Deploy or update a CloudFormation stack with the given parameters.
    """
    cf_parameters: list = [{"ParameterKey": k, "ParameterValue": v} for k, v in parameters.items()]
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
    except client.exceptions.ClientError as err:
        print(f"Error in stack operation: {err}")


def stack_exists(client, stack_name) -> bool:
    """
    Check if a CloudFormation stack exists.
    """
    try:
        client.describe_stacks(StackName=stack_name)
        return True
    except client.exceptions.ClientError:
        return False


def wait_for_stack_completion(client, stack_name) -> None:
    """
    Wait for the CloudFormation stack to complete its operation.
    """
    waiter = client.get_waiter("stack_create_complete")
    print(f"Waiting for stack {stack_name} to complete...")
    waiter.wait(StackName=stack_name)
    print("Stack operation completed.")


def get_stack_outputs(client, stack_name) -> dict:
    """
    Retrieve the outputs of a CloudFormation stack.
    """
    stack = client.describe_stacks(StackName=stack_name)
    outputs = stack["Stacks"][0]["Outputs"]
    return {output["OutputKey"]: output["OutputValue"] for output in outputs}


def start_codebuild_project(codebuild_client, project_name):
    """
    Starts an AWS CodeBuild project.
    """
    response = codebuild_client.start_build(projectName=project_name)
    build_id = response['build']['id']
    print(f"Started CodeBuild project: {project_name}, Build ID: {build_id}")
    return build_id


def wait_for_codebuild_completion(codebuild_client, build_id):
    """
    Waits for the specified CodeBuild project build to complete.
    """
    print(f"Waiting for CodeBuild build {build_id} to complete...")
    waiter = codebuild_client.get_waiter('build_completed')
    waiter.wait(ids=[build_id])
    print("CodeBuild build completed.")


def update_configuration_file(config_updates) -> None:
    """
    Updates the config.yml file based on the provided parameters.
    """
    try:
        with open(CONFIG_PATH, "r", encoding="utf-8") as file:
            config = yaml.safe_load(file) or {}

        config.update(config_updates)

        with open(CONFIG_PATH, "w", encoding="utf-8") as file:
            yaml.dump(config, file, default_flow_style=False)

        print("Configuration file updated successfully.")
    except Exception as err:
        print(f"An error occurred while updating configuration file: {err}")


def main() -> None:
    # Initialize Boto3 clients
    cloudformation_client = boto3.client("cloudformation")
    codebuild_client = boto3.client("codebuild")

    # Deploy Cognito stack
    cognito_parameters: dict = prompt_for_parameters("cognito")
    cognito_template: str = load_template(COGNITO_TEMPLATE_PATH)
    deploy_stack(cloudformation_client, COGNITO_STACK_NAME, cognito_template, cognito_parameters)
    cognito_outputs: dict = get_stack_outputs(cloudformation_client, COGNITO_STACK_NAME)

    # Deploy DynamoDB stack
    dynamo_parameters: dict = prompt_for_parameters("dynamo")
    dynamo_template: str = load_template(DYNAMO_TEMPLATE_PATH)
    deploy_stack(cloudformation_client, DYNAMO_STACK_NAME, dynamo_template, dynamo_parameters)
    dynamo_outputs: dict = get_stack_outputs(cloudformation_client, DYNAMO_STACK_NAME)

    # Deploy CodeBuild stack
    codebuild_parameters: dict = prompt_for_parameters("codebuild")
    codebuild_parameters.update(cognito_outputs)  # Add Cognito outputs as parameters for CodeBuild stack
    codebuild_parameters.update(dynamo_outputs)   # Add DynamoDB outputs as parameters for CodeBuild stack
    codebuild_template: str = load_template(CODEBUILD_TEMPLATE_PATH)
    deploy_stack(cloudformation_client, CODEBUILD_STACK_NAME, codebuild_template, codebuild_parameters)
    codebuild_outputs: dict = get_stack_outputs(cloudformation_client, CODEBUILD_STACK_NAME)

    # Start CodeBuild project
    codebuild_project_name = codebuild_outputs["CodeBuildProjectName"]
    build_id = start_codebuild_project(codebuild_client, codebuild_project_name)
    wait_for_codebuild_completion(codebuild_client, build_id)

    # Update configuration file with outputs from all stacks
    config_updates: dict = {**cognito_outputs, **dynamo_outputs, **codebuild_outputs}
    update_configuration_file(config_updates)


if __name__ == "__main__":
    main()