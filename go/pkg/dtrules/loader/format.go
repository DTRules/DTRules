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

package loader

import (
	"bytes"
	"fmt"
	"io"
)

// Format represents a file format for DTRules data files.
type Format int

const (
	FormatUnknown Format = iota // FormatUnknown indicates the format could not be determined.
	FormatXML                   // FormatXML indicates XML format (first non-whitespace byte is '<').
	FormatJSON                  // FormatJSON indicates JSON format (first non-whitespace byte is '{' or '[').
)

// String returns the string representation of a Format.
func (f Format) String() string {
	switch f {
	case FormatXML:
		return "XML"
	case FormatJSON:
		return "JSON"
	default:
		return "Unknown"
	}
}

// DetectFormat reads the beginning of the input to determine whether the
// content is XML or JSON. It returns the detected format and a new reader
// that replays the peeked bytes followed by the remaining input.
// The detection skips leading whitespace and BOM, then checks the first
// non-whitespace byte: '<' indicates XML, '{' or '[' indicates JSON.
func DetectFormat(r io.Reader) (Format, io.Reader, error) {
	// Read enough bytes to detect format (skip BOM + whitespace)
	buf := make([]byte, 512)
	n, err := io.ReadAtLeast(r, buf, 1)
	if err != nil {
		return FormatUnknown, nil, fmt.Errorf("failed to detect format: %w", err)
	}
	buf = buf[:n]

	// Skip UTF-8 BOM if present
	data := buf
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	// Find first non-whitespace byte
	for len(data) > 0 {
		b := data[0]
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			data = data[1:]
			continue
		}

		// Reconstruct a reader with the peeked bytes + remaining input
		combined := io.MultiReader(bytes.NewReader(buf), r)

		switch b {
		case '<':
			return FormatXML, combined, nil
		case '{', '[':
			return FormatJSON, combined, nil
		default:
			return FormatUnknown, combined, fmt.Errorf("unexpected leading byte 0x%02x; expected '<' (XML) or '{'/'[' (JSON)", b)
		}
	}

	// All bytes were whitespace; reconstruct reader and return unknown
	combined := io.MultiReader(bytes.NewReader(buf), r)
	return FormatUnknown, combined, fmt.Errorf("input contains only whitespace")
}
