// Copyright 2026 Paul Snow
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

package apiserver

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoveredProject is one project found under a directory. A project is
// identified by its DTRules.xml (one per project), or — for projects that
// predate the config file — by the layout convention: an xml/ directory
// holding rule XML, or rule XML directly in the directory.
type DiscoveredProject struct {
	Path   string `json:"path"`   // absolute directory
	Name   string `json:"name"`   // directory base name
	Marker string `json:"marker"` // "DTRules.xml" | "xml/" | "rule files"
	Entry  string `json:"entry"`  // <entry> from DTRules.xml, when present
}

// maxDiscoverDepth bounds the walk so discovery stays fast on big trees.
const maxDiscoverDepth = 5

// skipDirNames are never descended into during discovery.
var skipDirNames = map[string]bool{
	".git": true, "node_modules": true, "build": true, "dist": true,
	"legacy": true, "testdata": true,
}

// hasRuleXML reports whether dir directly contains at least one *_dt.xml
// or *edd*.xml file.
func hasRuleXML(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := strings.ToLower(e.Name())
		if !strings.HasSuffix(n, ".xml") {
			continue
		}
		if strings.Contains(n, "_dt") || strings.Contains(n, "edd") {
			return true
		}
	}
	return false
}

// ProjectMarker classifies dir as a project. It returns the marker that
// identifies it, or "" when the directory is not a project.
func ProjectMarker(dir string) string {
	if _, err := os.Stat(filepath.Join(dir, "DTRules.xml")); err == nil {
		return "DTRules.xml"
	}
	if xd := filepath.Join(dir, "xml"); dirExists(xd) && hasRuleXML(xd) {
		return "xml/"
	}
	if hasRuleXML(dir) {
		return "rule files"
	}
	return ""
}

// DiscoverProjects walks root (bounded depth, skipping VCS/build dirs) and
// returns every directory identified as a project. Once a directory is
// identified, its subtree is not searched further — a project does not
// contain other projects.
func DiscoverProjects(root string) []DiscoveredProject {
	var found []DiscoveredProject
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		if marker := ProjectMarker(dir); marker != "" {
			p := DiscoveredProject{
				Path:   dir,
				Name:   filepath.Base(dir),
				Marker: marker,
			}
			if marker == "DTRules.xml" {
				p.Entry = readProjectConfig(dir).Entry
			}
			found = append(found, p)
			return
		}
		if depth >= maxDiscoverDepth {
			return
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !e.IsDir() || skipDirNames[e.Name()] || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			walk(filepath.Join(dir, e.Name()), depth+1)
		}
	}
	// The root itself being a project is the caller's case to handle
	// (it would just load it); discovery is for the "not a project here,
	// what is nearby?" question — so start at the children.
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() || skipDirNames[e.Name()] || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		walk(filepath.Join(root, e.Name()), 1)
	}
	sort.Slice(found, func(i, j int) bool { return found[i].Path < found[j].Path })
	return found
}

// SetDiscoverRoot records the directory `dtrules edit` was launched from
// when that directory is not itself a project; the welcome screen offers
// the projects discovered beneath it.
func (s *Server) SetDiscoverRoot(dir string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.discoverRoot = dir
}

// handleProjectDiscover lists projects under the launch directory (or an
// explicit ?path=, validated against the project root restriction).
func (s *Server) handleProjectDiscover(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		jsonError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.RLock()
	root := s.discoverRoot
	s.mu.RUnlock()
	if q := r.URL.Query().Get("path"); q != "" {
		validated, err := s.validateProjectPath(q)
		if err != nil {
			jsonError(w, "invalid path: "+err.Error(), http.StatusBadRequest)
			return
		}
		root = validated
	}
	if root == "" {
		jsonResponse(w, map[string]interface{}{"success": true, "root": "", "projects": []DiscoveredProject{}})
		return
	}
	jsonResponse(w, map[string]interface{}{
		"success":  true,
		"root":     root,
		"projects": DiscoverProjects(root),
	})
}
