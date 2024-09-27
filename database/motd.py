import argparse
import uuid
from datetime import datetime

import boto3


def add_or_update_motd(message, active=True, is_welcome=False):
    dynamodb = boto3.resource("dynamodb")
    table = dynamodb.Table("motd") # type: ignore

    motd_id = "00000000-0000-0000-0000-000000000000" if is_welcome else str(uuid.uuid4())

    try:
        response = table.update_item(
            Key={"motdID": motd_id},
            UpdateExpression="SET #msg = :message, active = :active, createdAt = :created",
            ExpressionAttributeNames={"#msg": "message"},  # 'message' is a reserved word in DynamoDB
            ExpressionAttributeValues={":message": message, ":active": active, ":created": datetime.now().isoformat()},
            ReturnValues="UPDATED_NEW",
        )
        print(f"MOTD {'updated' if is_welcome else 'added'} successfully.")
        print(f"MOTD ID: {motd_id}")
        return response
    except Exception as e:
        print(f"Error adding/updating MOTD: {str(e)}")
        return None


def main():
    parser = argparse.ArgumentParser(description="Add or update a Message of the Day (MOTD)")
    parser.add_argument("message", type=str, help="The MOTD message")
    parser.add_argument("--welcome", action="store_true", help="Set this flag to add/update the welcome message")
    parser.add_argument("--inactive", action="store_true", help="Set this flag to make the MOTD inactive")

    args = parser.parse_args()

    add_or_update_motd(args.message, not args.inactive, args.welcome)


if __name__ == "__main__":
    main()
