// Copyright 2025 DTRules contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules/apiserver"
	"github.com/DTRules/DTRules/ui"
)

// runEdit serves the embedded editor UI with the API backend and opens the
// browser. The UI bundle is compiled in with `-tags ui` (see ui/embed.go);
// without it, this command explains how to get an editor-enabled build.
func (c *CLI) runEdit(args []string) int {
	port := 8080
	host := "127.0.0.1"
	openBrowser := true
	projectPath := ""
	projectRoot := ""
	readOnly := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--port", "-p":
			if i+1 < len(args) {
				i++
				fmt.Sscanf(args[i], "%d", &port)
			}
		case "--host":
			if i+1 < len(args) {
				i++
				host = args[i]
			}
		case "--project-root":
			if i+1 < len(args) {
				i++
				projectRoot = args[i]
			}
		case "--read-only":
			readOnly = true
		case "--no-browser":
			openBrowser = false
		case "-h", "--help":
			fmt.Println(`Usage: dtrules edit [project-dir] [options]

Serves the DTRules editor UI in your browser, backed by the embedded API
server. The project directory should contain the project's XML files
(defaults to ./xml if present, else the current directory).

Options:
  --port, -p <n>     Port to listen on (default 8080)
  --host <addr>      Bind address (default 127.0.0.1; use 0.0.0.0 to
                     serve the editor from a server)
  --project-root <d> Restrict project opening/browsing to this directory
                     tree (recommended for server deployments)
  --read-only        Publish rules for review: all editing and saving is
                     rejected server-side; reading and execution work
  --no-browser       Don't open the browser automatically

Server deployment example (published, read-only):
  dtrules edit /srv/rules/xml --host 0.0.0.0 --port 8080 \
      --project-root /srv/rules --read-only

Note: the editor has no authentication — on a server, front it with a
reverse proxy that provides TLS and access control.`)
			return 0
		default:
			projectPath = args[i]
		}
	}

	dist, ok := ui.Dist()
	if !ok {
		fmt.Fprintln(os.Stderr, "This build does not include the editor UI.")
		fmt.Fprintln(os.Stderr, "Build one with: make build-edit   (requires npm; runs `make ui-dist` then a -tags ui build)")
		return 1
	}

	// Default project: ./xml when present (project-layout convention),
	// otherwise the current directory.
	if projectPath == "" {
		if info, err := os.Stat("xml"); err == nil && info.IsDir() {
			projectPath = "xml"
		} else {
			projectPath = "."
		}
	}
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid project path: %v\n", err)
		return 1
	}

	server := apiserver.New(apiserver.Config{ProjectRoot: projectRoot, ReadOnly: readOnly})
	if err := server.LoadProject(absPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not open project %s: %v\n", absPath, err)
		fmt.Fprintln(os.Stderr, "The editor will start without a project; open one from the UI.")
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", server.Routes())
	mux.Handle("/", http.FileServer(http.FS(dist)))

	loopback := host == "127.0.0.1" || host == "localhost" || host == "::1"
	url := fmt.Sprintf("http://localhost:%d", port)
	if !loopback {
		url = fmt.Sprintf("http://%s:%d", host, port)
	}
	fmt.Printf("DTRules editor: %s  (project: %s)\n", url, absPath)
	if projectRoot != "" {
		fmt.Printf("Project access restricted to: %s\n", projectRoot)
	}
	if readOnly {
		fmt.Println("Read-only mode: editing and saving are disabled.")
	}
	if !loopback {
		fmt.Println("Serving on a non-loopback address — the editor has no authentication;")
		fmt.Println("front it with a reverse proxy for TLS and access control.")
	}
	fmt.Println("Press Ctrl+C to stop.")

	// Only auto-open a browser for local sessions.
	if openBrowser && loopback {
		go func() {
			time.Sleep(300 * time.Millisecond)
			launchBrowser(url)
		}()
	}

	if err := http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), mux); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		return 1
	}
	return 0
}

// launchBrowser best-effort opens the system browser at url.
func launchBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
