package parser

import (
	"strings"
	"testing"
)

func TestWikiOccurrencesPositions(t *testing.T) {
	src := "---\nrelated: '[[metadata]]'\n---\nα [[One]] and [[One]]\r\n> [[#Heading]]\n\n- ![[image.png]]\n\n`[[code]]`\n"
	_, doc, err := ParseDocument([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	refs := doc.WikiOccurrences()
	if len(refs) != 4 {
		t.Fatalf("occurrences: %#v", refs)
	}
	for i, want := range []string{"[[One]]", "[[One]]", "[[#Heading]]", "![[image.png]]"} {
		offset := refs[i].Offset
		if offset < 0 || !strings.HasPrefix(src[offset:], want) {
			t.Errorf("occurrence %d offset %d not at %q", i, offset, want)
		}
	}
	if refs[0].Offset == refs[1].Offset || refs[2].Heading != "Heading" || !refs[3].Image {
		t.Fatalf("occurrences: %#v", refs)
	}
}
