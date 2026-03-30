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

package runtime

import (
	"testing"
)

func TestDefaultContextConfig(t *testing.T) {
	cfg := DefaultContextConfig()

	if cfg.DataStackSize != 1024 {
		t.Errorf("expected DataStackSize 1024, got %d", cfg.DataStackSize)
	}
	if cfg.EntityStackSize != 64 {
		t.Errorf("expected EntityStackSize 64, got %d", cfg.EntityStackSize)
	}
	if cfg.EnableTracing {
		t.Error("expected EnableTracing false")
	}
	if cfg.TraceWriter != nil {
		t.Error("expected TraceWriter nil")
	}
}

func TestContextOptions(t *testing.T) {
	cfg := DefaultContextConfig()

	WithDataStackSize(2048)(&cfg)
	if cfg.DataStackSize != 2048 {
		t.Errorf("expected DataStackSize 2048, got %d", cfg.DataStackSize)
	}

	WithEntityStackSize(128)(&cfg)
	if cfg.EntityStackSize != 128 {
		t.Errorf("expected EntityStackSize 128, got %d", cfg.EntityStackSize)
	}
}

func TestApplyOptions(t *testing.T) {
	cfg := DefaultContextConfig()
	opts := []ContextOption{
		WithDataStackSize(512),
		WithEntityStackSize(32),
	}

	cfg.ApplyOptions(opts)

	if cfg.DataStackSize != 512 {
		t.Errorf("expected DataStackSize 512, got %d", cfg.DataStackSize)
	}
	if cfg.EntityStackSize != 32 {
		t.Errorf("expected EntityStackSize 32, got %d", cfg.EntityStackSize)
	}
}

func TestRuntimeError(t *testing.T) {
	err := &RuntimeError{Code: 1, Message: "test error"}

	if err.Error() != "test error" {
		t.Errorf("expected 'test error', got '%s'", err.Error())
	}

	errWithCause := err.WithCause(ErrStackOverflow)
	if errWithCause.Error() != "test error: stack overflow" {
		t.Errorf("expected 'test error: stack overflow', got '%s'", errWithCause.Error())
	}

	if errWithCause.Unwrap() != ErrStackOverflow {
		t.Error("expected Unwrap to return cause")
	}
}

func TestNewRuntimeError(t *testing.T) {
	err := NewRuntimeError(99, "custom error")

	if err.Code != 99 {
		t.Errorf("expected code 99, got %d", err.Code)
	}
	if err.Message != "custom error" {
		t.Errorf("expected 'custom error', got '%s'", err.Message)
	}
}

func TestErrorConstants(t *testing.T) {
	errors := []struct {
		err  *RuntimeError
		code int
	}{
		{ErrStackOverflow, 1},
		{ErrStackUnderflow, 2},
		{ErrTypeMismatch, 3},
		{ErrDivisionByZero, 4},
		{ErrOutOfMemory, 5},
		{ErrInvalidOpcode, 6},
		{ErrIndexOutOfBounds, 7},
		{ErrNameNotFound, 8},
		{ErrEntityStackUnderflow, 9},
		{ErrInvalidEntity, 10},
		{ErrContextClosed, 11},
		{ErrRuntimeClosed, 12},
	}

	for _, tc := range errors {
		if tc.err.Code != tc.code {
			t.Errorf("expected code %d for %s, got %d", tc.code, tc.err.Message, tc.err.Code)
		}
	}
}
