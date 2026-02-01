// Copyright 2004-2011 DTRules.com, Inc.
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

package trace

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
)

// TraceLoader parses trace XML files and builds a TraceNode tree.
type TraceLoader struct {
	number   int          // Sequential node number counter
	tagStack []*TraceNode // Stack of nodes being built
	root     *TraceNode   // Root node (first element encountered)
}

// NewTraceLoader creates a new trace loader.
func NewTraceLoader() *TraceLoader {
	return &TraceLoader{
		number:   1,
		tagStack: make([]*TraceNode, 0),
	}
}

// LoadFile loads a trace from a file path.
func LoadFile(filepath string) (*TraceNode, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open trace file: %w", err)
	}
	defer file.Close()
	return Load(file)
}

// Load loads a trace from a reader.
func Load(r io.Reader) (*TraceNode, error) {
	loader := NewTraceLoader()
	return loader.Parse(r)
}

// Parse parses XML from the reader and returns the root TraceNode.
func (l *TraceLoader) Parse(r io.Reader) (*TraceNode, error) {
	decoder := xml.NewDecoder(r)
	decoder.Strict = false
	decoder.AutoClose = xml.HTMLAutoClose
	decoder.Entity = xml.HTMLEntity

	var currentBody strings.Builder

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("XML parse error: %w", err)
		}

		switch t := token.(type) {
		case xml.StartElement:
			// Save any accumulated body text to current node
			if len(l.tagStack) > 0 {
				body := strings.TrimSpace(currentBody.String())
				if body != "" {
					l.tagStack[len(l.tagStack)-1].Body = body
				}
			}
			currentBody.Reset()

			// Create attributes map
			attrs := make(map[string]string)
			for _, attr := range t.Attr {
				attrs[attr.Name.Local] = attr.Value
			}

			// Create new node
			node := NewTraceNode(l.number, t.Name.Local, attrs)
			l.number++

			// Track root (first element)
			if l.root == nil {
				l.root = node
			}

			// Add to parent if exists
			if len(l.tagStack) > 0 {
				l.tagStack[len(l.tagStack)-1].AddChild(node)
			}

			// Push onto stack
			l.tagStack = append(l.tagStack, node)

		case xml.EndElement:
			if len(l.tagStack) == 0 {
				return nil, fmt.Errorf("unexpected end tag: %s", t.Name.Local)
			}

			// Set body text
			body := strings.TrimSpace(currentBody.String())
			if body != "" {
				l.tagStack[len(l.tagStack)-1].Body = body
			}
			currentBody.Reset()

			// Pop from stack
			l.tagStack = l.tagStack[:len(l.tagStack)-1]

		case xml.CharData:
			currentBody.Write(t)
		}
	}

	// Return the tracked root
	if l.root != nil {
		return l.root, nil
	}

	return nil, fmt.Errorf("no root element found in trace file")
}

// GetNodeCount returns the total number of nodes created.
func (l *TraceLoader) GetNodeCount() int {
	return l.number - 1
}

// LoadAndCount loads a trace file and returns both the root node and count.
func LoadAndCount(r io.Reader) (*TraceNode, int, error) {
	loader := NewTraceLoader()
	root, err := loader.Parse(r)
	if err != nil {
		return nil, 0, err
	}
	return root, loader.GetNodeCount(), nil
}
