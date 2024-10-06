"""
This module loads game data from JSON files and stores it in DynamoDB.

It handles rooms, exits, archetypes, and item prototypes, storing them into their respective DynamoDB tables.
It also loads the data back from DynamoDB and displays it for verification.
"""

import argparse
import json
import logging
from decimal import Decimal

import boto3
from botocore.exceptions import ClientError


def load_json(file_path):
    """
    Loads JSON data from a file.

    Args:
        file_path (str): The path to the JSON file.

    Returns:
        dict: The data loaded from the JSON file.
    """
    with open(file_path, "r", encoding="utf-8") as file:
        return json.load(file)


def convert_to_dynamodb_format(data):
    """
    Converts numeric values to Decimal for DynamoDB compatibility.

    Args:
        data: The data to convert.

    Returns:
        The data with numeric values converted to Decimal.
    """
    if isinstance(data, dict):
        return {k: convert_to_dynamodb_format(v) for k, v in data.items()}
    elif isinstance(data, list):
        return [convert_to_dynamodb_format(v) for v in data]
    elif isinstance(data, float):
        return Decimal(str(data))
    else:
        return data


def store_rooms_and_exits(dynamodb, rooms_data):
    """
    Stores room data into the 'rooms' DynamoDB table and exit data into the 'exits' table.

    Args:
        dynamodb: The DynamoDB resource object.
        rooms_data (dict): The rooms data to store.
    """
    rooms_table = dynamodb.Table("rooms")
    exits_table = dynamodb.Table("exits")
    try:
        with rooms_table.batch_writer() as rooms_batch, exits_table.batch_writer() as exits_batch:
            for room in rooms_data.get("rooms", []):
                room_item = {
                    "RoomID": room["RoomID"],
                    "Area": room["Area"],
                    "Title": room["Title"],
                    "Description": room["Description"],
                    "ItemIDs": room.get("ItemIDs", []),
                }
                rooms_batch.put_item(Item=convert_to_dynamodb_format(room_item))

                for exit_data in room.get("Exits", []):
                    exit_item = {
                        "ExitID": exit_data["ExitID"],
                        "RoomID": room["RoomID"],
                        "Direction": exit_data["Direction"],
                        "TargetRoom": exit_data["TargetRoom"],
                        "Visible": exit_data["Visible"],
                    }
                    exits_batch.put_item(Item=convert_to_dynamodb_format(exit_item))
        print("Room and exit data stored in DynamoDB successfully")
    except ClientError as e:
        logging.error(f"An error occurred while storing rooms and exits: {e.response['Error']['Message']}")
    except Exception as e:
        logging.error(f"An unexpected error occurred while storing rooms and exits: {str(e)}")


def store_archetypes(dynamodb, archetypes_data):
    """
    Stores archetype data into the 'archetypes' DynamoDB table.

    Args:
        dynamodb: The DynamoDB resource object.
        archetypes_data (dict): The archetypes data to store.
    """
    table = dynamodb.Table("archetypes")
    try:
        with table.batch_writer() as batch:
            for name, archetype in archetypes_data.get("archetypes", {}).items():
                archetype_item = {
                    "ArchetypeName": name,
                    "Name": archetype.get("name", ""),
                    "Description": archetype.get("description", ""),
                    "Attributes": archetype.get("Attributes", {}),
                    "Abilities": archetype.get("Abilities", {}),
                }
                batch.put_item(Item=convert_to_dynamodb_format(archetype_item))
        print("Archetype data stored in DynamoDB successfully")
    except ClientError as e:
        logging.error(f"An error occurred while storing archetypes: {e.response['Error']['Message']}")
    except Exception as e:
        logging.error(f"An unexpected error occurred while storing archetypes: {str(e)}")


def store_item_prototypes(dynamodb, prototypes_data):
    """
    Stores item prototype data into the 'prototypes' DynamoDB table.

    Args:
        dynamodb: The DynamoDB resource object.
        prototypes_data (dict): The item prototypes data to store.
    """
    table = dynamodb.Table("prototypes")
    try:
        with table.batch_writer() as batch:
            for prototype in prototypes_data.get("itemPrototypes", []):
                prototype["PrototypeID"] = prototype.pop("ID")
                batch.put_item(Item=convert_to_dynamodb_format(prototype))
        print("Item prototype data stored in DynamoDB successfully")
    except ClientError as err:
        logging.error(f"An error occurred while storing item prototypes: {err.response['Error']['Message']}")
    except Exception as err:
        logging.error(f"An unexpected error occurred while storing item prototypes: {str(err)}")


def load_rooms_and_exits(dynamodb):
    """
    Loads room data from the 'rooms' and 'exits' DynamoDB tables.

    Args:
        dynamodb: The DynamoDB resource object.

    Returns:
        dict: A dictionary of room data with exits included.
    """
    rooms_table = dynamodb.Table("rooms")
    exits_table = dynamodb.Table("exits")
    try:
        rooms_response = rooms_table.scan()
        exits_response = exits_table.scan()

        rooms = {item["RoomID"]: item for item in rooms_response.get("Items", [])}

        for exit_item in exits_response.get("Items", []):
            room_id = exit_item["RoomID"]
            if room_id in rooms:
                if "Exits" not in rooms[room_id]:
                    rooms[room_id]["Exits"] = []
                rooms[room_id]["Exits"].append(
                    {
                        "ExitID": exit_item["ExitID"],
                        "Direction": exit_item["Direction"],
                        "TargetRoom": exit_item["TargetRoom"],
                        "Visible": exit_item["Visible"],
                    }
                )

        print("Room and exit data loaded from DynamoDB successfully")
        return rooms
    except ClientError as e:
        logging.error(f"An error occurred while loading rooms and exits: {e.response['Error']['Message']}")
        return {}
    except Exception as e:
        logging.error(f"An unexpected error occurred while loading rooms and exits: {str(e)}")
        return {}


def load_archetypes(dynamodb):
    """
    Loads archetype data from the 'archetypes' DynamoDB table.

    Args:
        dynamodb: The DynamoDB resource object.

    Returns:
        dict: A dictionary containing the archetypes.
    """
    table = dynamodb.Table("archetypes")
    try:
        response = table.scan()
        archetypes = {"archetypes": {item["ArchetypeName"]: item for item in response.get("Items", [])}}
        print("Archetype data loaded from DynamoDB successfully")
        return archetypes
    except ClientError as e:
        logging.error(f"An error occurred while loading archetypes: {e.response['Error']['Message']}")
        return {}
    except Exception as e:
        logging.error(f"An unexpected error occurred while loading archetypes: {str(e)}")
        return {}


def load_item_prototypes(dynamodb):
    """
    Loads item prototype data from the 'prototypes' DynamoDB table.

    Args:
        dynamodb: The DynamoDB resource object.

    Returns:
        dict: A dictionary containing the item prototypes.
    """
    table = dynamodb.Table("prototypes")
    try:
        response = table.scan()
        prototypes = {"itemPrototypes": response.get("Items", [])}
        print("Item prototype data loaded from DynamoDB successfully")
        return prototypes
    except ClientError as e:
        logging.error(f"An error occurred while loading item prototypes: {e.response['Error']['Message']}")
        return {}
    except Exception as e:
        logging.error(f"An unexpected error occurred while loading item prototypes: {str(e)}")
        return {}


def display_rooms(rooms):
    """
    Displays room information.

    Args:
        rooms (dict): The rooms data to display.
    """
    print("Rooms:")
    for room_id, room in rooms.items():
        print(f"Room {room_id}: {room.get('Title', 'No Title')}")
        print(f"  Area: {room.get('Area', 'Unknown')}")
        print(f"  Description: {room.get('Description', 'No description')}")
        for exit_data in room.get("Exits", []):
            print(f"  Exit {exit_data['Direction']} (ID: {exit_data['ExitID']}) to room {exit_data['TargetRoom']}")
        print()


def display_archetypes(archetypes):
    """
    Displays archetype information.

    Args:
        archetypes (dict): The archetypes data to display.
    """
    print("Archetypes:")
    for name, archetype in archetypes.get("archetypes", {}).items():
        print(f"Name: {name}")
        print(f"  Description: {archetype.get('Description', 'No description')}")
        print("  Attributes:")
        for attr, value in archetype.get("Attributes", {}).items():
            print(f"    {attr}: {value}")
        print("  Abilities:")
        for ability, value in archetype.get("Abilities", {}).items():
            print(f"    {ability}: {value}")
        print()


def display_item_prototypes(prototypes):
    """
    Displays item prototype information.

    Args:
        prototypes (dict): The item prototypes data to display.
    """
    print("Item Prototypes:")
    for prototype in prototypes.get("itemPrototypes", []):
        print(f"ID: {prototype.get('PrototypeID', 'No ID')}")
        print(f"  Name: {prototype.get('Name', 'No Name')}")
        print(f"  Description: {prototype.get('Description', 'No description')}")
        print(f"  Mass: {prototype.get('mass', 'Unknown')}")
        print(f"  Value: {prototype.get('value', 'Unknown')}")
        print(f"  Wearable: {prototype.get('wearable', False)}")
        if prototype.get("wearable"):
            print(f"  Worn on: {', '.join(prototype.get('worn_on', []))}")
        print()


def main():
    """
    Main function to load game data from JSON files and store it in DynamoDB.

    - Parses command-line arguments for file paths and AWS region.
    - Loads data from JSON files.
    - Stores data into DynamoDB tables.
    - Loads data back from DynamoDB and displays it.
    """
    parser = argparse.ArgumentParser(description="Load and store game data in DynamoDB.")
    parser.add_argument("-r", "--rooms", default="test_rooms.json", help="Path to the Rooms JSON file.")
    parser.add_argument("-a", "--archetypes", default="test_archetypes.json", help="Path to the Archetypes JSON file.")
    parser.add_argument("-p", "--prototypes", default="test_prototypes.json", help="Path to the Prototypes JSON file.")
    parser.add_argument("-region", default="us-east-1", help="AWS region for DynamoDB.")
    args = parser.parse_args()

    logging.basicConfig(level=logging.INFO)

    try:
        dynamodb = boto3.resource("dynamodb", region_name=args.region)

        # Load and store rooms and exits
        rooms_data = load_json(args.rooms)
        store_rooms_and_exits(dynamodb, rooms_data)

        # Load and store archetypes
        archetypes_data = load_json(args.archetypes)
        store_archetypes(dynamodb, archetypes_data)

        # Load and store item prototypes
        prototypes_data = load_json(args.prototypes)
        store_item_prototypes(dynamodb, prototypes_data)

        # Load data from DynamoDB and display
        loaded_rooms = load_rooms_and_exits(dynamodb)
        display_rooms(loaded_rooms)

        loaded_archetypes = load_archetypes(dynamodb)
        display_archetypes(loaded_archetypes)

        loaded_prototypes = load_item_prototypes(dynamodb)
        display_item_prototypes(loaded_prototypes)

    except Exception as e:
        logging.error(f"An unexpected error occurred: {str(e)}")


if __name__ == "__main__":
    main()
