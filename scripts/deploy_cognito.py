"""
Script for dpeloyimng the Cognito stack and updating the Config file.
"""

import json

import boto3


STACK_NAME = "MUD-Cognito-Stack"

CONFIG_PATH = "../configuration.json"

CLOUDFORMATION_PATH = "../cloudformation/cognito.yml"


def prompt_for_parameters():
    """
    Prompts the user for parameters as defined in the cognito.yml file and returns them as a dictionary.
    """
    parameters = {
        "UserPoolName": input("Enter the Name of the user pool [default: mud-user-pool]: ") or "mud-user-pool",
        "AppClientName": input("Enter the Name of the app client [default: mud-app-client]: ") or "mud-app-client",
        "CallbackURL": input("Enter the URL of the callback for the app client [default: https://localhost:3000/callback]: ")
        or "https://localhost:3000/callback",
        "SignOutURL": input("Enter the URL of the sign-out page for the app client [default: https://localhost:3000/sign-out]: ")
        or "https://localhost:3000/sign-out",
    }
    return parameters


def update_configuration_file(config_updates):
    """
    Updates the configuration.json file based on the provided parameters.
    """

    try:
        with open(CONFIG_PATH, "r", encoding="utf-8") as file:
            config = json.load(file)

        # Update the configuration based on the stack outputs.
        config.update(config_updates)

        with open(CONFIG_PATH, "w", encoding="utf-8") as file:
            json.dump(config, file, indent=4)

        print("Configuration file updated successfully.")

    except Exception as err:
        print(f"An error occurred: {err}")


def create_or_update_stack(client, stack_name, template_body, parameters):
    """
    Create or update a CloudFormation stack.
    """
    try:
        client.describe_stacks(StackName=stack_name)
        print("Updating existing stack...")
        response = client.update_stack(
            StackName=stack_name,
            TemplateBody=template_body,
            Parameters=parameters,
            Capabilities=["CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"],
        )
    except client.exceptions.ClientError as err:
        if "does not exist" in str(err):
            print("Creating new stack...")
            response = client.create_stack(
                StackName=stack_name,
                TemplateBody=template_body,
                Parameters=parameters,
                Capabilities=["CAPABILITY_IAM", "CAPABILITY_NAMED_IAM"],
            )
        else:
            raise err

    return response


def wait_for_stack_completion(client, stack_name):
    """
    Wait for the CloudFormation stack to complete its creation or update process.
    """
    waiter = client.get_waiter("stack_create_complete")
    print(f"Waiting for stack {stack_name} to complete...")
    waiter.wait(StackName=stack_name)
    print("Stack operation completed.")


def get_stack_outputs(client, stack_name):
    """
    Retrieve the outputs of a CloudFormation stack.
    """
    stack = client.describe_stacks(StackName=stack_name)
    outputs = stack["Stacks"][0]["Outputs"]
    return {output["OutputKey"]: output["OutputValue"] for output in outputs}


def main():
    # Load CloudFormation template
    with open(CLOUDFORMATION_PATH, "r", encoding="utf-8") as file:
        template_body = file.read()

    parameters = prompt_for_parameters()
    cf_parameters = [{"ParameterKey": k, "ParameterValue": v} for k, v in parameters.items()]

    # Initialize Boto3 CloudFormation client
    client = boto3.client("cloudformation")

    # Create or update the stack
    create_or_update_stack(client, STACK_NAME, template_body, cf_parameters)

    # Wait for the stack operation to complete
    wait_for_stack_completion(client, STACK_NAME)

    # Get stack outputs
    outputs = get_stack_outputs(client, STACK_NAME)

    # Update configuration file with stack outputs
    update_configuration_file(outputs)
