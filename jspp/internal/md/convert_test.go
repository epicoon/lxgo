package md

import (
	"testing"
	"time"
)

// Regression cases covering the supported markdown block and inline rules.
// Empty and blank-lines-only input are deliberately not included here; see
// TestConvertEmptyInput below.
//
// "header_underline" exercises the checker-ordering rule in
// blocks_builder.go: checkTitle runs before checkLine so that a title
// underlined with the common "---" style becomes an <h1> rather than being
// classified as a thematic break.
func TestConvert(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"blockquote_nested_list", "> - a\n> - b", "<div class=\"lx-md-container\"><blockquote class=\"lx-md-blockquote\"><ul><li>a</li><li>b</li></ul></blockquote></div>"},
		{"blockquote_simple", "> quoted line one\n> quoted line two", "<div class=\"lx-md-container\"><blockquote class=\"lx-md-blockquote\"><p class=\"lx-md-paragraph\"> quoted line one quoted line two</p></blockquote></div>"},
		{"codeBlock_indented", "normal text\n\n    code line one\n    code line two\n\nafter", "<div class=\"lx-md-container\"><p class=\"lx-md-paragraph\"> normal text</p><pre class=\"lx-md-codeblock\">code line one<br>code line two</pre><p class=\"lx-md-paragraph\"> after</p></div>"},
		{"codeBlock_typed", "before\n\n```js\nconsole.log(1);\nconsole.log(2);\n```\n\nafter", "<div class=\"lx-md-container\"><p class=\"lx-md-paragraph\"> before</p><div><pre class=\"lx-md-codeblock\" code-type=\"js\">console.log(1);<br>console.log(2);</pre><img src=\"\" onerror=\"if(lx.MdHighlighter)lx.MdHighlighter.highlight(this.parentNode.children[0]);this.parentNode.removeChild(this)\"><p class=\"lx-md-paragraph\"> after</p></div>"},
		{"codeBlock_typed_no_lang", "```\nplain fenced\n```", "<div class=\"lx-md-container\"><div><pre class=\"lx-md-codeblock\">plain fenced</pre><img src=\"\" onerror=\"if(lx.MdHighlighter)lx.MdHighlighter.highlight(this.parentNode.children[0]);this.parentNode.removeChild(this)\"></div>"},
		{"codeBlock_indented_escaping", "before\n\n    a < b && c > d\n\nafter", "<div class=\"lx-md-container\"><p class=\"lx-md-paragraph\"> before</p><pre class=\"lx-md-codeblock\">a &lt; b &amp;&amp; c &gt; d</pre><p class=\"lx-md-paragraph\"> after</p></div>"},
		{"codeBlock_typed_escaping", "```go\nfunc F[T any](a, b T) bool { return a < b && b > a }\n```", "<div class=\"lx-md-container\"><div><pre class=\"lx-md-codeblock\" code-type=\"go\">func F[T any](a, b T) bool { return a &lt; b &amp;&amp; b &gt; a }</pre><img src=\"\" onerror=\"if(lx.MdHighlighter)lx.MdHighlighter.highlight(this.parentNode.children[0]);this.parentNode.removeChild(this)\"></div>"},
		{"codeBlock_typed_blank_line_inside", "```go\npackage test\n\nfunc eee(a int) int { return a + 10 }\n```\n\nafter", "<div class=\"lx-md-container\"><div><pre class=\"lx-md-codeblock\" code-type=\"go\">package test<br><br>func eee(a int) int { return a + 10 }</pre><img src=\"\" onerror=\"if(lx.MdHighlighter)lx.MdHighlighter.highlight(this.parentNode.children[0]);this.parentNode.removeChild(this)\"><p class=\"lx-md-paragraph\"> after</p></div>"},
		{"del_and_sub_together", "~~deleted~~ and ~subbed~ together", "<div class=\"lx-md-container\"><p class=\"lx-md-paragraph\"> <del>deleted</del> and <sub>subbed</sub> together</p></div>"},
		{"hard_break", "Line one  \nLine two", "<div class=\"lx-md-container\"><p class=\"lx-md-paragraph\"> Line one<br> Line two</p></div>"},
		{"header_id", "# Title {#my-id}", "<div class=\"lx-md-container\"><h1 id=\"my-id\"> Title</h1></div>"},
		{"header_underline", "Big Title\n=========\n\nSmall Title\n-----------", "<div class=\"lx-md-container\"><h2>Big Title</h2><h1>Small Title</h1></div>"},
		{"header_underline_short_dash", "Small Title\n--", "<div class=\"lx-md-container\"><h1>Small Title</h1></div>"},
		{"headers_hash", "# H1\n## H2\n### H3\n#### H4\n##### H5\n###### H6", "<div class=\"lx-md-container\"><h1> H1</h1><h2> H2</h2><h3> H3</h3><h4> H4</h4><h5> H5</h5><h6> H6</h6></div>"},
		{"hr", "before\n\n---\n\nafter\n\n***\n\nend", "<div class=\"lx-md-container\"><p class=\"lx-md-paragraph\"> before</p><hr><p class=\"lx-md-paragraph\"> after</p><hr><p class=\"lx-md-paragraph\"> end</p></div>"},
		{"inline_formatting", "**bold** and *italic* and __alsobold__ and _alsoitalic_ and ~~strike~~ and ==mark== and ~sub~ and ^sup^ and `code`", "<div class=\"lx-md-container\"><p class=\"lx-md-paragraph\"> <strong>bold</strong> and <em>italic</em> and <strong>alsobold</strong> and <em>alsoitalic</em> and <del>strike</del> and <mark>mark</mark> and <sub>sub</sub> and <sup>sup</sup> and <code>code</code></p></div>"},
		{"links_images", "See [my link](http://example.com \"Title\") and ![alt text](http://example.com/img.png \"Img Title\")", "<div class=\"lx-md-container\"><p class=\"lx-md-paragraph\"> See <a href=\"http://example.com\" title=\"Title\">my link</a> and <img src=\"http://example.com/img.png\" title=\"Img Title\" alt=\"alt text\"></p></div>"},
		{"list_item_continuation", "- one\n  still one\n- two", "<div class=\"lx-md-container\"><ul><li>one still one</li><li>two</li></ul></div>"},
		{"list_with_blank_then_nested", "- one\n\n    nested paragraph\n- two", "<div class=\"lx-md-container\"><ul><li>one<p class=\"lx-md-paragraph\"> nested paragraph</p></li><li>two</li></ul></div>"},
		{"mixed_doc", "# Title\n\nSome *intro* text with a [link](http://x.com).\n\n- item one\n- item two\n    - nested\n\n> a quote\n\n```go\nfmt.Println(\"hi\")\n```\n", "<div class=\"lx-md-container\"><h1> Title</h1><p class=\"lx-md-paragraph\"> Some <em>intro</em> text with a <a href=\"http://x.com\">link</a>.</p><ul><li>item one</li><li>item two<ul><li>nested</li></ul></li></ul><blockquote class=\"lx-md-blockquote\"><p class=\"lx-md-paragraph\"> a quote</p></blockquote><div><pre class=\"lx-md-codeblock\" code-type=\"go\">fmt.Println(\"hi\")</pre><img src=\"\" onerror=\"if(lx.MdHighlighter)lx.MdHighlighter.highlight(this.parentNode.children[0]);this.parentNode.removeChild(this)\"></div>"},
		{"ordered_and_unordered_separate", "1. one\n2. two\n\n- three\n- four", "<div class=\"lx-md-container\"><ol><li>one</li><li>two</li></ol><ul><li>three</li><li>four</li></ul></div>"},
		{"ordered_list_flat", "1. one\n2. two\n3. three", "<div class=\"lx-md-container\"><ol><li>one</li><li>two</li><li>three</li></ol></div>"},
		{"ordered_list_nested", "1. one\n    1. nested one\n    2. nested two\n2. two", "<div class=\"lx-md-container\"><ol><li>one<ol><li>nested one</li><li>nested two</li></ol></li><li>two</li></ol></div>"},
		{"paragraph_multiline", "Hello\nworld, this is\na paragraph", "<div class=\"lx-md-container\"><p class=\"lx-md-paragraph\"> Hello world, this is a paragraph</p></div>"},
		{"paragraph_simple", "Hello world", "<div class=\"lx-md-container\"><p class=\"lx-md-paragraph\"> Hello world</p></div>"},
		{"table_no_align", "| A | B |\n| x | y |", "<div class=\"lx-md-container\"><table class=\"lx-md-table\"><tbody><tr><td class=\"lx-md-table-cell\">A</td><td class=\"lx-md-table-cell\">B</td></tr><tr><td class=\"lx-md-table-cell\">x</td><td class=\"lx-md-table-cell\">y</td></tr></tbody></table></div>"},
		{"table_single_row_no_header_sep", "| just | one | row |", "<div class=\"lx-md-container\"><table class=\"lx-md-table\"><tbody><tr><td class=\"lx-md-table-cell\">just</td><td class=\"lx-md-table-cell\">one</td><td class=\"lx-md-table-cell\">row</td></tr></tbody></table></div>"},
		{"table_with_align", "| A | B | C |\n|:--|:-:|--:|\n| 1 | 2 | 3 |\n| 4 | 5 | 6 |", "<div class=\"lx-md-container\"><table class=\"lx-md-table\"><thead><tr><th class=\"lx-md-table-header\" style=\"text-align: center\">A</th><th class=\"lx-md-table-header\" style=\"text-align: center\">B</th><th class=\"lx-md-table-header\" style=\"text-align: center\">C</th></tr></thead><tbody><tr><td class=\"lx-md-table-cell\" style=\"text-align: left\">1</td><td class=\"lx-md-table-cell\" style=\"text-align: center\">2</td><td class=\"lx-md-table-cell\" style=\"text-align: right\">3</td></tr><tr><td class=\"lx-md-table-cell\" style=\"text-align: left\">4</td><td class=\"lx-md-table-cell\" style=\"text-align: center\">5</td><td class=\"lx-md-table-cell\" style=\"text-align: right\">6</td></tr></tbody></table></div>"},
		{"three_level_nested_list", "- a\n    - b\n        - c\n    - d\n- e", "<div class=\"lx-md-container\"><ul><li>a<ul><li>b<ul><li>c</li></ul></li><li>d</li></ul></li><li>e</li></ul></div>"},
		{"unordered_list_flat", "- one\n- two\n- three", "<div class=\"lx-md-container\"><ul><li>one</li><li>two</li><li>three</li></ul></div>"},
		{"unordered_list_nested", "- one\n    - nested one\n    - nested two\n- two", "<div class=\"lx-md-container\"><ul><li>one<ul><li>nested one</li><li>nested two</li></ul></li><li>two</li></ul></div>"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Convert(c.input)
			if got != c.want {
				t.Errorf("Convert(%q)\n got: %s\nwant: %s", c.input, got, c.want)
			}
		})
	}
}

func TestConvertEmptyInput(t *testing.T) {
	// Empty or blank-only input must produce an empty (wrapped) result and
	// must not hang — cutTrailingEmpty's loop needs a bounds check to avoid
	// spinning forever on an empty slice.
	for _, input := range []string{"", "\n\n\n"} {
		done := make(chan string, 1)
		go func() { done <- Convert(input) }()
		select {
		case got := <-done:
			want := `<div class="lx-md-container"></div>`
			if got != want {
				t.Errorf("Convert(%q) = %q, want %q", input, got, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("Convert(%q) did not return within 2s (possible infinite loop)", input)
		}
	}
}
