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
type NumberDiceRollToken uint

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
	SimulateValue() []uint
	String() string
	NeedsBrackets() bool
	Min() uint
	Max() uint
}

type ActualDiceRoll struct {
	count, faces uint
}

func (dice ActualDiceRoll) NeedsBrackets() bool {
	return dice.count != 1
}

func (dice ActualDiceRoll) String() string {
	return fmt.Sprintf("%dd%d", dice.count, dice.faces)
}

func (dice ActualDiceRoll) SimulateValue() []uint {
	rolls := make([]uint, 0)
	for i := uint(0); i < dice.count; i++ {
		rolls = append(rolls, 1+uint(rand.Intn(int(dice.faces))))
	}
	return rolls
}

func (dice ActualDiceRoll) Min() uint {
	return dice.count
}

func (dice ActualDiceRoll) Max() uint {
	return dice.count * dice.faces
}

type NumberDiceRoll uint

func (number NumberDiceRoll) NeedsBrackets() bool {
	return false
}

func (number NumberDiceRoll) String() string {
	return fmt.Sprintf("%d", number)
}

func (number NumberDiceRoll) SimulateValue() []uint {
	return []uint{uint(number)}
}

func (number NumberDiceRoll) Min() uint {
	return uint(number)
}

func (number NumberDiceRoll) Max() uint {
	return uint(number)
}

type DiceRolls struct {
	subtracted []bool
	rolls      []DiceRoll
}

func (diceRolls DiceRolls) String() string {
	result := ""
	for i, roll := range diceRolls.rolls {
		if i == 0 {
			if diceRolls.subtracted[i] {
				result += "-"
			}
			result += roll.String()
		} else {
			if diceRolls.subtracted[i] {
				result += " - "
			} else {
				result += " + "
			}
			result += roll.String()
		}
	}
	return result
}

func (diceRolls DiceRolls) Min() uint {
	min := uint(0)
	for i, roll := range diceRolls.rolls {
		if diceRolls.subtracted[i] {
			min -= roll.Max()
		} else {
			min += roll.Min()
		}
	}
	return min
}

func (diceRolls DiceRolls) Max() uint {
	max := uint(0)
	for i, roll := range diceRolls.rolls {
		if diceRolls.subtracted[i] {
			max -= roll.Min()
		} else {
			max += roll.Max()
		}
	}
	return max
}

// Need to be able to read
// d6
// -d6
// 2d6
// d6 - d6
// d6 + 1
// 2d6 - 2
// 3d8 + 2d6 + 2
func ParseDiceRollString(diceRollString string) (*DiceRolls, error) {
	tokenisedString, error := TokeniseDiceRollString(diceRollString)
	if error != nil {
		return nil, error
	}
	// Read up to the end of the string or the first add/sub token
	wholeSubtracted := make([]bool, 0)
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
			wholeSubtracted = append(wholeSubtracted, nextIsNegative)
			switch len(die) {
			case 1:
				value, ok := die[0].(NumberDiceRollToken)
				if !ok {
					return nil, errors.New("invalid die format")
				}
				wholeRoll = append(wholeRoll, NumberDiceRoll(value))
			case 2:
				_, ok := die[0].(DDiceRollToken)
				if !ok {
					return nil, errors.New("invalid die symbol")
				}
				value, valueOk := die[1].(NumberDiceRollToken)
				if !valueOk {
					return nil, errors.New("invalid number of sides marker")
				}
				wholeRoll = append(wholeRoll, ActualDiceRoll{1, uint(value)})
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
				wholeRoll = append(wholeRoll, ActualDiceRoll{uint(left), uint(right)})
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
	return &DiceRolls{wholeSubtracted, wholeRoll}, nil
}

type DiceRollResult struct {
	DiceRolled *DiceRolls
	Results    [][]uint
	Sum        int
}

func (diceRollResult DiceRollResult) StringSimulatedResults() string {
	result := ""
	for i, subRoll := range diceRollResult.Results {
		subResult := ""
		includeBrackets := len(diceRollResult.Results) != 1 && diceRollResult.DiceRolled.rolls[i].NeedsBrackets()
		if includeBrackets {
			subResult = "("
		}
		for j, roll := range subRoll {
			if j == 0 {
				subResult += fmt.Sprint(roll)
			} else {
				subResult += fmt.Sprintf(" + %d", roll)
			}
		}
		if includeBrackets {
			subResult += ")"
		}
		subtracted := diceRollResult.DiceRolled.subtracted
		if i == 0 {
			if subtracted[i] {
				result += "-"
			}
		} else {
			if subtracted[i] {
				result += " - "
			} else {
				result += " + "
			}
		}
		result += subResult
	}
	return result
}

func (diceRolls *DiceRolls) SimulateDiceRolls() DiceRollResult {
	sum := 0
	results := make([][]uint, len(diceRolls.rolls))
	for i, roll := range diceRolls.rolls {
		results[i] = roll.SimulateValue()
		addend := 0
		for _, individualRoll := range results[i] {
			addend += int(individualRoll)
		}
		if diceRolls.subtracted[i] {
			addend = -addend
		}
		sum += addend
	}
	return DiceRollResult{diceRolls, results, sum}
}

func ReverseDiceRollResultSlice(lst []DiceRollResult) chan DiceRollResult {
	ret := make(chan DiceRollResult)
	go func() {
		for i := range lst {
			ret <- lst[len(lst)-1-i]
		}
		close(ret)
	}()
	return ret
}
