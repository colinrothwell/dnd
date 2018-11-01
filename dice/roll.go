package dice

import (
	"errors"
	"fmt"
	"sort"
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

type faceCountMap struct {
	counts map[uint]uint
	faces  []uint
}

func createFaceCountMap() faceCountMap {
	return faceCountMap{
		faces: make([]uint, 0),
	}
}

func (faceCount *faceCountMap) add(count uint, faces uint) {
	if faceCount.counts[faces] == 0 {
		faceCount.faces = append(faceCount.faces, faces)
	}
	faceCount.counts[faces] += count
}

func (faceCount *faceCountMap) sortFacesDescending() {
	sort.Slice(faceCount.faces, func(i, j int) bool { return faceCount.faces[i] > faceCount.faces[j] })
}

func (faceCount *faceCountMap) String() string {
	var b strings.Builder
	faceCount.sortFacesDescending()
	for i, face := range faceCount.faces {
		if i != 0 {
			b.WriteString(" + ")
		}
		count := faceCount.counts[face]
		if count > 1 {
			b.WriteString(strconv.FormatUint(uint64(count), 10))
		}
		b.WriteRune('d')
		b.WriteString(strconv.FormatUint(uint64(face), 10))
	}
	return b.String()
}

func (faceCount *faceCountMap) isEmpty() bool {
	return len(faceCount.faces) == 0
}

func (faceCount *faceCountMap) Min() int {
	return len(faceCount.faces)
}

func (faceCount *faceCountMap) Max() (max int) {
	for faces, count := range faceCount.counts {
		max += int(faces) * int(count)
	}
	return max
}

type Roll struct {
	positive faceCountMap
	negative faceCountMap
	offset   int
}

func (roll *Roll) String() string {
	var b strings.Builder
	b.WriteString(roll.positive.String())
	if roll.positive.isEmpty() {
		b.WriteRune('-')
	} else {
		b.WriteString(" - ")
	}
	if len(roll.negative.faces) > 1 {
		b.WriteRune('(')
	}
	if len(roll.negative.faces) > 0 {
		b.WriteString(roll.negative.String())
	}
	if len(roll.negative.faces) > 1 {
		b.WriteRune(')')
	}
	if roll.offset > 0 {
		b.WriteString(" + ")
		b.WriteString(strconv.Itoa(roll.offset))
	} else if roll.offset < 0 {
		b.WriteString(" - ")
		b.WriteString(strconv.Itoa(-roll.offset))
	}
	return b.String()
}

func (roll Roll) Min() int {
	return int(roll.positive.Min()) - int(roll.negative.Max()) + roll.offset
}

func (roll Roll) Max() int {
	return int(roll.positive.Max()) - int(roll.negative.Min()) + roll.offset
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
	rolls := &Roll{createFaceCountMap(), createFaceCountMap(), 0}
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
			var count, faces uint
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
				addDie := false
			case 2:
				_, ok := die[0].(DdiceUnitToken)
				if !ok {
					return nil, errors.New("invalid die symbol")
				}
				value, valueOk := die[1].(NumberdiceUnitToken)
				if !valueOk {
					return nil, errors.New("invalid number of sides marker")
				}
				count = 1
				faces = uint(value)
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
				count = uint(left)
				faces = uint(right)
			default:
				return nil, errors.New(fmt.Sprint("Invalid die ", die))
			}
			if count != 0 {
				if nextIsNegative {
					rolls.negative.addDice(newDie)
				} else {
					rolls.positive.addDice(newDie)
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

type diceUnitResultSlice []faceCountMap

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
		make(diceUnitResultSlice, len(roll.positive.faces)),
		make(diceUnitResultSlice, len(roll.negative.faces)),
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
