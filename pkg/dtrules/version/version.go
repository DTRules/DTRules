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

// Package version provides version information for DTRules.
// Version values are injected at build time using ldflags.
package version

import (
	"fmt"
	"runtime/debug"
)

// These variables are set at build time using ldflags:
//
//	go build -ldflags "-X github.com/DTRules/DTRules/pkg/dtrules/version.Version=1.0.0"
var (
	// Version is the semantic version (e.g., "5.0.0", "5.0.0-SNAPSHOT")
	Version = "dev"

	// GitCommit is the git commit hash
	GitCommit = ""

	// GitBranch is the git branch name
	GitBranch = ""

	// BuildDate is the build timestamp
	BuildDate = ""
)

// Info returns formatted version information.
func Info() string {
	v := Version

	// Try to get version from build info if not set
	if v == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				v = info.Main.Version
			}
		}
	}

	return fmt.Sprintf("DTRules %s", v)
}

// Full returns full version information including git details.
func Full() string {
	info := Info()

	if GitCommit != "" {
		info += fmt.Sprintf("\nCommit: %s", GitCommit)
	}
	if GitBranch != "" {
		info += fmt.Sprintf("\nBranch: %s", GitBranch)
	}
	if BuildDate != "" {
		info += fmt.Sprintf("\nBuilt:  %s", BuildDate)
	}

	return info
}

// Short returns just the version string.
func Short() string {
	if Version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				return info.Main.Version
			}
		}
	}
	return Version
}
