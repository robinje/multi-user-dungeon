package core

import (
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
)

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

type Server struct {
	Port            uint16
	Listener        net.Listener
	SSHConfig       *ssh.ServerConfig
	PlayerCount     uint64
	Mutex           sync.Mutex
	Config          Configuration
	StartTime       time.Time
	Rooms           map[int64]*Room
	Database        *KeyPair
	PlayerIndex     *Index
	CharacterExists map[string]bool
	Characters      map[string]*Character
	Balance         float64
	AutoSave        uint16
	Archetypes      *ArchetypesData
	Health          uint16
	Essence         uint16
	Items           map[uint64]*Item
	ItemPrototypes  map[uint64]*Item
}

type Room struct {
	RoomID      int64
	Area        string
	Title       string
	Description string
	Exits       map[string]*Exit
	Characters  map[uint64]*Character
	Mutex       sync.Mutex
	Items       map[string]*Item
}

type Exit struct {
	ExitID     int64
	TargetRoom int64
	Visible    bool
	Direction  string
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

type Character struct {
	Index      uint64
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

type Player struct {
	PlayerID      string
	Index         uint64
	Name          string
	ToPlayer      chan string
	FromPlayer    chan string
	PlayerError   chan error
	Echo          bool
	Prompt        string
	Connection    net.Conn
	Server        *Server
	ConsoleWidth  int
	ConsoleHeight int
	CharacterList map[string]uint64
	Character     *Character
	LoginTime     time.Time
	PasswordHash  string
}
