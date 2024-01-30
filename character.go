package main

type Character struct {
	CharacterID int64
	RoomID      int64
	Name        string
	Player      *Player
}

func NewCharacter(CharacterID int64, Player *Player) *Character {
	return &Character{
		CharacterID: CharacterID,
		RoomID:      RoomID,
		Name:        Name,
		Player:      Player,
	}
}

func (c *Character) SendMessage(message string) {
	c.Player.SendMessage(message)
}
