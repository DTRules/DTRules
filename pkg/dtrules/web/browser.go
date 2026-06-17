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
	"os"
	"os/exec"
	"runtime"
)

// browserURL builds a localhost URL from a bound listener address, preferring
// "localhost" over the wildcard host a listener reports (e.g. "[::]:8080").
func browserURL(addr net.Addr) string {
	port := "8080"
	if tcp, ok := addr.(*net.TCPAddr); ok {
		port = fmt.Sprintf("%d", tcp.Port)
	}
	return "http://localhost:" + port + "/"
}

// openBrowser best-effort opens url in the user's default browser, trying a
// few launchers in turn. Returns an error only if none could be started.
func openBrowser(url string) error {
	var candidates [][]string
	switch runtime.GOOS {
	case "darwin":
		candidates = [][]string{{"open", url}}
	case "windows":
		candidates = [][]string{{"rundll32", "url.dll,FileProtocolHandler", url}}
	default: // linux, *bsd
		if b := os.Getenv("BROWSER"); b != "" {
			candidates = append(candidates, []string{b, url})
		}
		candidates = append(candidates,
			[]string{"xdg-open", url},
			[]string{"gio", "open", url},
			[]string{"sensible-browser", url},
			[]string{"x-www-browser", url},
		)
	}
	for _, c := range candidates {
		path, err := exec.LookPath(c[0])
		if err != nil {
			continue
		}
		if err := exec.Command(path, c[1:]...).Start(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("no usable browser launcher found")
}
