## Player Table

| Field | Type | Description |
| --- | --- | --- |
| `PlayerID` | `STRING` | Email of the player. |
| `CharacterList` | `MAP` | A map of the player's character and their UUID |
| `SeenMotDs` | `LIST` | List of viewed messages of the day.
| --- | --- | --- |

The `PlayerID` is the email address of the player.

`CharacterList` is a map of the player's characters. The key is the character's name and the value is the character's UUID.

`SeenMotDs` is a list of the messages of the day that the player has seen.

## Character Table

| Field | Type | Description |
| --- | --- | --- |
| `CharacterID` | `STRING` | UUID of the character. |
| `PlayerID` | `STRING` | Email of the player. |
| `CharacterName` | `STRING` | Name of the character. |
| `RoomID` | `NUMBER` | ID of the room the character is in. |
| `Inventory` | `MAP` | A map of the character's inventory, the key is the location of the object, the value is the object's UUID. |
| `Attributes` | `MAP` | A map of the character's attributes. |
| `Abilites` | `MAP` | A map of the character's abilities. |
| `Essence` | `NUMBER` | The character's essence. |
| `Health` | `NUMBER` | The character's health. |



## Rooms Table


## Exits Table


## Items Table


## Item Prototypes Table


## Archetypes Table


## MOTD Table


