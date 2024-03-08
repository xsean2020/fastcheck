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
	var runes = []rune(str)
	fc.find(runes, skip, func(idxs []uint16) bool {
		for i := range idxs {
			original[idxs[i]] = char
		}
		return false
	})
	return string(original)
}

func (fc *FastCheck) find(runes []rune, skip func(rune) bool, handle func(idxs []uint16) bool) {
	var index uint16
	var length = uint16(len(runes))
	var lastIndex = length - 1
	var wordsIndex = make([]uint16, 0, length)

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
		wordsIndex = wordsIndex[:0]
		wordsIndex = append(wordsIndex, index)
		if first.Min == 1 {
			if !fc.inWhitelist(string(begin)) {
				if handle(wordsIndex) {
					return
				}
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
			wordsIndex = append(wordsIndex, j+index)
			if j+1-ignoreCount >= first.Min {
				if first.CheckLen(j+1-ignoreCount) && letter.IsEnd {
					// 不允许复用
					var b strings.Builder
					for _, i := range wordsIndex {
						b.WriteRune(runes[i])
					}
					target := b.String()
					if _, ok := fc.hashSet[target]; ok {
						if !fc.inWhitelist(target) {
							if handle(wordsIndex) {
								return
							}
						}
					}
				}
			}
		}
		index += counter
	}
}

func (fc *FastCheck) Find(str string, skip func(r rune) bool) []string {
	if fc.ignoreCase {
		str = strings.ToUpper(str)
	}
	var all [][]uint16
	var runes = []rune(str)
	fc.find(runes, skip, func(idxs []uint16) bool {
		var cp = make([]uint16, len(idxs))
		copy(cp, idxs)
		all = append(all, cp)
		return false
	})

	if len(all) == 0 {
		return nil
	}
	var ret = make([]string, len(all))
	for i, words := range all {
		var b strings.Builder
		for i := range words {
			b.WriteRune(runes[words[i]])
		}
		ret[i] = b.String()
	}
	return ret
}

func (fc *FastCheck) HasWord(str string, skip func(rune) bool) (string, bool) {
	if fc.ignoreCase {
		str = strings.ToUpper(str)
	}
	var runes = []rune(str)
	var words []uint16
	fc.find(runes, skip, func(idxs []uint16) bool {
		words = idxs
		return true
	})

	if len(words) == 0 {
		return "", false
	}
	var b strings.Builder
	for i := range words {
		b.WriteRune(runes[words[i]])
	}
	return b.String(), true
}
