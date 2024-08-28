package core

import (
	"net"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/uuid"
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
	db    *dynamodb.DynamoDB
	Mutex sync.Mutex
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
	Name          string            `json:"name" dynamodbav:"Name"`
	CharacterList map[string]string `json:"characterList" dynamodbav:"CharacterList"`
}

type Room struct {
	RoomID      int64                    `json:"roomID" dynamodbav:"RoomID"`
	Area        string                   `json:"area" dynamodbav:"Area"`
	Title       string                   `json:"title" dynamodbav:"Title"`
	Description string                   `json:"description" dynamodbav:"Description"`
	Exits       map[string]*Exit         `json:"-"`
	Characters  map[uuid.UUID]*Character `json:"-"`
	Items       map[string]*Item         `json:"-"`
	Mutex       sync.Mutex               `json:"-"`
}

type Exit struct {
	TargetRoom int64  `json:"targetRoom" dynamodbav:"TargetRoom"`
	Visible    bool   `json:"visible" dynamodbav:"Visible"`
	Direction  string `json:"direction" dynamodbav:"Direction"`
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
	Index      string             `json:"index" dynamodbav:"Index"`
	PlayerID   string             `json:"playerID" dynamodbav:"PlayerID"`
	Name       string             `json:"name" dynamodbav:"Name"`
	Attributes map[string]float64 `json:"attributes" dynamodbav:"Attributes"`
	Abilities  map[string]float64 `json:"abilities" dynamodbav:"Abilities"`
	Essence    float64            `json:"essence" dynamodbav:"Essence"`
	Health     float64            `json:"health" dynamodbav:"Health"`
	RoomID     int64              `json:"roomID" dynamodbav:"RoomID"`
	Inventory  map[string]string  `json:"inventory" dynamodbav:"Inventory"`
}

type Archetype struct {
	Name        string             `json:"name" dynamodbav:"Name"`
	Description string             `json:"description" dynamodbav:"Description"`
	Attributes  map[string]float64 `json:"Attributes" dynamodbav:"Attributes"`
	Abilities   map[string]float64 `json:"Abilities" dynamodbav:"Abilities"`
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
	ID          string            `json:"id" dynamodbav:"ID"`
	Name        string            `json:"name" dynamodbav:"Name"`
	Description string            `json:"description" dynamodbav:"Description"`
	Mass        float64           `json:"mass" dynamodbav:"Mass"`
	Value       uint64            `json:"value" dynamodbav:"Value"`
	Stackable   bool              `json:"stackable" dynamodbav:"Stackable"`
	MaxStack    uint32            `json:"max_stack" dynamodbav:"MaxStack"`
	Quantity    uint32            `json:"quantity" dynamodbav:"Quantity"`
	Wearable    bool              `json:"wearable" dynamodbav:"Wearable"`
	WornOn      []string          `json:"worn_on" dynamodbav:"WornOn"`
	Verbs       map[string]string `json:"verbs" dynamodbav:"Verbs"`
	Overrides   map[string]string `json:"overrides" dynamodbav:"Overrides"`
	TraitMods   map[string]int8   `json:"trait_mods" dynamodbav:"TraitMods"`
	Container   bool              `json:"container" dynamodbav:"Container"`
	Contents    []string          `json:"contents" dynamodbav:"Contents"`
	IsPrototype bool              `json:"is_prototype" dynamodbav:"IsPrototype"`
	IsWorn      bool              `json:"is_worn" dynamodbav:"IsWorn"`
	CanPickUp   bool              `json:"can_pick_up" dynamodbav:"CanPickUp"`
	Metadata    map[string]string `json:"metadata" dynamodbav:"Metadata"`
}

type PrototypesData struct {
	ItemPrototypes []ItemData `json:"itemPrototypes"`
}
