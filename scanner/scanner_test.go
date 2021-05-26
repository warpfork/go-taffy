package scanner

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestTaffy(t *testing.T) {
	f, err := os.Open("../taffy.taf")
	if err != nil {
		panic(err)
	}
	t.Run("detects-first-section", func(t *testing.T) {
		f.Seek(0, 0)
		scanner := NewScanner(f)
		tok, err := scanner.Scan()
		qt.Check(t, tok, qt.DeepEquals, Token{Title: "comment"})
		qt.Check(t, err, qt.IsNil)
	})
	t.Run("detects-all-sections", func(t *testing.T) {
		f.Seek(0, 0)
		scanner := NewScanner(f)
		titles, sectionCount := scanTitles(t, scanner)
		expectedTitles := []string{
			"comment",
			"zerobytes/fixture",
			"zerobytes/length",
			"oneline-stripped/fixture",
			"oneline-stripped/length",
			"twoline-stripped/fixture",
			"twoline-stripped/length",
			"basic-content/fixture",
			"basic-content/comment",
			"basic-content/length",
			"multiline/fixture",
			"multiline/comment",
			"multiline/length",
			"subtle-body/fixture",
			"subtle-body/comment",
			"subtle-body/length",
		}
		qt.Check(t, titles, qt.DeepEquals, expectedTitles)
		t.Run("detects-all-section-content", func(t *testing.T) {
			// There's one content hunk for each section.
			qt.Check(t, sectionCount, qt.Equals, len(expectedTitles)*2)
		})
	})
	t.Run("testcases", func(t *testing.T) {
		f.Seek(0, 0)
		scanner := NewScanner(f)
		runTestcases(t, scanner)
	})
}

func scanTitles(t *testing.T, scanner *scanner) (titles []string, sectionCount int) {
	for {
		tok, err := scanner.Scan()
		if err != nil {
			qt.Check(t, err, qt.Equals, io.EOF)
			break
		}
		sectionCount++
		if tok.IsSectionHeader() {
			titles = append(titles, tok.Title)
		}
	}
	return
}

func runTestcases(t *testing.T, scanner *scanner) {
	// We're going to parse the taffy.taf file, but assume a few things for simplicity and linearity:
	//  - We identify a fixture by the title pattern "{title}/fixture";
	//  - We're going to skip over anything ending in "comment";
	//  - And "{title}/length" had better follow shortly.
	//  - As soon as we've got the "{title}/length" section, we can test.
	// ... in other words, if the taf file was in a weird order, this test wouldn't work right.
	// This isn't because of limitations of taffy files, and we wouldn't recommend others write tests like this;
	//  it's because we're trying to only test the scanner at this moment, and it's very linear.
	var title string
	var fixtureNext bool
	var fixtureRaw []byte
	var lengthNext bool

	for {
		tok, err := scanner.Scan()
		if err != nil {
			qt.Check(t, err, qt.Equals, io.EOF)
			break
		}
		if tok.IsSectionHeader() {
			fixtureNext = false
			lengthNext = false
			if strings.HasSuffix(tok.Title, "comment") {
				continue
			}
			if strings.HasSuffix(tok.Title, "/fixture") {
				title = tok.Title[0 : len(tok.Title)-8]
				fixtureNext = true
				continue
			}
			if strings.HasSuffix(tok.Title, "/length") {
				title2 := tok.Title[0 : len(tok.Title)-7]
				if title2 != title {
					t.Fatalf("out of order fixture data: got length section for %q, expected info for section %q", title2, title)
				}
				lengthNext = true
				continue
			}
		} else {
			if fixtureNext {
				fixtureRaw = bytes.Repeat(tok.Content, 1)
			}
			if lengthNext {
				length, err := strconv.Atoi(strings.TrimSpace(string(tok.Content)))
				if err != nil {
					t.Fatalf("expected %s/length section to parse as int: %s", title, err)
				}
				// Finally, the test!
				t.Run("title="+title, func(t *testing.T) {
					qt.Check(t, len(fixtureRaw), qt.Equals, length)
				})
			}
		}
	}
}

func TestTricky(t *testing.T) {
	f, err := os.Open("../taffy-tricky.taf")
	if err != nil {
		panic(err)
	}
	t.Run("detects-all-sections", func(t *testing.T) {
		f.Seek(0, 0)
		scanner := NewScanner(f)
		titles, sectionCount := scanTitles(t, scanner)
		expectedTitles := []string{
			"comment",
			"single-space/fixture",
			"single-space/length",
			"single-tab/fixture",
			"single-tab/comment",
			"single-tab/length",
			"tab-and-break/fixture",
			"tab-and-break/comment",
			"tab-and-break/length",
			"trailing",
		}
		qt.Check(t, titles, qt.DeepEquals, expectedTitles)
		t.Run("detects-all-section-content", func(t *testing.T) {
			// There's one content hunk for each section.
			qt.Check(t, sectionCount, qt.Equals, len(expectedTitles)*2)
		})
	})

	t.Run("testcases", func(t *testing.T) {
		f.Seek(0, 0)
		scanner := NewScanner(f)
		runTestcases(t, scanner)
	})
}

func TestNoncanonical(t *testing.T) {
	f, err := os.Open("../taffy-noncanonical.taf")
	if err != nil {
		panic(err)
	}
	t.Run("detects-leading-comment", func(t *testing.T) {
		f.Seek(0, 0)
		scanner := NewScanner(f)
		tok, err := scanner.Scan()
		qt.Check(t, tok, qt.DeepEquals, Token{Content: []byte("This file has a leading comment, which makes it noncanonical.")})
		qt.Check(t, err, qt.IsNil)
	})
	t.Run("detects-all-sections", func(t *testing.T) {
		f.Seek(0, 0)
		scanner := NewScanner(f)
		titles, sectionCount := scanTitles(t, scanner)
		expectedTitles := []string{
			"comment",
			"not-a-header/one/fixture", "not-a-header/one/length",
			"not-a-header/two/fixture", "not-a-header/two/length",
			"not-a-header/three/fixture", "not-a-header/three/length",
			"not-a-header/four/fixture", "not-a-header/four/length",
			"not-a-header/five/fixture", "not-a-header/five/length",
			"not-a-header/six/fixture", "not-a-header/six/length",
			"oneline-indented/fixture", "oneline-indented/length",
			"twoline-indented/fixture", "twoline-indented/length",
			"trailing",
		}
		qt.Check(t, titles, qt.DeepEquals, expectedTitles)
		t.Run("detects-all-section-content", func(t *testing.T) {
			// There's one content hunk for each section... plus one, for the leading comment.
			qt.Check(t, sectionCount, qt.Equals, len(expectedTitles)*2+1)
		})
	})

	t.Run("testcases", func(t *testing.T) {
		f.Seek(0, 0)
		scanner := NewScanner(f)
		runTestcases(t, scanner)
	})
}
