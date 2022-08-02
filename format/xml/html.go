package xml

import (
	"embed"
	"strings"

	"github.com/wader/fq/format"
	"github.com/wader/fq/pkg/bitio"
	"github.com/wader/fq/pkg/decode"
	"github.com/wader/fq/pkg/interp"
	"github.com/wader/fq/pkg/scalar"
	"golang.org/x/net/html"
)

//go:embed html.jq
var htmlFS embed.FS

func init() {
	interp.RegisterFormat(decode.Format{
		Name:        format.HTML,
		Description: "HyperText Markup Language",
		DecodeFn:    decodeHTML,
		DecodeInArg: format.HTMLIn{
			Seq:   false,
			Array: false,
		},
		Functions: []string{"_todisplay"},
	})
	interp.RegisterFS(htmlFS)
}

func fromHTMLObject(n *html.Node, hi format.HTMLIn) any {
	var f func(n *html.Node, seq int) any
	f = func(n *html.Node, seq int) any {
		attrs := map[string]any{}

		switch n.Type {
		case html.ElementNode:
			for _, a := range n.Attr {
				attrs["-"+a.Key] = a.Val
			}
		default:
			// skip
		}

		nNodes := 0
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode {
				nNodes++
			}
		}
		nSeq := -1
		if nNodes > 1 {
			nSeq = 0
		}

		var textSb *strings.Builder
		var commentSb *strings.Builder

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			switch c.Type {
			case html.ElementNode:
				if e, ok := attrs[c.Data]; ok {
					if ea, ok := e.([]any); ok {
						attrs[c.Data] = append(ea, f(c, nSeq))
					} else {
						attrs[c.Data] = []any{e, f(c, nSeq)}
					}
				} else {
					attrs[c.Data] = f(c, nSeq)
				}
				if nNodes > 1 {
					nSeq++
				}
			case html.TextNode:
				if !whitespaceRE.MatchString(c.Data) {
					if textSb == nil {
						textSb = &strings.Builder{}
					}
					textSb.WriteString(c.Data)
				}
			case html.CommentNode:
				if !whitespaceRE.MatchString(c.Data) {
					if commentSb == nil {
						commentSb = &strings.Builder{}
					}
					commentSb.WriteString(c.Data)
				}
			default:
				// skip other nodes
			}

			if textSb != nil {
				attrs["#text"] = strings.TrimSpace(textSb.String())
			}
			if commentSb != nil {
				attrs["#comment"] = strings.TrimSpace(commentSb.String())
			}
		}

		if hi.Seq && seq != -1 {
			attrs["#seq"] = seq
		}

		if len(attrs) == 0 {
			return ""
		} else if len(attrs) == 1 && attrs["#text"] != nil {
			return attrs["#text"]
		}

		return attrs
	}

	return f(n, -1)
}

func fromHTMLArray(n *html.Node) any {
	var f func(n *html.Node) any
	f = func(n *html.Node) any {
		attrs := map[string]any{}

		switch n.Type {
		case html.ElementNode:
			for _, a := range n.Attr {
				attrs[a.Key] = a.Val
			}
		default:
			// skip
		}

		nodes := []any{}
		var textSb *strings.Builder
		var commentSb *strings.Builder

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			switch c.Type {
			case html.ElementNode:
				nodes = append(nodes, f(c))
			case html.TextNode:
				if !whitespaceRE.MatchString(c.Data) {
					if textSb == nil {
						textSb = &strings.Builder{}
					}
					textSb.WriteString(c.Data)
				}
			case html.CommentNode:
				if !whitespaceRE.MatchString(c.Data) {
					if commentSb == nil {
						commentSb = &strings.Builder{}
					}
					commentSb.WriteString(c.Data)
				}
			default:
				// skip other nodes
			}
		}

		if textSb != nil {
			attrs["#text"] = strings.TrimSpace(textSb.String())
		}
		if commentSb != nil {
			attrs["#comment"] = strings.TrimSpace(commentSb.String())
		}

		elm := []any{n.Data}
		if len(attrs) > 0 {
			elm = append(elm, attrs)
		}
		if len(nodes) > 0 {
			elm = append(elm, nodes)
		}

		return elm
	}

	return f(n.FirstChild)
}

func decodeHTML(d *decode.D, in any) any {
	hi, _ := in.(format.HTMLIn)

	br := d.RawLen(d.Len())
	var r any
	var err error
	// disabled scripting means parse noscript tags etc
	n, err := html.ParseWithOptions(bitio.NewIOReader(br), html.ParseOptionEnableScripting(false))
	if err != nil {
		d.Fatalf("%s", err)
	}

	if hi.Array {
		r = fromHTMLArray(n)
	} else {
		r = fromHTMLObject(n, hi)
	}
	if err != nil {
		d.Fatalf("%s", err)
	}
	var s scalar.S
	s.Actual = r

	d.Value.V = &s
	d.Value.Range.Len = d.Len()

	return nil
}
