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

package web

import (
	"fmt"
	"net"
	"net/http"
	"path/filepath"

	"github.com/DTRules/DTRules/pkg/dtrules/interview"
)

// Options configures ServeDir.
type Options struct {
	Entry        string // decision table to run (required)
	Input        string // optional input data file (loaded via the mapping)
	ResultEntity string // output entity to render (default "result")
	Title        string // browser tab/page title (default: the xmlDir name)
	NoOpen       bool   // suppress auto-opening the browser
}

// ServeDir serves the rule set in xmlDir as an interactive web interview on
// addr. It is the default UI wired to the default engine (interview.Project):
// each browser session runs the entry table, and any reached collect field
// that isn't supplied is asked via the web form. Unless NoOpen is set, the
// user's browser is opened once the listener is bound.
//
// To serve a custom engine instead, build an interview.Runner and pass it to
// NewServer directly.
func ServeDir(addr, xmlDir string, opts Options) error {
	title := opts.Title
	if title == "" {
		title = filepath.Base(xmlDir)
	}
	runner := interview.Project(xmlDir, interview.Options{
		Entry:        opts.Entry,
		Input:        opts.Input,
		ResultEntity: opts.ResultEntity,
	})
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	url := browserURL(ln.Addr())
	fmt.Printf("\n  %s web interview ready:  %s\n  (Ctrl-C to stop)\n\n", title, url)
	if !opts.NoOpen {
		go func() {
			if err := openBrowser(url); err != nil {
				fmt.Printf("  Could not auto-open a browser (%v). Open the URL above manually.\n", err)
			}
		}()
	}
	return http.Serve(ln, NewServer(runner, title))
}
