package core

import (
	"encoding/json"
	"fmt"
	"log"

	bolt "go.etcd.io/bbolt"
)

func (k *KeyPair) WritePlayer(player *Player) error {
	// Create a PlayerData instance containing only the data to be serialized
	pd := PlayerData{
		Name:          player.PlayerID,
		CharacterList: player.CharacterList,
	}

	// Serialize the PlayerData struct to JSON
	playerData, err := json.Marshal(pd)
	if err != nil {
		return err
	}

	// Use the player's Name as the key to store the serialized data
	return k.Put("Players", []byte(player.Name), playerData)
}

func (k *KeyPair) ReadPlayer(playerName string) (string, map[string]uint64, error) {
	playerData, err := k.Get("Players", []byte(playerName))
	if err != nil {
		if err == bolt.ErrBucketNotFound {
			log.Println("Player bucket not found")
			return "", nil, fmt.Errorf("player not found")
		}
		log.Printf("Error reading player data: %v", err)
		return "", nil, fmt.Errorf("database read failed: %w", err)
	}

	if playerData == nil {
		log.Printf("Player %s not found", playerName)
		return "", nil, fmt.Errorf("player not found")
	}

	var pd PlayerData
	if err := json.Unmarshal(playerData, &pd); err != nil {
		log.Printf("Error unmarshalling player data: %v", err)
		return "", nil, fmt.Errorf("unmarshal player data: %w", err)
	}

	return pd.Name, pd.CharacterList, nil
}
