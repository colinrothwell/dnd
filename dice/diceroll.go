package dice

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"unicode"
)

type DiceRollToken interface{}
type DDiceRollToken struct{}
type SignDiceRollToken rune
type NumberDiceRollToken int

func (_ DDiceRollToken) String() string {
	return "<DRT: d>"
}

func (specifier SignDiceRollToken) String() string {
	return fmt.Sprintf("<DRT: %s>", string(specifier))
}

type tokeniseState int

const (
	NotReadingAnything tokeniseState = iota
	ReadingNumber
)

func TokeniseDiceRollString(diceRollString string) ([]DiceRollToken, error) {
	tokenised := make([]DiceRollToken, 0)
	state := NotReadingAnything
	var builder strings.Builder

	tokeniseNumberIfNecessary := func() {
		if state == ReadingNumber {
			state = NotReadingAnything
			number, error := strconv.Atoi(builder.String())
			if error != nil {
				panic(error)
			}
			tokenised = append(tokenised, NumberDiceRollToken(number))
			builder.Reset()
		}
	}

	for _, r := range diceRollString {
		if !unicode.IsNumber(r) {
			tokeniseNumberIfNecessary()
		}
		switch r {
		case 'd', 'D':
			tokenised = append(tokenised, DDiceRollToken{})
		case '+', '-':
			tokenised = append(tokenised, SignDiceRollToken(r))
		default:
			if unicode.IsNumber(r) {
				state = ReadingNumber
				builder.WriteRune(r)
			} else if !unicode.IsSpace(r) {
				return nil, fmt.Errorf("%s is not part of a valid dice string", string(r))
			}
		}
	}
	tokeniseNumberIfNecessary()
	return tokenised, nil
}

type DiceRoll interface {
	SimulateValue() []int
}

type ActualDiceRoll struct {
	negative     bool
	count, faces int
}

func (dice ActualDiceRoll) String() string {
	if dice.negative {
		return fmt.Sprintf("- %dd%d", dice.count, dice.faces)
	} else {
		return fmt.Sprintf("%dd%d", dice.count, dice.faces)
	}
}

func (dice ActualDiceRoll) SimulateValue() []int {
	rolls := make([]int, 0)
	for i := 0; i < dice.count; i++ {
		rolls = append(rolls, 1+rand.Intn(dice.faces))
	}
	if dice.negative {
		for i, roll := range rolls {
			rolls[i] = -roll
		}
	}
	return rolls
}

type NumberDiceRoll struct {
	value int
}

func (roll NumberDiceRoll) String() string {
	if roll.value > 0 {
		return fmt.Sprintf("+ %d", roll.value)
	} else {
		return fmt.Sprintf("- %d", -roll.value)
	}
}

func (dice NumberDiceRoll) SimulateValue() []int {
	return []int{dice.value}
}

type DiceRolls []DiceRoll

// Need to be able to read
// d6
// -d6
// 2d6
// d6 - d6
// d6 + 1
// 2d6 - 2
// 3d8 + 2d6 + 2
func ParseDiceRollString(diceRollString string) (DiceRolls, error) {
	tokenisedString, error := TokeniseDiceRollString(diceRollString)
	if error != nil {
		return nil, error
	}
	// Read up to the end of the string or the first add/sub token
	wholeRoll := make([]DiceRoll, 0)
	var die []DiceRollToken
	var nextElement SignDiceRollToken
	var nextElementIsAddendSpecifier bool
	nextIsNegative := false
	startIndex := 0
	endIndex := 1
	for endIndex <= len(tokenisedString) {
		if endIndex < len(tokenisedString) {
			nextElement, nextElementIsAddendSpecifier = tokenisedString[endIndex].(SignDiceRollToken)
		}
		if nextElementIsAddendSpecifier || endIndex == len(tokenisedString) {
			die = tokenisedString[startIndex:endIndex]
			switch len(die) {
			case 1:
				value, ok := die[0].(NumberDiceRollToken)
				if !ok {
					return nil, errors.New("invalid die format")
				}
				if nextIsNegative {
					value = -value
				}
				wholeRoll = append(wholeRoll, NumberDiceRoll{int(value)})
			case 2:
				_, ok := die[0].(DDiceRollToken)
				if !ok {
					return nil, errors.New("invalid die symbol")
				}
				value, valueOk := die[1].(NumberDiceRollToken)
				if !valueOk {
					return nil, errors.New("invalid number of sides marker")
				}
				wholeRoll = append(wholeRoll, ActualDiceRoll{
					nextIsNegative,
					1,
					int(value),
				})
			case 3:
				left, leftOk := die[0].(NumberDiceRollToken)
				if !leftOk {
					return nil, errors.New("invalid number of dice")
				}
				_, dieSymbolOk := die[1].(DDiceRollToken)
				if !dieSymbolOk {
					return nil, errors.New("invalid die symbol")
				}
				right, rightOk := die[2].(NumberDiceRollToken)
				if !rightOk {
					return nil, errors.New("invalid number of sides")
				}
				wholeRoll = append(wholeRoll, ActualDiceRoll{
					nextIsNegative, int(left), int(right),
				})
			default:
				return nil, errors.New(fmt.Sprint("Invalid die ", die))
			}
			if nextElement == SignDiceRollToken('-') {
				nextIsNegative = true
			} else {
				nextIsNegative = false
			}
			startIndex = endIndex + 1
			endIndex = startIndex + 1
		} else {
			endIndex++
		}
	}
	return wholeRoll, nil
}

func (dice DiceRolls) SimulateValue() []int {
	result := make([]int, 0)
	for _, d := range dice {
		result = append(result, d.SimulateValue()...)
	}
	return result
}

type DiceRollResult struct {
	DiceRolled DiceRoll
	Results    []int
	Sum        int
}

func ReverseDiceRollResult(lst []DiceRollResult) chan DiceRollResult {
	ret := make(chan DiceRollResult)
	go func() {
		for i, _ := range lst {
			ret <- lst[len(lst)-1-i]
		}
		close(ret)
	}()
	return ret
}
