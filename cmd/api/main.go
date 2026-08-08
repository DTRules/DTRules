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

// Command api runs the DTRules REST API server standalone. The server
// implementation lives in pkg/dtrules/apiserver, which `dtrules edit`
// also embeds alongside the static UI.
package main

import (
	"flag"
	"log"

	"github.com/DTRules/DTRules/pkg/dtrules/apiserver"
)

func main() {
	var (
		port        = flag.Int("port", 8080, "Port to listen on")
		projectRoot = flag.String("project-root", "", "Restrict project access to this directory tree")
		corsOrigin  = flag.String("cors-origin", "*", "Allowed CORS origin (default: * for development)")
		maxBodySize = flag.Int64("max-body-size", 10<<20, "Maximum request body size in bytes (default: 10MB)")
	)
	flag.Parse()

	log.Fatal(apiserver.Run(apiserver.Config{
		Port:        *port,
		ProjectRoot: *projectRoot,
		CORSOrigin:  *corsOrigin,
		MaxBodySize: *maxBodySize,
	}))
}
