package main

type Room struct {
	RoomID      int64
	Area        string
	Title       string
	Description string
	Exits       map[string]*Exit
	Characters  map[uint64]*Character
}

type Exit struct {
	ExitID     int64
	TargetRoom int64
	Visibile   bool
	Direction  string
}

func (d *DataBase) LoadRooms() (map[int64]*Room, error) {
	rooms := make(map[int64]*Room)

	// Open the BoltDB connection
	db, err := bolt.Open(d.DataFile, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("error opening BoltDB file: %w", err)
	}
	defer db.Close() // Ensure the database is closed when the function returns

	err = db.View(func(tx *bolt.Tx) error {
		roomsBucket := tx.Bucket([]byte("Rooms"))
		if roomsBucket == nil {
			return fmt.Errorf("Rooms bucket not found")
		}

		exitsBucket := tx.Bucket([]byte("Exits"))
		if exitsBucket == nil {
			return fmt.Errorf("Exits bucket not found")
		}

		err := roomsBucket.ForEach(func(k, v []byte) error {
			var room Room
			if err := json.Unmarshal(v, &room); err != nil {
				return fmt.Errorf("error unmarshalling room data for key %s: %w", k, err)
			}
			rooms[room.RoomID] = &room
			return nil
		})
		if err != nil {
			return err
		}

		return exitsBucket.ForEach(func(k, v []byte) error {
			var exit Exit
			if err := json.Unmarshal(v, &exit); err != nil {
				return fmt.Errorf("error unmarshalling exit data for key %s: %w", k, err)
			}

			keyParts := strings.SplitN(string(k), "_", 2)
			if len(keyParts) != 2 {
				return fmt.Errorf("invalid exit key format: %s", k)
			}
			roomID, err := strconv.ParseInt(keyParts[0], 10, 64)
			if err != nil {
				return fmt.Errorf("error parsing room ID from key %s: %w", k, err)
			}

			if room, exists := rooms[roomID]; exists {
				room.Exits[exit.Direction] = &exit
			} else {
				return fmt.Errorf("room not found for exit key %s", k)
			}
			return nil
		})
	})

	if err != nil {
		return nil, fmt.Errorf("error reading from BoltDB: %w", err)
	}

	return rooms, nil
}

func NewRoom(RoomID int64, Area string, Title string, Description string) *Room {
	return &Room{
		RoomID:      RoomID,
		Area:        Area,
		Title:       Title,
		Description: Description,
		Exits:       make(map[string]*Exit),
		Characters:  make(map[uint64]*Character),
	}
}

func (r *Room) AddExit(exit *Exit) {
	r.Exits[exit.Direction] = exit
}

func (r *Room) SendRoomMessage(message string) {

	for _, character := range r.Characters {
		character.SendMessage(message)
	}
}
