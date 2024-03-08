# FastCheck

FastCheck is a Go package designed to provide efficient and reliable sensitive word detection and replacement functionality. It allows users to quickly scan text for sensitive words, replace them with a specified character, and perform various operations related to sensitive word checking.

## Features

- Efficient detection of sensitive words in text.
- Replacement of sensitive words with a specified character.
- Support for case-insensitive matching.
- Whitelisting of certain words to exclude them from detection.
- Simple and intuitive API.

## Installation

You can install FastCheck using `go get`:

```bash
go get github.com/xsean2020/fastcheck
```

## Example
Below is an example demonstrating the usage and performance of FastCheck:

```go
package main

import (
    "fmt"
    "github.com/xsean2020/fastcheck"
)


func main() {
	fc := fastcheck.NewFastCheck(false)

	// Add sensitive words to the checker
	fc.AddWord("五星红旗")
	fc.AddWord("毛主席")

	// Check if a text contains sensitive words
	text := "五 星   红旗迎风飘扬，毛@主席的画像屹立在天    安门前。"
	if word, found := fc.HasWord(text, func(r rune) bool { // output:  Sensitive word found: 五星红旗
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	}); found {
		fmt.Printf("Sensitive word found: %s\n", word)
	} else {
		fmt.Println("No sensitive words found.")
	}

	// Replace sensitive words with asterisks
	newText := fc.Replace(text, '⭑', func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
	fmt.Println("Replaced text:", newText)
	// "Replaced text: ⭑ ⭑   ⭑⭑迎风飘扬，⭑@⭑⭑的画像屹立在⭑    ⭑⭑前。"
}

```


## Performance Testing
We have conducted performance testing to evaluate the efficiency of FastCheckPlus. Below are the results:
```bash
goos: darwin
goarch: arm64
pkg: github.com/xsean2020/fastcheck
BenchmarkFastCheckPlus_Replace
BenchmarkFastCheckPlus_Replace-8                   	  949407	      1271 ns/op
BenchmarkFastCheckPlus_HasWord
BenchmarkFastCheckPlus_HasWord-8                   	 2158707	       478.0 ns/op
BenchmarkFastCheckPlus_Replace_CaseInsensitive
BenchmarkFastCheckPlus_Replace_CaseInsensitive-8   	  857062	      1425 ns/op
BenchmarkFastCheckPlus_HasWord_CaseInsensitive
BenchmarkFastCheckPlus_HasWord_CaseInsensitive-8   	 1816110	       662.6 ns/op
```


