package fastcheck

import (
	"fmt"
	"math"
	"strings"
	"sync"
)

const (
	MaxStrLen  = 15
	MaxTextLen = math.MaxUint8 // 脏词长度不能超过255
)

func min(a, b uint16) uint16 {
	if a > b {
		return b
	}
	return a
}

// 利用概率 碰撞的概率不大
// 脏词库越大，越不准确,碰撞的概率会越大
type Letter struct {
	Pos    uint16 // Position of this character in a string
	Length uint16 // Length of string starting with this character
	Max    uint16 // Maximum length of string starting with this character
	Min    uint8  // Minimum length of string starting with this character
	IsEnd  uint8  // Indicates if this is the end character
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

func (l *Letter) SetMin(min uint8) {
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
	if len -= 1; len > MaxStrLen {
		len = MaxStrLen
	}
	l.Length |= uint16(1 << len)
}

func (l *Letter) CheckLen(len uint16) bool {
	if len -= 1; len > MaxStrLen {
		len = MaxStrLen
	}
	return (l.Length & uint16(1<<len)) > 0
}

type FastCheck struct {
	hashSet    map[string]struct{}
	whitelist  map[string]struct{}
	letters    []Letter
	indices    map[rune]int
	ignoreCase bool
	sync.RWMutex
}

func NewFastCheck(ignoreCase bool) *FastCheck {
	return &FastCheck{
		hashSet:    make(map[string]struct{}),
		ignoreCase: ignoreCase,
		indices:    make(map[rune]int),
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
	idx, ok := fc.indices[r]
	if !ok {
		idx = len(fc.letters)
		fc.indices[r] = idx
		fc.letters = append(fc.letters, Letter{Min: math.MaxUint8})
	}
	return &fc.letters[idx]
}

func (fc *FastCheck) letter(r rune) *Letter {
	idx, ok := fc.indices[r]
	if ok {
		return &fc.letters[idx]
	}
	return nil
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
	fc.mustLetter(runes[size-1]).IsEnd = 1
	start := fc.mustLetter(runes[0])
	start.SetMax(size)
	start.SetMin(uint8(size))
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

// Find all dirty words in the sentence
func (fc *FastCheck) Find(str string, skip func(r rune) bool) []string {
	var original = []rune(str)
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
			b.WriteRune(original[words[i]])
		}
		ret[i] = b.String()
	}
	return ret
}

// Replace the dirty words in the sentence with a specified character.
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
			first = fc.letter(runes[index])
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

		var ignoreCount uint16 // Number of ignored characters
		var counter = uint16(1)
		var minLen = uint16(first.Min)
		for j := uint16(1); j <= min(length-index-1, first.Max+ignoreCount); j++ {
			var current = runes[index+j]
			if skip != nil && skip(current) {
				ignoreCount++
				continue
			}

			letter := fc.letter(current)
			if letter == nil {
				break
			}

			if !letter.IsFirst() {
				counter++
			}

			if !letter.CheckPos(j - ignoreCount) {
				break
			}

			wordsIndex = append(wordsIndex, j+index)
			if j+1-ignoreCount >= minLen {
				if first.CheckLen(j+1-ignoreCount) && letter.IsEnd == 1 {
					var b strings.Builder
					for _, i := range wordsIndex {
						b.WriteRune(runes[i])
					}
					target := b.String()
					if _, ok := fc.hashSet[target]; ok {
						//fmt.Println("target", target, "source", string(runes[index:index+j+1]))
						if !fc.inWhitelist(target) {
							if handle(wordsIndex) {
								return
							}
						}
						counter = j + 1 // find skip this world
					}
				}
			}
		}
		index += counter
	}
}

// Check if the sentence contains dirty words
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
