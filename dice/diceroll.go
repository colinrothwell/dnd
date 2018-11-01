package dice

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"unicode"
)

type diceUnitToken interface{}
type DdiceUnitToken struct{}
type SigndiceUnitToken rune
type NumberdiceUnitToken uint

func (_ DdiceUnitToken) String() string {
	return "<DRT: d>"
}

func (specifier SigndiceUnitToken) String() string {
	return fmt.Sprintf("<DRT: %s>", string(specifier))
}

type tokeniseState int

const (
	NotReadingAnything tokeniseState = iota
	ReadingNumber
)

func TokenisediceUnitString(diceRollString string) ([]diceUnitToken, error) {
	tokenised := make([]diceUnitToken, 0)
	state := NotReadingAnything
	var builder strings.Builder

	tokeniseNumberIfNecessary := func() {
		if state == ReadingNumber {
			state = NotReadingAnything
			number, error := strconv.Atoi(builder.String())
			if error != nil {
				panic(error)
			}
			tokenised = append(tokenised, NumberdiceUnitToken(number))
			builder.Reset()
		}
	}

	for _, r := range diceRollString {
		if !unicode.IsNumber(r) {
			tokeniseNumberIfNecessary()
		}
		switch r {
		case 'd', 'D':
			tokenised = append(tokenised, DdiceUnitToken{})
		case '+', '-':
			tokenised = append(tokenised, SigndiceUnitToken(r))
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

type diceUnit struct {
	Count, Faces uint
}

func (dice *diceUnit) String() string {
	return fmt.Sprintf("%dd%d", dice.Count, dice.Faces)
}

type diceUnitResult []uint

func (dice *diceUnit) SimulateValue() diceUnitResult {
	rolls := make([]uint, 0)
	for i := uint(0); i < dice.Count; i++ {
		rolls = append(rolls, 1+uint(rand.Intn(int(dice.Faces))))
	}
	return rolls
}

func (result diceUnitResult) Sum() (sum uint) {
	for _, num := range result {
		sum += num
	}
	return sum
}

func (result diceUnitResult) String() string {
	stringSlice := make([]string, len(result))
	for i, result := range result {
		stringSlice[i] = strconv.FormatUint(uint64(result), 10)
	}
	return strings.Join(stringSlice, " + ")
}

func (dice diceUnit) Min() uint {
	return dice.Count
}

func (dice diceUnit) Max() uint {
	return dice.Count * dice.Faces
}

type diceUnitSlice []diceUnit

type Roll struct {
	positive diceUnitSlice
	negative diceUnitSlice
	offset   int
}

func (units diceUnitSlice) String() string {
	result := ""
	unitStrings := make([]string, len(units))
	for i, unit := range units {
		unitStrings[i] = unit.String()
	}
	result += strings.Join(unitStrings, " + ")
	return result
}

func (roll *Roll) String() string {
	result := ""
	result += roll.positive.String()
	if len(roll.positive) == 0 {
		result += "-"
	} else {
		result += " - "
	}
	negativeString := roll.negative.String()
	if len(roll.negative) > 1 {
		negativeString = fmt.Sprintf("(%s)", negativeString)
	}
	if len(roll.negative) > 0 {
		result += negativeString
	}
	if roll.offset > 0 {
		result += " + "
		result += strconv.Itoa(roll.offset)
	} else if roll.offset < 0 {
		result += " - "
		result += strconv.Itoa(-roll.offset)
	}
	return result
}

func diceUnitSliceMin(units []diceUnit) (sum uint) {
	for _, unit := range units {
		sum += unit.Min()
	}
	return sum
}

func diceUnitSliceMax(units []diceUnit) (sum uint) {
	for _, unit := range units {
		sum += unit.Max()
	}
	return sum
}

func (roll Roll) Min() int {
	return int(diceUnitSliceMin(roll.positive)) - int(diceUnitSliceMax(roll.negative)) + roll.offset
}

func (roll Roll) Max() int {
	return int(diceUnitSliceMax(roll.positive)) - int(diceUnitSliceMin(roll.negative)) + roll.offset
}

// Need to be able to read
// d6
// -d6
// 2d6
// d6 - d6
// d6 + 1
// 2d6 - 2
// 3d8 + 2d6 + 2
func ParseRollString(diceRollString string) (*Roll, error) {
	tokenisedString, error := TokenisediceUnitString(diceRollString)
	if error != nil {
		return nil, error
	}
	// Read up to the end of the string or the first add/sub token
	rolls := &Roll{
		make([]diceUnit, 0),
		make([]diceUnit, 0),
		0,
	}
	var die []diceUnitToken
	var nextElement SigndiceUnitToken
	var nextElementIsAddendSpecifier bool
	nextIsNegative := false
	startIndex := 0
	endIndex := 1
	for endIndex <= len(tokenisedString) {
		if endIndex < len(tokenisedString) {
			nextElement, nextElementIsAddendSpecifier = tokenisedString[endIndex].(SigndiceUnitToken)
		}
		if nextElementIsAddendSpecifier || endIndex == len(tokenisedString) {
			die = tokenisedString[startIndex:endIndex]
			var newDie *diceUnit
			switch len(die) {
			case 1:
				value, ok := die[0].(NumberdiceUnitToken)
				if !ok {
					return nil, errors.New("invalid die format")
				}
				if nextIsNegative {
					rolls.offset -= int(value)
				} else {
					rolls.offset += int(value)
				}
			case 2:
				_, ok := die[0].(DdiceUnitToken)
				if !ok {
					return nil, errors.New("invalid die symbol")
				}
				value, valueOk := die[1].(NumberdiceUnitToken)
				if !valueOk {
					return nil, errors.New("invalid number of sides marker")
				}
				newDie = &diceUnit{1, uint(value)}
			case 3:
				left, leftOk := die[0].(NumberdiceUnitToken)
				if !leftOk {
					return nil, errors.New("invalid number of dice")
				}
				_, dieSymbolOk := die[1].(DdiceUnitToken)
				if !dieSymbolOk {
					return nil, errors.New("invalid die symbol")
				}
				right, rightOk := die[2].(NumberdiceUnitToken)
				if !rightOk {
					return nil, errors.New("invalid number of sides")
				}
				newDie = &diceUnit{uint(left), uint(right)}
			default:
				return nil, errors.New(fmt.Sprint("Invalid die ", die))
			}
			if newDie != nil {
				if nextIsNegative {
					rolls.negative = append(rolls.negative, *newDie)
				} else {
					rolls.positive = append(rolls.positive, *newDie)
				}
			}
			if nextElement == SigndiceUnitToken('-') {
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
	return rolls, nil
}

type diceUnitResultSlice []diceUnitResult

func (resultSlice diceUnitResultSlice) String() string {
	stringSlice := make([]string, len(resultSlice))
	for i, result := range resultSlice {
		stringSlice[i] = result.String()
	}
	return strings.Join(stringSlice, " + ")
}

type RollResult struct {
	Roll                             *Roll
	PositiveResults, NegativeResults diceUnitResultSlice
	Sum                              int
}

func (roll *Roll) Simulate() RollResult {
	result := RollResult{
		roll,
		make(diceUnitResultSlice, len(roll.positive)),
		make(diceUnitResultSlice, len(roll.negative)),
		0,
	}
	for i, unit := range roll.positive {
		unitResult := unit.SimulateValue()
		result.PositiveResults[i] = unitResult
		result.Sum += int(unitResult.Sum())
	}
	for i, unit := range roll.negative {
		unitResult := unit.SimulateValue()
		result.NegativeResults[i] = unitResult
		result.Sum -= int(unitResult.Sum())
	}
	result.Sum += roll.offset
	return result
}

func (result *RollResult) StringIndividualRolls() string {
	return result.PositiveResults.String() + " - " +
		result.NegativeResults.String() + " + " + strconv.Itoa(result.Roll.offset)
}

func ReverseRollResultSlice(slice []RollResult) chan *RollResult {
	ret := make(chan *RollResult)
	go func() {
		for i := range slice {
			ret <- &slice[len(slice)-1-i]
		}
		close(ret)
	}()
	return ret
}
