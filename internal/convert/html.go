// Package convert reconstructs the minimal per-line HTML that Notes.app
// stores internally from the plain text a user edits in the TUI, and vice
// versa. It has no dependency on Notes.app itself.
package convert

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// List markup in the edited plain text. Notes.app represents each of these
// as a distinct block type in its body HTML (confirmed by inspecting real
// notes, since Notes has no public format documentation): a plain <ul> for
// bullets, <ul class="Apple-dash-list"> for dashes, and <ol> for numbered
// lists — each wrapping one <li> per item.
const (
	bulletPrefix  = "* "
	dashPrefix    = "- "
	dashListClass = "Apple-dash-list"
)

var numberedPrefix = regexp.MustCompile(`^\d+\.\s+`)

type lineKind int

const (
	kindPlain lineKind = iota
	kindBullet
	kindDash
	kindNumbered
)

// classifyLine identifies a line's list marker (if any) and returns its
// text with that marker stripped.
func classifyLine(s string) (lineKind, string) {
	if rest, ok := strings.CutPrefix(s, bulletPrefix); ok {
		return kindBullet, rest
	}
	if rest, ok := strings.CutPrefix(s, dashPrefix); ok {
		return kindDash, rest
	}
	if loc := numberedPrefix.FindStringIndex(s); loc != nil {
		return kindNumbered, s[loc[1]:]
	}
	return kindPlain, s
}

// LinesToHTML converts multi-line plain text — where, matching Notes.app's
// own convention, the first line is the note's title — into the per-line
// HTML Notes.app expects for a note body. Notes.app derives both the
// note's name and its plaintext property from this structure, so callers
// should never also set the note's name directly.
//
// A line prefixed with "* " becomes a bulleted list item, "- " a dashed
// list item, and "1. " (any number) a numbered list item; consecutive
// lines of the same kind are grouped into one list block, matching how
// Notes.app itself stores lists. Any other line becomes a plain paragraph.
func LinesToHTML(text string) string {
	lines := strings.Split(text, "\n")
	var blocks []string
	lastKind := kindPlain
	for i := 0; i < len(lines); {
		kind, first := classifyLine(lines[i])
		if kind == kindPlain {
			blocks = append(blocks, divBlock(lines[i]))
			lastKind = kindPlain
			i++
			continue
		}
		items := []string{first}
		i++
		for i < len(lines) {
			k, text := classifyLine(lines[i])
			if k != kind {
				break
			}
			items = append(items, text)
			i++
		}
		// Notes.app merges two adjacent list blocks of different types into
		// one (silently discarding the second one's type — confirmed by
		// inspecting a real note saved through the app), so a blank line
		// must separate them even if the user didn't type one.
		if lastKind != kindPlain && lastKind != kind {
			blocks = append(blocks, divBlock(""))
		}
		blocks = append(blocks, listBlock(kind, items))
		lastKind = kind
	}
	return strings.Join(blocks, "\n")
}

func divBlock(line string) string {
	if line == "" {
		return "<div><br></div>"
	}
	return "<div>" + html.EscapeString(line) + "</div>"
}

func listBlock(kind lineKind, items []string) string {
	tag, attrs := "ul", ""
	if kind == kindNumbered {
		tag = "ol"
	} else if kind == kindDash {
		attrs = fmt.Sprintf(` class="%s"`, dashListClass)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "<%s%s>", tag, attrs)
	for _, item := range items {
		b.WriteString("\n<li>")
		b.WriteString(html.EscapeString(item))
		b.WriteString("</li>")
	}
	fmt.Fprintf(&b, "\n</%s>", tag)
	return b.String()
}

// HTMLToLines is the inverse of LinesToHTML: it walks a note's real body
// HTML and reconstructs the plain-text (with list markers) representation
// the TUI editor shows. Inline formatting (bold, italic, headings, colors)
// has no plain-text equivalent and is dropped, keeping only the text
// content — this is the documented v1 tradeoff of editing notes as plain
// text.
func HTMLToLines(bodyHTML string) (string, error) {
	context := &xhtml.Node{Type: xhtml.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := xhtml.ParseFragment(strings.NewReader(bodyHTML), context)
	if err != nil {
		return "", fmt.Errorf("convert: parsing note body: %w", err)
	}

	var lines []string
	for _, n := range nodes {
		if n.Type != xhtml.ElementNode {
			continue // stray whitespace between blocks
		}
		switch n.Data {
		case "ul":
			prefix := bulletPrefix
			if hasClass(n, dashListClass) {
				prefix = dashPrefix
			}
			for _, item := range listItems(n) {
				lines = append(lines, prefix+item)
			}
		case "ol":
			for i, item := range listItems(n) {
				lines = append(lines, fmt.Sprintf("%d. %s", i+1, item))
			}
		default: // div, or any other block-level element Notes emits
			lines = append(lines, innerText(n))
		}
	}
	return strings.Join(lines, "\n"), nil
}

func listItems(list *xhtml.Node) []string {
	var items []string
	for li := list.FirstChild; li != nil; li = li.NextSibling {
		if li.Type == xhtml.ElementNode && li.Data == "li" {
			items = append(items, innerText(li))
		}
	}
	return items
}

func hasClass(n *xhtml.Node, class string) bool {
	for _, a := range n.Attr {
		if a.Key != "class" {
			continue
		}
		for _, c := range strings.Fields(a.Val) {
			if c == class {
				return true
			}
		}
	}
	return false
}

// innerText concatenates the text content of n and its descendants,
// dropping all tags — the mechanism by which inline formatting (b/i/u/
// font/h1/h2/br) is stripped down to plain text.
func innerText(n *xhtml.Node) string {
	var b strings.Builder
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.TextNode {
			b.WriteString(n.Data)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}
