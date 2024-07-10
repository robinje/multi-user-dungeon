package core

import (
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/crypto/ssh"
)

// The Index struct is to be depricated in favor of UUIDs
type Index struct {
	IndexID uint64
	mu      sync.Mutex
}

type Configuration struct {
	Port           uint16  `json:"Port"`
	UserPoolID     string  `json:"UserPoolId"`
	ClientSecret   string  `json:"UserPoolClientSecret"`
	UserPoolRegion string  `json:"UserPoolRegion"`
	ClientID       string  `json:"UserPoolClientId"`
	DataFile       string  `json:"DataFile"`
	Balance        float64 `json:"Balance"`
	AutoSave       uint16  `json:"AutoSave"`
	Essence        uint16  `json:"StartingEssence"`
	Health         uint16  `json:"StartingHealth"`
}

type KeyPair struct {
	db    *bolt.DB
	file  string
	Mutex sync.Mutex // Mutex to synchronize write access
}

type Server struct {
	Port            uint16
	Listener        net.Listener
	SSHConfig       *ssh.ServerConfig
	PlayerCount     uint64
	Config          Configuration
	StartTime       time.Time
	Rooms           map[int64]*Room
	Database        *KeyPair
	PlayerIndex     *Index
	CharacterExists map[string]bool
	Characters      map[uuid.UUID]*Character
	Balance         float64
	AutoSave        uint16
	Archetypes      *ArchetypesData
	Health          uint16
	Essence         uint16
	Items           map[uint64]*Item
	ItemPrototypes  map[uint64]*Item
	Mutex           sync.Mutex
}

type Player struct {
	PlayerID      string
	Index         uint64
	Name          string
	ToPlayer      chan string
	FromPlayer    chan string
	PlayerError   chan error
	Echo          bool
	Prompt        string
	Connection    ssh.Channel
	Server        *Server
	ConsoleWidth  int
	ConsoleHeight int
	CharacterList map[string]uuid.UUID
	Character     *Character
	LoginTime     time.Time
	PasswordHash  string
	Mutex         sync.Mutex
}

type PlayerData struct {
	Name          string
	CharacterList map[string]uint64
}

type Room struct {
	RoomID      int64
	Area        string
	Title       string
	Description string
	Exits       map[string]*Exit
	Characters  map[uuid.UUID]*Character
	Items       map[string]*Item
	Mutex       sync.Mutex
}

type Exit struct {
	ExitID     int64
	TargetRoom int64
	Visible    bool
	Direction  string
}

type Character struct {
	ID         uuid.UUID
	Player     *Player
	Name       string
	Attributes map[string]float64
	Abilities  map[string]float64
	Essence    float64
	Health     float64
	Room       *Room
	Inventory  map[string]*Item
	Server     *Server
	Mutex      sync.Mutex
}

// CharacterData for unmarshalling character.
type CharacterData struct {
	Index      string             `json:"index"`
	PlayerID   string             `json:"playerID"`
	Name       string             `json:"name"`
	Attributes map[string]float64 `json:"attributes"`
	Abilities  map[string]float64 `json:"abilities"`
	Essence    float64            `json:"essence"`
	Health     float64            `json:"health"`
	RoomID     int64              `json:"roomID"`
	Inventory  map[string]string  `json:"inventory"` // Changed to map[string]string for UUIDs
}

type Archetype struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Attributes  map[string]float64 `json:"Attributes"`
	Abilities   map[string]float64 `json:"Abilities"`
}

type ArchetypesData struct {
	Archetypes map[string]Archetype `json:"archetypes"`
}

type Item struct {
	ID          uuid.UUID
	Name        string
	Description string
	Mass        float64
	Value       uint64
	Stackable   bool
	MaxStack    uint32
	Quantity    uint32
	Wearable    bool
	WornOn      []string
	Verbs       map[string]string
	Overrides   map[string]string
	TraitMods   map[string]int8
	Container   bool
	Contents    []*Item
	IsPrototype bool
	IsWorn      bool
	CanPickUp   bool
	Metadata    map[string]string
	Mutex       sync.Mutex
}

type ItemData struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Mass        float64           `json:"mass"`
	Value       uint64            `json:"value"`
	Stackable   bool              `json:"stackable"`
	MaxStack    uint32            `json:"max_stack"`
	Quantity    uint32            `json:"quantity"`
	Wearable    bool              `json:"wearable"`
	WornOn      []string          `json:"worn_on"`
	Verbs       map[string]string `json:"verbs"`
	Overrides   map[string]string `json:"overrides"`
	TraitMods   map[string]int8   `json:"trait_mods"`
	Container   bool              `json:"container"`
	Contents    []string          `json:"contents"`
	IsPrototype bool              `json:"is_prototype"`
	IsWorn      bool              `json:"is_worn"`
	CanPickUp   bool              `json:"can_pick_up"`
	Metadata    map[string]string `json:"metadata"`
}

type PrototypesData struct {
	ItemPrototypes []ItemData `json:"itemPrototypes"`
}
