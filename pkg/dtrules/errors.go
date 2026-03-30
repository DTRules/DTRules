// Copyright 2004-2011 DTRules.com, Inc.
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

package dtrules

import (
	"fmt"
)

// RulesError represents an error in the rules engine.
type RulesError struct {
	ErrorType string
	Location  string
	Message   string
	PostFix   string
	Cause     error
}

// NewRulesError creates a new RulesError.
func NewRulesError(errorType, location, message string) *RulesError {
	return &RulesError{
		ErrorType: errorType,
		Location:  location,
		Message:   message,
	}
}

// NewRulesErrorWithCause creates a new RulesError with a cause.
func NewRulesErrorWithCause(errorType, location, message string, cause error) *RulesError {
	return &RulesError{
		ErrorType: errorType,
		Location:  location,
		Message:   message,
		Cause:     cause,
	}
}

// Error implements the error interface.
func (e *RulesError) Error() string {
	result := fmt.Sprintf("[%s] %s: %s", e.ErrorType, e.Location, e.Message)
	if e.PostFix != "" {
		result += "\nPostFix: " + e.PostFix
	}
	if e.Cause != nil {
		result += "\nCaused by: " + e.Cause.Error()
	}
	return result
}

// Unwrap returns the underlying cause.
func (e *RulesError) Unwrap() error {
	return e.Cause
}

// SetPostFix sets the postfix context for this error.
func (e *RulesError) SetPostFix(postfix string) {
	e.PostFix = postfix
}

// Common error constructors

// ConversionError creates a conversion error.
func ConversionError(location, message string) *RulesError {
	return NewRulesError("Conversion Error", location, message)
}

// UndefinedError creates an undefined error.
func UndefinedError(location, message string) *RulesError {
	return NewRulesError("Undefined", location, message)
}

// TypeCheckError creates a type check error.
func TypeCheckError(location, message string) *RulesError {
	return NewRulesError("typecheck", location, message)
}

// StackUnderflowError creates a stack underflow error.
func StackUnderflowError(location string) *RulesError {
	return NewRulesError("Stack Underflow", location, "Stack is empty")
}

// StackOverflowError creates a stack overflow error.
func StackOverflowError(location, message string) *RulesError {
	return NewRulesError("Stack Overflow", location, message)
}

// OutOfBoundsError creates an out of bounds error.
func OutOfBoundsError(location, message string) *RulesError {
	return NewRulesError("OutOfBounds", location, message)
}

// ParsingError creates a parsing error.
func ParsingError(location, message string) *RulesError {
	return NewRulesError("Parsing Error", location, message)
}
