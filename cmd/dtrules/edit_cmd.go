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
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/DTRules/DTRules/pkg/dtrules/apiserver"
	"github.com/DTRules/DTRules/ui"
)

// defaultEditPort is where the editor's free-port scan starts. Deliberately
// not 8080 — that's every dev tool's default, so assuming it is free (or
// that whatever answers on it is us) caused silent cross-talk.
const defaultEditPort = 8330

// pickListener binds the editor's port. An explicitly requested port is
// bound exactly (failing loudly if taken — the user asked for it). With no
// request, ports are probed from defaultEditPort upward and the first that
// actually binds wins, falling back to an OS-assigned port. Binding is the
// test: no check-then-bind race.
func pickListener(host string, requested int) (net.Listener, error) {
	if requested > 0 {
		return net.Listen("tcp", fmt.Sprintf("%s:%d", host, requested))
	}
	for p := defaultEditPort; p < defaultEditPort+100; p++ {
		if ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, p)); err == nil {
			return ln, nil
		}
	}
	return net.Listen("tcp", fmt.Sprintf("%s:0", host))
}

// runEdit serves the embedded editor UI with the API backend and opens the
// browser. The UI bundle is compiled in with `-tags ui` (see ui/embed.go);
// without it, this command explains how to get an editor-enabled build.
func (c *CLI) runEdit(args []string) int {
	port := 0 // 0 = probe for a free port from defaultEditPort
	host := "127.0.0.1"
	openBrowser := true
	projectPath := ""
	projectRoot := ""
	readOnly := false
	tracePath := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--trace":
			if i+1 < len(args) {
				i++
				tracePath = args[i]
			}
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
  --port, -p <n>     Port to listen on. Default: the first free port from
                     8330 (probed by binding, so it is testably free); an
                     explicit port fails if something already holds it
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
	} else {
		reportProjectScope(server)
	}

	// Preload a trace so the Debug tab opens ready (the `dtrules debug` flow).
	if tracePath != "" {
		absTrace, err := filepath.Abs(tracePath)
		if err == nil {
			err = server.LoadDebugTrace(absTrace)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not load trace %s: %v\n", tracePath, err)
		} else {
			fmt.Printf("Trace loaded: %s\n", tracePath)
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", server.Routes())
	static := http.FileServer(http.FS(dist))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// index.html must always revalidate, or browsers keep serving a
		// cached page referencing the PREVIOUS build's hashed bundle —
		// the "works after a hard refresh" trap. The content-hashed
		// assets themselves are immutable and safe to cache hard.
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		static.ServeHTTP(w, r)
	}))

	ln, err := pickListener(host, port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not bind %s:%d: %v\n", host, port, err)
		return 1
	}
	boundPort := ln.Addr().(*net.TCPAddr).Port

	loopback := host == "127.0.0.1" || host == "localhost" || host == "::1"
	url := fmt.Sprintf("http://localhost:%d", boundPort)
	if !loopback {
		url = fmt.Sprintf("http://%s:%d", host, boundPort)
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

	if err := http.Serve(ln, mux); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		return 1
	}
	return 0
}

// launchBrowser best-effort opens the system browser at url.
// reportProjectScope prints where the editor actually got its rules, and
// warns when the scan fell back to the project root and swept rule files
// out of multiple nested directories — the signature of launching from a
// repo root that contains several projects (the whole tree gets walked,
// and same-named tables from unrelated projects silently collide).
func reportProjectScope(server *apiserver.Server) {
	projectPath, rulesDir, dtFiles, eddFiles := server.ProjectSummary()
	if projectPath == "" {
		return
	}
	fmt.Printf("Project:   %s\n", projectPath)
	fmt.Printf("Rules dir: %s  (%d decision-table files, %d EDD files)\n",
		rulesDir, len(dtFiles), len(eddFiles))

	if rulesDir != projectPath {
		return
	}
	// Fallback-to-root scan: warn when NO decision tables live at the scan
	// root itself and several subdirectories contributed them — a real
	// project keeps its main *_dt.xml at its top level (subfolders like
	// states/ are fine), while a repo root has everything nested under
	// unrelated directories.
	tops := map[string]bool{}
	rootFiles := 0
	for _, f := range dtFiles {
		parts := strings.SplitN(filepath.ToSlash(f), "/", 2)
		if len(parts) == 2 {
			tops[parts[0]] = true
		} else {
			rootFiles++
		}
	}
	if rootFiles == 0 && len(tops) > 1 {
		names := make([]string, 0, len(tops))
		for t := range tops {
			names = append(names, t)
		}
		sort.Strings(names)
		fmt.Fprintf(os.Stderr, `
WARNING: no DTRules.xml or xml/ directory here, so the WHOLE tree was
scanned and rule files were found under %d different top-level
directories (%s).
Tables from unrelated projects can collide. Run the editor from a
project directory, or pass one:  dtrules edit <project-dir>

`, len(tops), strings.Join(names, ", "))
	}
}

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
