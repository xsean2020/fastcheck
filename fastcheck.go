package fastcheck

import (
	"fmt"
	"math"
	"strings"
	"sync"
)

const (
	MaxStrLen  = 15
	MaxTextLen = math.MaxUint16
)

func min(a, b uint16) uint16 {
	if a > b {
		return b
	}
	return a
}

type Letter struct {
	Length uint16 // Length of string starting with this character
	Pos    uint16 // Position of this character in a string
	Max    uint16 // Maximum length of string starting with this character
	Min    uint16 // Minimum length of string starting with this character
	IsEnd  bool   // Indicates if this is the end character
}

func (l *Letter) String() string {
	return fmt.Sprintf("%b %v %v %v", l.Pos, l.Max, l.Min, l.IsEnd)
}

func (l *Letter) IsFirst() bool {
	if l == nil {
		return false
	}
	return l.Pos&1 == 1
}

func (l *Letter) SetMin(min uint16) {
	if l.Min < min {
		return
	}
	l.Min = min
}

func (l *Letter) SetMax(max uint16) {
	if l.Max > max {
		return
	}
	l.Max = max
}

func (l *Letter) CheckPos(pos uint16) bool {
	if pos > MaxStrLen {
		pos = MaxStrLen
	}
	return (l.Pos & uint16(1<<pos)) > 0
}

func (l *Letter) SetPos(pos int) {
	if pos > MaxStrLen {
		pos = MaxStrLen
	}
	l.Pos |= uint16(1 << pos)
}

func (l *Letter) SetLen(len uint16) {
	len -= 1
	if len > MaxStrLen {
		len = MaxStrLen
	}
	l.Length |= uint16(1 << len)
}

func (l *Letter) CheckLen(len uint16) bool {
	len -= 1
	if len > MaxStrLen {
		len = MaxStrLen
	}
	return (l.Length & uint16(1<<len)) > 0
}

type FastCheck struct {
	hashSet    map[string]struct{}
	whitelist  map[string]struct{}
	letters    map[rune]*Letter
	ignoreCase bool
	sync.RWMutex
}

func NewFastCheck(ignoreCase bool) *FastCheck {
	return &FastCheck{
		hashSet:    make(map[string]struct{}),
		ignoreCase: ignoreCase,
		letters:    make(map[rune]*Letter),
		whitelist:  make(map[string]struct{}),
	}
}

func (fc *FastCheck) AddWhitelist(words ...string) {
	fc.Lock()
	defer fc.Unlock()
	for _, word := range words {
		if fc.ignoreCase {
			word = strings.ToUpper(word)
		}
		fc.whitelist[word] = struct{}{}
	}
}

func (fc *FastCheck) mustLetter(r rune) *Letter {
	letter, ok := fc.letters[r]
	if ok {
		return letter
	}
	letter = new(Letter)
	letter.Min = math.MaxUint16
	fc.letters[r] = letter
	return letter
}

func (fc *FastCheck) AddWord(text string) bool {
	if len(text) == 0 {
		return false
	}
	var runes []rune
	if fc.ignoreCase {
		text = strings.ToUpper(text)
	}

	if len(text) > MaxTextLen {
		panic("text too long")
	}

	fc.Lock()
	defer fc.Unlock()

	if _, ok := fc.hashSet[text]; ok {
		return false
	}

	runes = []rune(text)
	size := uint16(len(runes))
	fc.mustLetter(runes[size-1]).IsEnd = true
	start := fc.mustLetter(runes[0])
	start.SetMax(size)
	start.SetMin(size)
	start.SetLen(size)
	for i, r := range runes {
		fc.mustLetter(r).SetPos(i)
	}
	fc.hashSet[text] = struct{}{}
	return true
}

func (fc *FastCheck) inWhitelist(word string) bool {
	_, ok := fc.whitelist[word]
	return ok
}

func (fc *FastCheck) Replace(str string, char rune, skip func(rune) bool) string {

	var original = []rune(str) // Original characters
	if fc.ignoreCase {
		str = strings.ToUpper(str)
	}
	var index uint16
	var runes = []rune(str)
	var length = uint16(len(runes))
	var lastIndex = length - 1
	var replacedIndex uint16 // Index of the last replaced character

	fc.RLock()
	defer fc.RUnlock()

	for index < length {
		var first *Letter
		for index < lastIndex {
			if skip != nil && skip(runes[index]) {
				index++
				continue
			}

			first = fc.letters[runes[index]]
			if first.IsFirst() {
				break
			}
			index++
		}

		if first == nil {
			break
		}

		var begin = runes[index]
		if first.Min == 1 {
			if !fc.inWhitelist(string(begin)) {
				original[index] = char
			}
			index++
			continue
		}

		var stopCounter bool
		var ignoreCount uint16 // Number of ignored characters
		var counter = uint16(1)
		if replacedIndex < index {
			replacedIndex = index
		}

		for j := uint16(1); j <= min(length-index-1, first.Max+ignoreCount); j++ {
			var current = runes[index+j]
			if skip != nil && skip(current) {
				if !stopCounter {
					counter++
				}
				ignoreCount++
				continue
			}

			letter := fc.letters[current]
			if letter == nil {
				break
			}

			if stopCounter = stopCounter || letter.IsFirst(); !stopCounter {
				counter++
			}

			if !letter.CheckPos(j - ignoreCount) {
				break
			}

			if j+1-ignoreCount >= first.Min {
				if first.CheckLen(j+1-ignoreCount) && letter.IsEnd {
					var target string
					if ignoreCount > 0 {
						var tmps = make([]rune, 0, j+1)
						for i := index; i < index+j+1; i++ {
							if skip(runes[i]) {
								continue
							}
							tmps = append(tmps, runes[i])
						}
						target = string(tmps)
					} else {
						target = string(runes[index : index+j+1])
					}

					if _, ok := fc.hashSet[target]; ok {
						if !fc.inWhitelist(target) {
							for ; replacedIndex < index+j+1; replacedIndex++ {
								if ignoreCount > 0 && skip(runes[replacedIndex]) {
									continue
								}
								original[replacedIndex] = char
							}
						}
					}
				}
			}
		}
		index += counter
	}
	return string(original)
}

func (fc *FastCheck) HasWord(str string, skip func(rune) bool) (string, bool) {
	if fc.ignoreCase {
		str = strings.ToUpper(str)
	}
	var index uint16
	var runes = []rune(str)
	var length = uint16(len(runes))
	var lastIndex = length - 1

	fc.RLock()
	defer fc.RUnlock()

	for index < length {
		var first *Letter
		for index < lastIndex {
			if skip != nil && skip(runes[index]) {
				index++
				continue
			}
			first = fc.letters[runes[index]]
			if first.IsFirst() {
				break
			}
			index++
		}

		if first == nil {
			break
		}

		var begin = runes[index]
		if first.Min == 1 {
			word := string(begin)
			if !fc.inWhitelist(word) {
				return word, true
			}
			index++
			continue
		}

		var stopCounter bool
		var ignoreCount uint16 // Number of ignored characters
		var counter = uint16(1)

		for j := uint16(1); j <= min(length-index-1, first.Max+ignoreCount); j++ {
			var current = runes[index+j]
			if skip != nil && skip(current) {
				if !stopCounter {
					counter++
				}
				ignoreCount++
				continue
			}

			letter := fc.letters[current]
			if letter == nil {
				break
			}

			if stopCounter = stopCounter || letter.IsFirst(); !stopCounter {
				counter++
			}

			if !letter.CheckPos(j - ignoreCount) {
				break
			}

			if j+1-ignoreCount >= first.Min {
				if first.CheckLen(j+1-ignoreCount) && letter.IsEnd {
					var target string
					if ignoreCount > 0 {
						var tmps = make([]rune, 0, j+1)
						for i := index; i < index+j+1; i++ {
							if skip(runes[i]) {
								continue
							}
							tmps = append(tmps, runes[i])
						}
						target = string(tmps)
					} else {
						target = string(runes[index : index+j+1])
					}

					if _, ok := fc.hashSet[target]; ok {
						if !fc.inWhitelist(target) {
							return target, true
						}
					}
				}
			}
		}
		index += counter
	}
	return "", false
}
