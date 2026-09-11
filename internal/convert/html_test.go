package convert

import "testing"

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
