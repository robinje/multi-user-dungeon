package core

import (
	"encoding/json"
	"fmt"

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

	if err == bolt.ErrBucketNotFound {
		return "", nil, fmt.Errorf("player not found")
	}

	if err != nil {
		return "", nil, err
	}

	if playerData == nil {
		return "", nil, fmt.Errorf("player not found")
	}

	// Deserialize the JSON into a PlayerData struct
	var pd PlayerData
	err = json.Unmarshal(playerData, &pd)
	if err != nil {
		return "", nil, err
	}

	return pd.Name, pd.CharacterList, nil
}
