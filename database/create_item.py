import uuid

import boto3
from botocore.exceptions import ClientError


def display_rooms(dynamodb):
    table = dynamodb.Table("rooms")
    response = table.scan()
    rooms = response["Items"]
    print("Available Rooms:")
    for room in rooms:
        print(f"{room['RoomID']}: {room['Title']}")
    return rooms


def prompt_for_room():
    while True:
        try:
            room_id = int(input("Enter room ID (0 to quit): "))
            return room_id
        except ValueError:
            print("Please enter a valid number.")


def display_prototypes(dynamodb):
    table = dynamodb.Table("prototypes")
    response = table.scan()
    prototypes = response["Items"]
    print("Available Prototypes:")
    for prototype in prototypes:
        print(f"{prototype['ID']}: {prototype['Name']}")
    return prototypes


def prompt_for_prototype():
    return input("Enter prototype ID (empty to cancel): ").strip()


def create_new_item_from_prototype(prototype):
    new_id = str(uuid.uuid4())
    new_item = {
        "ID": new_id,
        "Name": prototype["Name"],
        "Description": prototype["Description"],
        "Mass": prototype["Mass"],
        "Value": prototype["Value"],
        "Stackable": prototype["Stackable"],
        "MaxStack": prototype["MaxStack"],
        "Quantity": 1,
        "Wearable": prototype["Wearable"],
        "WornOn": prototype.get("WornOn", []),
        "Verbs": prototype.get("Verbs", {}),
        "Overrides": prototype.get("Overrides", {}),
        "TraitMods": prototype.get("TraitMods", {}),
        "Container": prototype["Container"],
        "IsPrototype": False,
        "IsWorn": False,
        "CanPickUp": prototype["CanPickUp"],
        "Contents": [],
        "Metadata": prototype.get("Metadata", {}),
    }
    return new_item


def add_item_to_room(dynamodb, room, new_item):
    # Save new item to database
    items_table = dynamodb.Table("items")
    try:
        items_table.put_item(Item=new_item)
    except ClientError as e:
        print(f"Error saving new item: {e.response['Error']['Message']}")
        return False

    # Add item to room
    rooms_table = dynamodb.Table("rooms")
    try:
        rooms_table.update_item(
            Key={"RoomID": room["RoomID"]},
            UpdateExpression="SET Items = list_append(if_not_exists(Items, :empty_list), :new_item)",
            ExpressionAttributeValues={":empty_list": [], ":new_item": [new_item["ID"]]},
        )
    except ClientError as e:
        print(f"Error updating room: {e.response['Error']['Message']}")
        return False

    print(f"Added item {new_item['Name']} (ID: {new_item['ID']}) to room {room['RoomID']}")
    return True


def main():
    dynamodb = boto3.resource("dynamodb", region_name="us-east-1")  # Replace with your AWS region

    while True:
        rooms = display_rooms(dynamodb)
        room_id = prompt_for_room()
        if room_id == 0:
            break

        room = next((r for r in rooms if r["RoomID"] == room_id), None)
        if not room:
            print("Room not found.")
            continue

        prototypes = display_prototypes(dynamodb)
        if not prototypes:
            print("No item prototypes found. Please add some prototypes first.")
            continue

        prototype_id = prompt_for_prototype()
        if not prototype_id:
            continue

        selected_prototype = next((p for p in prototypes if p["ID"] == prototype_id), None)
        if not selected_prototype:
            print("Prototype not found.")
            continue

        new_item = create_new_item_from_prototype(selected_prototype)
        if add_item_to_room(dynamodb, room, new_item):
            print(f"Added {new_item['Name']} to room {room['RoomID']}.")
        else:
            print("Failed to add item to room.")


if __name__ == "__main__":
    main()
