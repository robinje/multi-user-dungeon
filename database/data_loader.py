"""
This module loads game data from JSON files and stores it in DynamoDB.

It handles rooms, archetypes, and prototypes, storing them into their respective DynamoDB tables.
It also loads the data back from DynamoDB and displays it for verification.
"""

import argparse
import json
import logging

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


def store_rooms(dynamodb, rooms) -> None:
    """
    Stores room data into the 'rooms' DynamoDB table and exits into the 'exits' table.

    Args:
        dynamodb: The DynamoDB resource object.
        rooms (dict): The rooms data to store.
    """
    rooms_table = dynamodb.Table("rooms")
    exits_table = dynamodb.Table("exits")
    try:
        with rooms_table.batch_writer() as rooms_batch, exits_table.batch_writer() as exits_batch:
            for room_id, room in rooms.items():
                # Ensure that room data includes the primary key 'RoomID'
                if "RoomID" not in room:
                    room["RoomID"] = int(room_id)
                else:
                    room["RoomID"] = int(room["RoomID"])
                # Extract 'Exits' from room data
                exits = room.pop("Exits", {})
                # Store room data without 'Exits'
                rooms_batch.put_item(Item=room)
                # Store exits separately
                for exit_dir, exit_data in exits.items():
                    exit_item = {
                        "RoomID": room["RoomID"],
                        "Direction": exit_dir,
                        "TargetRoom": exit_data["TargetRoom"],
                        "Visible": exit_data.get("Visible", True),
                    }
                    exits_batch.put_item(Item=exit_item)
        print("Room and exit data stored in DynamoDB successfully")
    except ClientError as e:
        logging.error(f"An error occurred while storing rooms and exits: {e.response['Error']['Message']}")
    except Exception as e:
        logging.error(f"An unexpected error occurred while storing rooms and exits: {str(e)}")


def store_archetypes(dynamodb, archetypes) -> None:
    """
    Stores archetype data into the 'archetypes' DynamoDB table.

    Args:
        dynamodb: The DynamoDB resource object.
        archetypes (dict): The archetypes data to store.
    """
    table = dynamodb.Table("archetypes")
    try:
        with table.batch_writer() as batch:
            for name, archetype in archetypes.get("Archetypes", {}).items():
                # Ensure the archetype has a 'Name' key
                if "Name" not in archetype:
                    archetype["Name"] = name
                batch.put_item(Item=archetype)
        print("Archetype data stored in DynamoDB successfully")
    except ClientError as e:
        logging.error(f"An error occurred while storing archetypes: {e.response['Error']['Message']}")
    except Exception as e:
        logging.error(f"An unexpected error occurred while storing archetypes: {str(e)}")


def store_prototypes(dynamodb, prototypes) -> None:
    """
    Stores item prototype data into the 'prototypes' DynamoDB table.

    Args:
        dynamodb: The DynamoDB resource object.
        prototypes (dict): The prototypes data to store.
    """
    table = dynamodb.Table("prototypes")
    try:
        with table.batch_writer() as batch:
            for prototype in prototypes.get("ItemPrototypes", []):
                # Ensure the prototype has an 'ID' key
                if "ID" not in prototype:
                    logging.warning(f"Prototype missing 'ID': {prototype}")
                    continue
                batch.put_item(Item=prototype)
        print("Prototype data stored in DynamoDB successfully")
    except ClientError as e:
        logging.error(f"An error occurred while storing prototypes: {e.response['Error']['Message']}")
    except Exception as e:
        logging.error(f"An unexpected error occurred while storing prototypes: {str(e)}")


def load_rooms(dynamodb) -> dict:
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
        # Load rooms
        response = rooms_table.scan()
        rooms = {str(item["RoomID"]): item for item in response.get("Items", [])}
        # Load exits
        response = exits_table.scan()
        for exit_item in response.get("Items", []):
            room_id = str(exit_item["RoomID"])
            direction = exit_item["Direction"]
            if room_id in rooms:
                room = rooms[room_id]
                if "Exits" not in room:
                    room["Exits"] = {}
                room["Exits"][direction] = {
                    "TargetRoom": exit_item["TargetRoom"],
                    "Visible": exit_item.get("Visible", True),
                }
        print("Room data loaded from DynamoDB successfully")
        return rooms
    except ClientError as e:
        logging.error(f"An error occurred while loading rooms and exits: {e.response['Error']['Message']}")
        return {}
    except Exception as e:
        logging.error(f"An unexpected error occurred while loading rooms and exits: {str(e)}")
        return {}


def load_archetypes(dynamodb) -> dict:
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
        archetypes = {"Archetypes": {item["Name"]: item for item in response.get("Items", [])}}
        print("Archetype data loaded from DynamoDB successfully")
        return archetypes
    except ClientError as e:
        logging.error(f"An error occurred while loading archetypes: {e.response['Error']['Message']}")
        return {}
    except Exception as e:
        logging.error(f"An unexpected error occurred while loading archetypes: {str(e)}")
        return {}


def load_prototypes(dynamodb) -> dict:
    """
    Loads item prototype data from the 'prototypes' DynamoDB table.

    Args:
        dynamodb: The DynamoDB resource object.

    Returns:
        dict: A dictionary containing the prototypes.
    """
    table = dynamodb.Table("prototypes")
    try:
        response = table.scan()
        prototypes = {"ItemPrototypes": response.get("Items", [])}
        print("Prototype data loaded from DynamoDB successfully")
        return prototypes
    except ClientError as e:
        logging.error(f"An error occurred while loading prototypes: {e.response['Error']['Message']}")
        return {}
    except Exception as e:
        logging.error(f"An unexpected error occurred while loading prototypes: {str(e)}")
        return {}


def display_rooms(rooms) -> None:
    """
    Displays room information.

    Args:
        rooms (dict): The rooms data to display.
    """
    print("Rooms:")
    for room_id, room in rooms.items():
        title = room.get("Title", "No Title")
        print(f"Room {room_id}: {title}")
        exits = room.get("Exits", {})
        for exit_dir, exit_data in exits.items():
            target_room = exit_data.get("TargetRoom")
            print(f"  Exit {exit_dir} to room {target_room}")


def display_archetypes(archetypes) -> None:
    """
    Displays archetype information.

    Args:
        archetypes (dict): The archetypes data to display.
    """
    print("Archetypes:")
    for name, archetype in archetypes.get("Archetypes", {}).items():
        description = archetype.get("Description", "")
        print(f"Name: {name}, Description: {description}")


def display_prototypes(prototypes) -> None:
    """
    Displays prototype information.

    Args:
        prototypes (dict): The prototypes data to display.
    """
    print("Prototypes:")
    for prototype in prototypes.get("ItemPrototypes", []):
        prototype_id = prototype.get("ID", "No ID")
        name = prototype.get("Name", "No Name")
        description = prototype.get("Description", "")
        print(f"ID: {prototype_id}, Name: {name}, Description: {description}")


def main() -> None:
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

    # Configure logging
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
        loaded_rooms: dict = load_rooms(dynamodb)
        display_rooms(loaded_rooms)

        loaded_archetypes: dict = load_archetypes(dynamodb)
        display_archetypes(loaded_archetypes)

        loaded_prototypes: dict = load_prototypes(dynamodb)
        display_prototypes(loaded_prototypes)

    except ClientError as e:
        logging.error(f"An error occurred: {e.response['Error']['Message']}")
    except Exception as e:
        logging.error(f"An unexpected error occurred: {str(e)}")


if __name__ == "__main__":
    main()
