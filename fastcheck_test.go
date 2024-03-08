package fastcheck

import (
	"bufio"
	"embed"
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
		if reslut == text {
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
	var text = `2 girls 1 cup,2g1c,4r5e,5h1t,5hit,a_s_s,a$$,a$$hole,rien à foutre,ssblaster,assclown,asscock,asscracker,asses,assface,assfaces,assfuck,assfucker,assfukka,assgoblin,assh0le,asshat,asshead,assho1e,ايريببخشطيزالامك`
	ret := fc.Replace(text, '⭑', skipFn)
	t.Logf("输入：%v \n 输出：%v  \n ", text, ret)
	t.Logf("Find : %v", fc.Find(text, skipFn))

}
