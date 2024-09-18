import argparse
import json
import logging

import boto3
from botocore.exceptions import ClientError


def load_json(file_path):
    with open(file_path, "r") as file:
        return json.load(file)


def store_rooms(dynamodb, rooms):
    table = dynamodb.Table("rooms")
    with table.batch_writer() as batch:
        for room in rooms.values():
            batch.put_item(Item=room)
    print("Room data stored in DynamoDB successfully")


def store_archetypes(dynamodb, archetypes):
    table = dynamodb.Table("archetypes")
    with table.batch_writer() as batch:
        for archetype in archetypes["Archetypes"].values():
            batch.put_item(Item=archetype)
    print("Archetype data stored in DynamoDB successfully")


def store_prototypes(dynamodb, prototypes):
    table = dynamodb.Table("prototypes")
    with table.batch_writer() as batch:
        for prototype in prototypes["ItemPrototypes"]:
            batch.put_item(Item=prototype)
    print("Prototype data stored in DynamoDB successfully")


def load_rooms(dynamodb):
    table = dynamodb.Table("rooms")
    response = table.scan()
    rooms = {item["RoomID"]: item for item in response["Items"]}
    print("Room data loaded from DynamoDB successfully")
    return rooms


def load_archetypes(dynamodb):
    table = dynamodb.Table("archetypes")
    response = table.scan()
    archetypes = {"Archetypes": {item["Name"]: item for item in response["Items"]}}
    print("Archetype data loaded from DynamoDB successfully")
    return archetypes


def load_prototypes(dynamodb):
    table = dynamodb.Table("prototypes")
    response = table.scan()
    prototypes = {"ItemPrototypes": response["Items"]}
    print("Prototype data loaded from DynamoDB successfully")
    return prototypes


def display_rooms(rooms):
    print("Rooms:")
    for room_id, room in rooms.items():
        print(f"Room {room_id}: {room['Title']}")
        for exit_dir, exit_data in room.get("Exits", {}).items():
            print(f"  Exit {exit_dir} to room {exit_data['TargetRoom']}")


def display_archetypes(archetypes):
    for name, archetype in archetypes["Archetypes"].items():
        print(f"Name: {name}, Description: {archetype['Description']}")


def display_prototypes(prototypes):
    for prototype in prototypes["ItemPrototypes"]:
        print(f"ID: {prototype['ID']}, Name: {prototype['Name']}, Description: {prototype['Description']}")


def main():
    parser = argparse.ArgumentParser(description="Load and store game data in DynamoDB.")
    parser.add_argument("-r", "--rooms", default="test_rooms.json", help="Path to the Rooms JSON file.")
    parser.add_argument("-a", "--archetypes", default="test_archetypes.json", help="Path to the Archetypes JSON file.")
    parser.add_argument("-p", "--prototypes", default="test_prototypes.json", help="Path to the Prototypes JSON file.")
    parser.add_argument("-region", default="us-east-1", help="AWS region for DynamoDB.")
    args = parser.parse_args()

    logging.basicConfig(level=logging.INFO)

    try:
        dynamodb = boto3.resource("dynamodb", region_name=args.region)

        # Load and store rooms
        rooms = load_json(args.rooms)
        store_rooms(dynamodb, rooms)

        # Load and store archetypes
        archetypes = load_json(args.archetypes)
        store_archetypes(dynamodb, archetypes)

        # Load and store prototypes
        prototypes = load_json(args.prototypes)
        store_prototypes(dynamodb, prototypes)

        # Load data from DynamoDB and display
        loaded_rooms = load_rooms(dynamodb)
        display_rooms(loaded_rooms)

        loaded_archetypes = load_archetypes(dynamodb)
        display_archetypes(loaded_archetypes)

        loaded_prototypes = load_prototypes(dynamodb)
        display_prototypes(loaded_prototypes)

    except ClientError as e:
        logging.error(f"An error occurred: {e.response['Error']['Message']}")
    except Exception as e:
        logging.error(f"An unexpected error occurred: {str(e)}")


if __name__ == "__main__":
    main()
