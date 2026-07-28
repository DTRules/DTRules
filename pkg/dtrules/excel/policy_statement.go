// Copyright 2024 Paul Snow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package excel

import "strings"

// CompilePolicyStatement turns a policy-statement template into postfix.
//
// A policy statement is not an EL expression — it is a template string in
// which `{expr}` substitutes the runtime value of `expr`:
//
//	thing.value is out of range, i.e.  {thing.value}
//
// compiles to
//
//	"thing.value is out of range, i.e.  " thing.value cvs strconcat "" strconcat
//
// which is the shape the original (Java) compiler emitted and what the
// checked-in sample XML carries: the leading literal, then one
// `<expr> cvs strconcat <literal> strconcat` group per substitution, folded
// left to right. A template that ends on a substitution therefore concatenates
// a trailing empty literal.
//
// A template with no `{...}` compiles to a single quoted literal, so plain
// descriptions are unchanged.
//
// Interpolation used to be dropped on the Excel→XML path: the description was
// wrapped in quotes verbatim, so `{thing.value}` became literal braces in the
// output string and the field reference was lost.
func CompilePolicyStatement(desc string) string {
	head, subs := splitTemplate(desc)
	var b strings.Builder
	b.WriteString(quotePostfixString(head))
	for _, s := range subs {
		b.WriteString(" ")
		b.WriteString(s.expr)
		b.WriteString(" cvs strconcat ")
		b.WriteString(quotePostfixString(s.tail))
		b.WriteString(" strconcat")
	}
	return b.String()
}

// substitution is one `{expr}` group plus the literal text that follows it.
type substitution struct {
	expr string
	tail string
}

// splitTemplate breaks a policy-statement template into the literal text
// before the first `{...}` and one substitution per group after it. An
// unterminated `{` is treated as literal text.
func splitTemplate(desc string) (head string, subs []substitution) {
	rest := desc
	for {
		open := strings.Index(rest, "{")
		if open < 0 {
			break
		}
		closeIdx := strings.Index(rest[open:], "}")
		if closeIdx < 0 {
			break
		}
		expr := strings.TrimSpace(rest[open+1 : open+closeIdx])
		if expr == "" {
			// `{}` carries no expression; leave it as literal text.
			break
		}
		if subs == nil {
			head = rest[:open]
		} else {
			subs[len(subs)-1].tail = rest[:open]
		}
		subs = append(subs, substitution{expr: expr})
		rest = rest[open+closeIdx+1:]
	}
	if subs == nil {
		return desc, nil
	}
	subs[len(subs)-1].tail = rest
	return head, subs
}

// quotePostfixString renders text as a postfix string literal. The postfix
// tokenizer (compiler.tokenize) recognizes backslash escapes inside quoted
// strings, so embedded quotes and backslashes are escaped rather than left to
// terminate the token early.
func quotePostfixString(text string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + r.Replace(text) + `"`
}
