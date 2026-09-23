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

package authoring

// Typed errors for the failures a caller has to tell apart. Each wraps the
// error it was built from and prints it unchanged, so messages stay as they
// were; what is new is that errors.As can say which kind of failure it was.
//
// Without them, `table put` reported every refusal as an EL compile error --
// a table number outside its file's range included -- and sent the caller to
// look for a broken expression that did not exist (#1222).

// ELError is an EL expression that did not compile.
type ELError struct {
	Kind string // "condition", "action" or "context"
	DSL  string
	Err  error
}

func (e *ELError) Error() string { return e.Err.Error() }
func (e *ELError) Unwrap() error { return e.Err }

// OtherwiseError is a column grid that breaks the otherwise-column rule
// (OtherwiseRule).
type OtherwiseError struct{ Err error }

func (e *OtherwiseError) Error() string { return e.Err.Error() }
func (e *OtherwiseError) Unwrap() error { return e.Err }

// NumberError is a table number that is outside its file's range or already
// taken.
type NumberError struct{ Err error }

func (e *NumberError) Error() string { return e.Err.Error() }
func (e *NumberError) Unwrap() error { return e.Err }

func elError(kind, dsl string, err error) error {
	if err == nil {
		return nil
	}
	return &ELError{Kind: kind, DSL: dsl, Err: err}
}
