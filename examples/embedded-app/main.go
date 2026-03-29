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

// Example: Embedding DTRules into an application
//
// This demonstrates how to embed decision tables into a single binary.
// During development, rules are files on disk. At build time, they're
// compiled into the binary.
//
// Development:
//   - Edit rules/xml/*.xml or rules/excel/*.xlsx
//   - Run: dtrules sync import (sync Excel to XML)
//   - Run: go test ./... (test with files)
//
// Production:
//   - Build: go build -o myapp .
//   - Deploy: just the binary, no files needed
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/DTRules/DTRules/pkg/dtrules/sdk"
)

// Embed rules at compile time - these become part of the binary
//
//go:embed rules/xml/*.xml
var rulesFS embed.FS

// StakingEngine wraps DTRules for staking-specific operations
type StakingEngine struct {
	engine *sdk.Engine
}

// NewStakingEngine creates the staking engine with embedded rules
func NewStakingEngine() (*StakingEngine, error) {
	engine, err := sdk.NewEngine("Staking", sdk.WithFS(rulesFS, "rules/xml"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize rules engine: %w", err)
	}

	return &StakingEngine{engine: engine}, nil
}

// StakeRequest represents a staking request
type StakeRequest struct {
	UserID       string  `json:"user_id"`
	Amount       float64 `json:"amount"`
	Duration     int     `json:"duration_days"`
	TokenType    string  `json:"token_type"`
	IsValidator  bool    `json:"is_validator"`
}

// StakeResult represents the staking calculation result
type StakeResult struct {
	Eligible     bool    `json:"eligible"`
	APY          float64 `json:"apy"`
	EstReward    float64 `json:"estimated_reward"`
	LockupDays   int     `json:"lockup_days"`
	Tier         string  `json:"tier"`
	Reason       string  `json:"reason,omitempty"`
}

// CalculateStake runs the staking decision tables
func (s *StakingEngine) CalculateStake(req *StakeRequest) (*StakeResult, error) {
	ctx := s.engine.NewContext()

	// Set input values
	ctx.SetEntity("stake_request", "amount", req.Amount)
	ctx.SetEntity("stake_request", "duration_days", req.Duration)
	ctx.SetEntity("stake_request", "token_type", req.TokenType)
	ctx.SetEntity("stake_request", "is_validator", req.IsValidator)

	// Execute decision table
	result, err := s.engine.Execute("Calculate_Staking_Reward", ctx)
	if err != nil {
		return nil, fmt.Errorf("rule execution failed: %w", err)
	}

	// Extract results
	return &StakeResult{
		Eligible:   result.GetBool("eligible"),
		APY:        result.GetFloat("apy"),
		EstReward:  result.GetFloat("estimated_reward"),
		LockupDays: int(result.GetInt("lockup_days")),
		Tier:       result.GetString("tier"),
		Reason:     result.GetString("reason"),
	}, nil
}

// ListAvailableTables returns the decision tables available
func (s *StakingEngine) ListAvailableTables() []string {
	return s.engine.ListTables()
}

func main() {
	// Initialize the staking engine (rules are embedded in binary)
	staking, err := NewStakingEngine()
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	fmt.Println("Staking Engine initialized")
	fmt.Printf("Available tables: %v\n", staking.ListAvailableTables())

	// Example: HTTP API
	http.HandleFunc("/stake/calculate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}

		var req StakeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		result, err := staking.CalculateStake(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	http.HandleFunc("/stake/tables", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(staking.ListAvailableTables())
	})

	fmt.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
