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

// Command sinusitis-web is the #850/#855 demo: a single self-contained binary
// that embeds the SinusitisTherapy rule set (//go:embed, no xlsx/xml on disk
// at runtime) and serves it as an interactive web interview. Run it, open the
// page, answer the questions, and the rules compute the therapy.
package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/DTRules/DTRules/pkg/dtrules/web"
)

//go:embed rules/xml
var rulesFS embed.FS

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	flag.Parse()

	dir, err := extractRules()
	if err != nil {
		fmt.Fprintf(os.Stderr, "extract rules: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)

	fmt.Println("Sinusitis Therapy web demo — opening your browser...")
	if err := web.ServeDir(*addr, dir, web.Options{Entry: "Determine_Therapy"}); err != nil {
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}
}

// extractRules writes the embedded compiled rules to a temp directory so the
// loader (which reads from disk) can pick them up. The binary itself carries
// the rules; nothing else needs to ship with it.
func extractRules() (string, error) {
	dir, err := os.MkdirTemp("", "sinusitis-web-*")
	if err != nil {
		return "", err
	}
	const root = "rules/xml"
	err = fs.WalkDir(rulesFS, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		target := filepath.Join(dir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := rulesFS.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}
