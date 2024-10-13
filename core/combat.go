package core

import (
	"github.com/google/uuid"
)

// EnterCombat initializes the CombatRange map when a character enters combat
func (c *Character) EnterCombat() {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	if c.CombatRange == nil {
		c.CombatRange = make(map[uuid.UUID]int)
	}
}

// ExitCombat clears the CombatRange map when a character exits combat
func (c *Character) ExitCombat() {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	c.CombatRange = nil
}

// SetCombatRange sets the range to a target character, initializing the map if necessary
func (c *Character) SetCombatRange(target *Character, CombatRange int) {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()
	if c.CombatRange == nil {
		c.CombatRange = make(map[uuid.UUID]int)
	}
	c.CombatRange[target.ID] = CombatRange
}

// GetCombatRange gets the range to a target character, returning RangeFar if not in combat
func (c *Character) GetCombatRange(target *Character) int {
	if c.CombatRange == nil {
		return 0 // RangeFar
	}
	if CombatRange, exists := c.CombatRange[target.ID]; exists {
		return CombatRange
	}
	return 0 // RangeFar
}

// IsInCombat checks if the character is currently in combat
func (c *Character) IsInCombat() bool {
	return c.CombatRange != nil && len(c.CombatRange) > 0
}

// CanEscape checks if the character can escape from combat
// Returns true if no other characters are at melee range
func (c *Character) CanEscape() bool {
	c.Mutex.Lock()
	defer c.Mutex.Unlock()

	// If not in combat, can always escape
	if c.CombatRange == nil {
		return true
	}

	// Check if any character is at melee range
	for _, distance := range c.CombatRange {
		if distance == 2 { // RangeMelee
			return false
		}
	}
	// No characters at melee range, can escape
	return true
}
