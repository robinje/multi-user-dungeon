"""
This module adds an item based on a prototype to a room.
"""

import uuid
from decimal import Decimal

import boto3
from botocore.exceptions import ClientError


def display_rooms(dynamodb) -> list:
    """
    Fetches and displays all rooms from the 'rooms' DynamoDB table.

    Args:
        dynamodb: The DynamoDB resource object.

    Returns:
        A list of room dictionaries.
    """
    try:
        table = dynamodb.Table("rooms")
        response = table.scan()
        rooms = response.get("Items", [])
        if not rooms:
            print("No rooms found.")
            return []

        print("Available Rooms:")
        for room in rooms:
            room_id = int(room["RoomID"])
            title = room.get("Title", "No Title")
            print(f"{room_id}: {title}")
        return rooms
    except ClientError as e:
        print(f"Error fetching rooms: {e.response['Error']['Message']}")
        return []


def prompt_for_room() -> int:
    """
    Prompts the user to enter a room ID.

    Returns:
        The room ID entered by the user, or 0 to quit.
    """
    while True:
        try:
            room_id = int(input("Enter room ID (0 to quit): "))
            return room_id
        except ValueError:
            print("Please enter a valid number.")


def display_prototypes(dynamodb) -> list:
    """
    Fetches and displays all item prototypes from the 'prototypes' DynamoDB table.

    Args:
        dynamodb: The DynamoDB resource object.

    Returns:
        A list of prototype dictionaries.
    """
    try:
        table = dynamodb.Table("prototypes")
        response = table.scan()
        prototypes = response.get("Items", [])
        if not prototypes:
            print("No prototypes found.")
            return []

        print("Available Prototypes:")
        for prototype in prototypes:
            prototype_id = prototype["ID"]
            name = prototype.get("Name", "No Name")
            print(f"{prototype_id}: {name}")
        return prototypes
    except ClientError as e:
        print(f"Error fetching prototypes: {e.response['Error']['Message']}")
        return []


def prompt_for_prototype() -> str:
    """
    Prompts the user to enter a prototype ID.

    Returns:
        The prototype ID entered by the user, or an empty string to cancel.
    """
    return input("Enter prototype ID (empty to cancel): ").strip()


def create_new_item_from_prototype(prototype: dict) -> dict:
    """
    Creates a new item based on the given prototype.

    Args:
        prototype: The prototype dictionary.

    Returns:
        A new item dictionary with a unique ID and properties copied from the prototype.
    """
    new_id = str(uuid.uuid4())
    new_item: dict = {
        "ID": new_id,
        "Name": prototype.get("Name", "Unnamed Item"),
        "Description": prototype.get("Description", ""),
        "Mass": prototype.get("Mass", Decimal(0)),
        "Value": prototype.get("Value", Decimal(0)),
        "Stackable": prototype.get("Stackable", False),
        "MaxStack": prototype.get("MaxStack", Decimal(1)),
        "Quantity": Decimal(1),
        "Wearable": prototype.get("Wearable", False),
        "WornOn": prototype.get("WornOn", []),
        "Verbs": prototype.get("Verbs", {}),
        "Overrides": prototype.get("Overrides", {}),
        "TraitMods": prototype.get("TraitMods", {}),
        "Container": prototype.get("Container", False),
        "IsPrototype": False,  # This is a real item, not a prototype
        "IsWorn": False,
        "CanPickUp": prototype.get("CanPickUp", True),
        "Contents": [],  # Contents are empty initially
        "Metadata": prototype.get("Metadata", {}),
    }
    return new_item


def add_item_to_room(dynamodb, room: dict, new_item: dict) -> bool:
    """
    Adds the new item to the 'items' table and updates the room to include the item.

    Args:
        dynamodb: The DynamoDB resource object.
        room: The room dictionary where the item will be added.
        new_item: The item dictionary to add.

    Returns:
        True if the item was successfully added to the room, False otherwise.
    """
    # Save new item to the 'items' table
    items_table = dynamodb.Table("items")
    try:
        items_table.put_item(Item=new_item)
    except ClientError as e:
        print(f"Error saving new item: {e.response['Error']['Message']}")
        return False

    # Add item ID to room's ItemIDs list in the 'rooms' table
    rooms_table = dynamodb.Table("rooms")
    try:
        # Ensure that the RoomID is of the correct type
        room_id = room["RoomID"]
        if isinstance(room_id, Decimal):
            room_id = int(room_id)
        elif not isinstance(room_id, int):
            room_id = int(room_id)

        rooms_table.update_item(
            Key={"RoomID": room_id},
            UpdateExpression="SET ItemIDs = list_append(if_not_exists(ItemIDs, :empty_list), :new_item)",
            ExpressionAttributeValues={
                ":empty_list": [],
                ":new_item": [new_item["ID"]],
            },
            ConditionExpression="attribute_exists(RoomID)",
        )
    except ClientError as e:
        print(f"Error updating room: {e.response['Error']['Message']}")
        # Optionally, delete the item we just added to maintain consistency
        try:
            items_table.delete_item(Key={"ID": new_item["ID"]})
        except ClientError as del_e:
            print(f"Error rolling back item addition: {del_e.response['Error']['Message']}")
        return False

    print(f"Added item '{new_item['Name']}' (ID: {new_item['ID']}) to room {room_id}")
    return True


def main() -> None:
    """
    Main function to run the script.

    - Connects to DynamoDB.
    - Allows user to select a room and an item prototype.
    - Creates a new item based on the prototype.
    - Adds the new item to the selected room.
    """
    dynamodb = boto3.resource("dynamodb", region_name="us-east-1")  # Replace with your AWS region

    while True:
        rooms = display_rooms(dynamodb)
        if not rooms:
            print("No rooms available. Exiting.")
            break

        room_id: int = prompt_for_room()
        if room_id == 0:
            print("Exiting.")
            break

        # Find the room with the matching RoomID
        room = next((r for r in rooms if int(r["RoomID"]) == room_id), None)
        if not room:
            print("Room not found.")
            continue

        prototypes = display_prototypes(dynamodb)
        if not prototypes:
            print("No item prototypes found. Please add some prototypes first.")
            continue

        prototype_id = prompt_for_prototype()
        if not prototype_id:
            print("No prototype selected. Returning to room selection.")
            continue

        selected_prototype = next((p for p in prototypes if p["ID"] == prototype_id), None)
        if not selected_prototype:
            print("Prototype not found.")
            continue

        new_item: dict = create_new_item_from_prototype(selected_prototype)
        if add_item_to_room(dynamodb, room, new_item):
            print(f"Successfully added '{new_item['Name']}' to room {room_id}.")
        else:
            print("Failed to add item to room.")


if __name__ == "__main__":
    main()
