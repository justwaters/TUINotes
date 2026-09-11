package convert

import "testing"

func TestLinesToHTMLLists(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "bullet list",
			in:   "Title\n* one\n* two",
			want: "<div>Title</div>\n<ul>\n<li>one</li>\n<li>two</li>\n</ul>",
		},
		{
			name: "dash list",
			in:   "Title\n- one\n- two",
			want: "<div>Title</div>\n<ul class=\"Apple-dash-list\">\n<li>one</li>\n<li>two</li>\n</ul>",
		},
		{
			name: "numbered list",
			in:   "Title\n1. one\n2. two",
			want: "<div>Title</div>\n<ol>\n<li>one</li>\n<li>two</li>\n</ol>",
		},
		{
			name: "list ends at a plain line",
			in:   "* one\n* two\nback to prose",
			want: "<ul>\n<li>one</li>\n<li>two</li>\n</ul>\n<div>back to prose</div>",
		},
		{
			name: "two separate bullet blocks stay separate",
			in:   "* a\n* b\n\n* c",
			want: "<ul>\n<li>a</li>\n<li>b</li>\n</ul>\n<div><br></div>\n<ul>\n<li>c</li>\n</ul>",
		},
		{
			// Notes.app merges two adjacent list blocks of different types
			// into one (confirmed against a real saved note), silently
			// discarding the second list's type. A separator must be
			// inserted even though the user didn't type a blank line.
			name: "adjacent different-kind lists get an inserted separator",
			in:   "1. one\n2. two\n- x\n- y",
			want: "<ol>\n<li>one</li>\n<li>two</li>\n</ol>\n<div><br></div>\n<ul class=\"Apple-dash-list\">\n<li>x</li>\n<li>y</li>\n</ul>",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := LinesToHTML(tc.in)
			if got != tc.want {
				t.Errorf("LinesToHTML(%q) =\n%q\nwant\n%q", tc.in, got, tc.want)
			}
		})
	}
}

func TestHTMLToLines(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain divs",
			in:   "<div>Title</div>\n<div>body</div>",
			want: "Title\nbody",
		},
		{
			name: "bullet list",
			in:   "<div>Title</div>\n<ul>\n<li>one</li>\n<li>two</li>\n</ul>",
			want: "Title\n* one\n* two",
		},
		{
			name: "dash list",
			in:   "<div>Title</div>\n<ul class=\"Apple-dash-list\">\n<li>one</li>\n<li>two</li>\n</ul>",
			want: "Title\n- one\n- two",
		},
		{
			name: "numbered list",
			in:   "<div>Title</div>\n<ol>\n<li>one</li>\n<li>two</li>\n</ol>",
			want: "Title\n1. one\n2. two",
		},
		{
			name: "strips inline formatting to plain text",
			in:   "<div><h1>Title</h1></div>\n<div><b>bold</b> and <i>italic</i></div>",
			want: "Title\nbold and italic",
		},
		{
			name: "real note: bullet list with trailing br and inline tags",
			in:   "<div><h1>Shot List</h1></div>\n<ul>\n<li>3 shots (outside)<br></li>\n<li>2 <u>posed</u> shots</li>\n</ul>\n",
			want: "Shot List\n* 3 shots (outside)\n* 2 posed shots",
		},
		{
			name: "real note: numbered list with entity and link text",
			in:   "<div><h1>List</h1></div>\n<ol>\n<li>griddle &amp; element</li>\n<li><i>Book</i> by Author</li>\n</ol>\n",
			want: "List\n1. griddle & element\n2. Book by Author",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := HTMLToLines(tc.in)
			if err != nil {
				t.Fatalf("HTMLToLines(%q) error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("HTMLToLines(%q) =\n%q\nwant\n%q", tc.in, got, tc.want)
			}
		})
	}
}

func TestListRoundTrip(t *testing.T) {
	cases := []string{
		"Title\n* one\n* two",
		"Title\n- one\n- two",
		"Title\n1. one\n2. two",
		"Title\nplain line\n* a\n* b\nback to plain",
	}
	for _, text := range cases {
		t.Run(text, func(t *testing.T) {
			htmlOut := LinesToHTML(text)
			back, err := HTMLToLines(htmlOut)
			if err != nil {
				t.Fatalf("HTMLToLines error: %v", err)
			}
			if back != text {
				t.Errorf("round trip mismatch:\n  original: %q\n  html:     %q\n  back:     %q", text, htmlOut, back)
			}
		})
	}
}

func TestLinesToHTML(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "title and body lines",
			in:   "My Title\nsecond line",
			want: "<div>My Title</div>\n<div>second line</div>",
		},
		{
			name: "blank line becomes br",
			in:   "Title\n\nafter blank",
			want: "<div>Title</div>\n<div><br></div>\n<div>after blank</div>",
		},
		{
			name: "escapes HTML-significant characters",
			in:   "A & B < C",
			want: "<div>A &amp; B &lt; C</div>",
		},
		{
			name: "single line",
			in:   "Just a title",
			want: "<div>Just a title</div>",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := LinesToHTML(tc.in)
			if got != tc.want {
				t.Errorf("LinesToHTML(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
