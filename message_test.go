package data

import (
	"reflect"
	"testing"
)

func TestContentFromString(t *testing.T) {
	// Long headers can be folded across multiple lines.
	get := ContentFromString(
		"To: foo@bar.com\r\n" +
			"X-Foo-Digest:\r\n" +
			" f71324948a11ad59c9f52aa27a1f194391968da6b7623186fedd0d190fd2f484\r\n" +
			"\r\n" +
			"body\r\n",
	)

	expect := &Content{
		Body: "body\r\n",
		Headers: map[string][]string{
			"To": []string{"foo@bar.com"},
			"X-Foo-Digest": []string{
				"f71324948a11ad59c9f52aa27a1f194391968da6b7623186fedd0d190fd2f484",
			},
		},
		Size: 107,
	}

	if !reflect.DeepEqual(get, expect) {
		t.Fatal("ContentFromString expect", expect, "but get", get)
	}
}

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
