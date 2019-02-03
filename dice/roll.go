package dice

import (
	"errors"
	"fmt"
	"math/rand"
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

type FaceCountMap struct {
	Counts map[uint]uint
	Faces  []uint
}

func createFaceCountMap() FaceCountMap {
	return FaceCountMap{
		Counts: make(map[uint]uint),
		Faces:  make([]uint, 0),
	}
}

func (faceCount *FaceCountMap) add(count uint, faces uint) {
	if faceCount.Counts[faces] == 0 {
		faceCount.Faces = append(faceCount.Faces, faces)
	}
	faceCount.Counts[faces] += count
}

func (faceCount *FaceCountMap) sortFacesDescending() {
	sort.Slice(faceCount.Faces, func(i, j int) bool { return faceCount.Faces[i] > faceCount.Faces[j] })
}

func (faceCount *FaceCountMap) String() string {
	var b strings.Builder
	for i, face := range faceCount.Faces {
		if i != 0 {
			b.WriteString(" + ")
		}
		count := faceCount.Counts[face]
		if count > 1 {
			b.WriteString(strconv.FormatUint(uint64(count), 10))
		}
		b.WriteRune('d')
		b.WriteString(strconv.FormatUint(uint64(face), 10))
	}
	return b.String()
}

func (faceCount *FaceCountMap) isEmpty() bool {
	return len(faceCount.Faces) == 0
}

func (faceCount *FaceCountMap) Min() (min int) {
	for _, count := range faceCount.Counts {
		min += int(count)
	}
	return min
}

func (faceCount *FaceCountMap) Max() (max int) {
	for faces, count := range faceCount.Counts {
		max += int(faces) * int(count)
	}
	return max
}

type faceCountMapResult struct {
	rolls [][]uint
	sum   uint
}

func (faceCount *FaceCountMap) SimulateResult() *faceCountMapResult {
	faceCount.sortFacesDescending()
	var result faceCountMapResult
	result.rolls = make([][]uint, len(faceCount.Faces))
	for i, face := range faceCount.Faces {
		count := faceCount.Counts[face]
		result.rolls[i] = make([]uint, count)
		for j := uint(0); j < count; j++ {
			roll := 1 + uint(rand.Intn(int(face)))
			result.sum += roll
			result.rolls[i][j] = roll
		}
	}
	return &result
}

type Roll struct {
	Positive FaceCountMap
	Negative FaceCountMap
	Offset   int
}

func (roll *Roll) stringOffset() string {
	var b strings.Builder
	// This would be a stupid thing to do.
	offsetOnly := roll.Positive.isEmpty() && roll.Negative.isEmpty()
	if roll.Offset > 0 {
		if !offsetOnly {
			b.WriteString(" + ")
		}
		b.WriteString(strconv.Itoa(roll.Offset))
	} else if roll.Offset < 0 {
		if offsetOnly {
			b.WriteString(strconv.Itoa(roll.Offset))
		} else {
			b.WriteString(" - ")
			b.WriteString(strconv.Itoa(-roll.Offset))
		}
	}
	return b.String()
}

func (roll *Roll) String() string {
	var b strings.Builder
	b.WriteString(roll.Positive.String())
	if !roll.Negative.isEmpty() {
		if roll.Positive.isEmpty() {
			b.WriteRune('-')
		} else {
			b.WriteString(" - ")
		}
	}
	if len(roll.Negative.Faces) > 1 {
		b.WriteRune('(')
	}
	if len(roll.Negative.Faces) > 0 {
		b.WriteString(roll.Negative.String())
	}
	if len(roll.Negative.Faces) > 1 {
		b.WriteRune(')')
	}
	b.WriteString(roll.stringOffset())
	return b.String()
}

func (roll Roll) Min() int {
	return int(roll.Positive.Min()) - int(roll.Negative.Max()) + roll.Offset
}

func (roll Roll) Max() int {
	return int(roll.Positive.Max()) - int(roll.Negative.Min()) + roll.Offset
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
					rolls.Offset -= int(value)
				} else {
					rolls.Offset += int(value)
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
					rolls.Negative.add(count, faces)
				} else {
					rolls.Positive.add(count, faces)
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

type RollResult struct {
	Roll                             *Roll
	PositiveResults, NegativeResults [][]uint
	Sum                              int
}

func (roll *Roll) Simulate() RollResult {
	var result RollResult
	result.Roll = roll
	positiveResults := roll.Positive.SimulateResult()
	negativeResults := roll.Negative.SimulateResult()
	result.PositiveResults = positiveResults.rolls
	result.NegativeResults = negativeResults.rolls
	result.Sum = int(positiveResults.sum) - int(negativeResults.sum) + roll.Offset
	return result
}

func writeUint(builder *strings.Builder, x uint) {
	builder.WriteString(strconv.FormatUint(uint64(x), 10))
}

func StringFaceCountMapResults(results [][]uint) string {
	var stringsToJoin []string
	if len(results) == 1 {
		// Avoid wrapping results of single face-type in bracket unecessarily
		stringsToJoin = make([]string, len(results[0]))
		for i, roll := range results[0] {
			stringsToJoin[i] = strconv.FormatUint(uint64(roll), 10)
		}
	} else {
		stringsToJoin = make([]string, len(results))
		var faceStringBuilder strings.Builder
		for i, rollsForFace := range results {
			if len(rollsForFace) == 1 {
				stringsToJoin[i] = strconv.FormatUint(uint64(rollsForFace[0]), 10)
			} else {
				faceStringBuilder.Reset()
				faceStringBuilder.WriteRune('(')
				for j, roll := range rollsForFace {
					if j != 0 {
						faceStringBuilder.WriteString(" + ")
					}
					writeUint(&faceStringBuilder, roll)

				}
				faceStringBuilder.WriteRune(')')
				stringsToJoin[i] = faceStringBuilder.String()
			}
		}
	}
	return strings.Join(stringsToJoin, " + ")
}

func (result *RollResult) StringIndividualRolls() string {
	var b strings.Builder
	if len(result.PositiveResults) > 0 {
		b.WriteString(StringFaceCountMapResults(result.PositiveResults))
	}
	if len(result.NegativeResults) > 0 {
		b.WriteString(" - ")
		b.WriteString(StringFaceCountMapResults(result.NegativeResults))
	}
	b.WriteString(result.Roll.stringOffset())
	return b.String()
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
