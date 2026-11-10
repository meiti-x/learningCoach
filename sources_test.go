package main

import "testing"

func TestSplitMarkdownSections(t *testing.T) {
	md := "# Title\n\nintro\n\n## One\n\nbody one\n\n### Two\n\nbody two\n"
	secs := splitMarkdownSections(md)
	if len(secs) != 2 {
		t.Fatalf("got %d sections, want 2", len(secs))
	}
	if secs[0].Title != "One" || secs[1].Title != "Two" {
		t.Fatalf("titles: %#v", secs)
	}
	if secs[0].Body == "" || secs[1].Body == "" {
		t.Fatal("empty bodies")
	}
}
