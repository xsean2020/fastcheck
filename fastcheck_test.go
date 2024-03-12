package fastcheck

import (
	"bufio"
	"embed"
	"fmt"
	"log"
	"testing"
	"time"
	"unicode"

	_ "embed"
)

var fc = NewFastCheck(true)

//go:embed dirty.txt
var f embed.FS

func init() {
	file, _ := f.Open("dirty.txt")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var text = scanner.Text()
		fc.AddWord(text)
	}

	fc.AddWord("2 girls 1 cup")
	fc.AddWord("激情小姐")

	var min uint8 = 255
	var max uint16 = 0
	for _, v := range fc.letters {

		if min > v.Min {
			min = v.Min
		}

		if v.Max > max {
			max = v.Max
		}

	}
	fmt.Println(min, max)
	// fc.AddWhitelist("fat")
}

func BenchmarkFastCheckPlus_Replace(b *testing.B) {
	fc := NewFastCheck(false)
	fc.AddWord("badword")
	fc.AddWord("anotherbadword")
	text := "This is a test text with badword and anotherbadword."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fc.Replace(text, '*', nil)
	}
}

func BenchmarkFastCheckPlus_HasWord(b *testing.B) {
	fc := NewFastCheck(false)
	fc.AddWord("badword")
	fc.AddWord("anotherbadword")
	text := "This is a test text with badword and anotherbadword."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = fc.HasWord(text, nil)
	}
}

func BenchmarkFastCheckPlus_Replace_CaseInsensitive(b *testing.B) {
	fc := NewFastCheck(true)
	fc.AddWord("badword")
	fc.AddWord("anotherbadword")
	text := "This is a test text with BadWord and AnotherBadWord."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fc.Replace(text, '*', nil)
	}
}

func BenchmarkFastCheckPlus_HasWord_CaseInsensitive(b *testing.B) {
	fc := NewFastCheck(true)
	fc.AddWord("badword")
	fc.AddWord("anotherbadword")
	text := "This is a test text with BadWord and AnotherBadWord."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = fc.HasWord(text, nil)
	}
}

func TestValid(t *testing.T) {
	file, err := f.Open("dirty.txt")
	if err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(file)
	check := fc
	for scanner.Scan() {
		var text = scanner.Text()
		if len([]rune(text)) == 1 {
			continue
		}

		reslut := check.Replace(text, '*', nil)
		if reslut != "fat" && reslut == text {
			log.Fatal("replace error", text)
		}

		if _, ok := check.HasWord(text, nil); !ok {
			log.Fatal("check error", text)
		}
	}
}

func Test100w(t *testing.T) {
	var words = []string{"你好啊, fuck you ! 这里是严格的脏词匹配", "～你就是个,垃.圾 哈 哈 哈 哈 @  , 这是一个测试字符串，里面含有中文符号。|||"}
	now := time.Now()
	for i := 0; i < 1000000; i++ {
		fc.Replace(words[0], '⛤', skipFn)
	}

	t.Log(time.Since(now))
}

var skipFn = func(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsPunct(r)
}

func TestSkip(t *testing.T) {
	var str = "a$$"
	for _, r := range []rune(str) {
		t.Log(skipFn(r))
	}
}

func TestRelace(t *testing.T) {
	var text = `
Bad Words Types & Meaning
It’s important to mention that there are many types of bad words. Our full list of bad words includes all the following types:
Curse Words – Profane or obscene words, especially as used in anger or for emphasis.
Insult Words – Words used to treat with insolence, indignity, or contempt, also to affect offensively or damagingly.
Offensive Words – Words that arouse resentment, annoyance, or anger.
Dirty Words – A vulgar or taboo word or any word, name, or concept considered reprehensible.
Rude Words – Discourteous or impolite words, used especially in a deliberate way.
Sexual Words – Words related to male and female, mother, father, sister, wife, lesbians, homosexuals, people, animals, intersex organisms, and their body parts.
Vulgar Words – Words that are characterized by ignorance of or lack of good breeding or taste.
Obscene Words – Offends you because it relates to sex or violence in a way that you think is unpleasant and shocking.
Naughty Words – Means disobedient, mischievous, or generally misbehaving, particularly when applied to children.
Inappropriate Words – Words not useful or suitable for a particular situation or purpose.
Our list has been tested by thousands of our visitors! All have confirmed that our full list works perfectly. Moreover,  they did not have any banning by Google or by any other authority.
来直播间在线观看激情小姐姐
	`
	ret := fc.Replace(text, '⭑', nil)
	t.Logf("输入：%v \n 输出：%v  \n ", text, ret)
	t.Logf("Find : %v", fc.Find(text, nil))

}
