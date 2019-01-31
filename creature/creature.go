package creature

import "dnd/dice"

// Type is a type of creature
type Type struct {
	Name    string
	HitDice *dice.Roll
}

// Creature is an individual creature
type Creature struct {
	Type                      *Type
	Name                      string
	RolledHealth, DamageTaken int
}

// Create a creature of a given type and name with given hit dice
func Create(creatureType string, name string, hitDice *dice.Roll) *Creature {
	return &Creature{
		&Type{creatureType, hitDice},
		name,
		hitDice.Simulate().Sum,
		0,
	}
}
