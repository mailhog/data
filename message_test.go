package data

import (
	"testing"
)

func TestExtractBoundary(t *testing.T) {
	contents := []struct {
		content string
		expect  string
	}{
		{
			`multipart/alternative; boundary="_----------=_MCPart_498914860"`,
			`_----------=_MCPart_498914860`,
		},
		{
			`multipart/alternative; boundary=047d7bd74a2049b624050d805118`,
			`047d7bd74a2049b624050d805118`,
		},
	}

	for _, c := range contents {
		if b := extractBoundary(c.content); b != c.expect {
			t.Fatal("extractBoundary expect", c.expect, "but get", b)
		}
	}
}

func TestPathEquals(t *testing.T) {
	a := Path{
		Relays:  []string{"rel"},
		Mailbox: "foo",
		Domain:  "bar",
		Params:  "",
	}

	if a.equals(nil) {
		t.Errorf("Path should never be equal to nil")
	}

	if !a.equals(&a) {
		t.Errorf("references should be equal")
	}
	b := a
	if !a.equals(&b) {
		t.Errorf("%s should be equal to %s.", a, b)
	}
	mailboxChanged := a
	mailboxChanged.Mailbox = "!"
	if a.equals(&mailboxChanged) {
		t.Errorf("%s should NOT be equal to %s.", a, mailboxChanged)
	}
	domainChanged := a
	domainChanged.Domain = "!"
	if a.equals(&domainChanged) {
		t.Errorf("%s should NOT be equal to %s.", a, domainChanged)
	}
	paramsChanged := a
	paramsChanged.Params = "!"
	if a.equals(&paramsChanged) {
		t.Errorf("%s should NOT be equal to %s.", a, paramsChanged)
	}
	relaysChanged := a
	relaysChanged.Relays = []string{"baz"}
	if a.equals(&relaysChanged) {
		t.Errorf("%s should NOT be equal to %s.", a, relaysChanged)
	}
	relaysChanged.Relays = []string{"baz", "rel"}
	if a.equals(&relaysChanged) {
		t.Errorf("%s should NOT be equal to %s.", a, relaysChanged)
	}
}

func TestExtractBcc(t *testing.T) {
	content := Content{
		Size:    42,
		Headers: map[string][]string{},
		Body:    "body",
	}
	emptyTo := extractBcc(toPathes([]string{}), &content)
	if l := len(emptyTo); l != 0 {
		t.Errorf("result should be empty but had %d entries.", l)
	}

	content.Headers["Cc"] = []string{"admin@localhost, Alice <cjoao0@tinyurl.com>, aalesipo@example.com"}
	content.Headers["To"] = []string{"circulars@localhost, Bob <mdenslow2@taobao.com>"}
	allTos := []string{
		"circulars@localhost", "admin@localhost", "cjoao0@tinyurl.com",
		"aalesipo@example.com", "admin@localhost", "mdenslow2@taobao.com"}
	bcc := extractBcc(toPathes(allTos), &content)
	if l := len(bcc); l != 2 {
		t.Errorf("%v should have 2 entries but had %d entries.", bcc, l)
	}
	for _, x := range []string{"admin@localhost", "mdenslow2@taobao.com"} {
		if indexOf(bcc, PathFromString(x)) == -1 {
			t.Errorf("%v should contain %s.", bcc, x)
		}
	}

}
