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

/*
Package sdk provides an embeddable DTRules engine for Go applications.

# Overview

The SDK allows applications to embed and execute decision tables.
Rules can be loaded from the filesystem or embedded directly into
the binary using Go's embed package.

# Basic Usage

Load rules from a directory:

	engine, err := sdk.NewEngine("MyRules", sdk.WithDirectory("./rules"))
	if err != nil {
	    log.Fatal(err)
	}

	ctx := engine.NewContext()
	ctx.SetEntity("input", "income", 50000)
	ctx.SetEntity("input", "dependents", 2)

	result, err := engine.Execute("Calculate_Eligibility", ctx)
	if err != nil {
	    log.Fatal(err)
	}

	eligible := result.GetBool("eligible")

# Embedding Rules

Use Go's embed package to include rules in the binary:

	import "embed"

	//go:embed rules/*.xml
	var rulesFS embed.FS

	func main() {
	    engine, err := sdk.NewEngine("MyRules", sdk.WithFS(rulesFS, "rules"))
	    if err != nil {
	        log.Fatal(err)
	    }
	    // Use engine...
	}

# Concurrent Execution

The Engine is safe for concurrent use. Each call to NewContext() creates
an independent execution context:

	engine, _ := sdk.NewEngine("MyRules", sdk.WithDirectory("./rules"))

	// Execute concurrently
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
	    wg.Add(1)
	    go func(n int) {
	        defer wg.Done()
	        ctx := engine.NewContext()
	        ctx.Set("value", n)
	        result, _ := engine.Execute("Process", ctx)
	        _ = result.GetInt("output")
	    }(i)
	}
	wg.Wait()

# Entity Data

Set values on specific entities:

	ctx := engine.NewContext()

	// Set values on the "case" entity
	ctx.SetEntity("case", "income", 75000)
	ctx.SetEntity("case", "age", 42)
	ctx.SetEntity("case", "married", true)

	// Set values on the "address" entity
	ctx.SetEntity("address", "state", "CO")
	ctx.SetEntity("address", "zip", "80202")

# Entity Arrays (for iteration)

Use SetEntityArray to provide arrays of entities that decision tables
can iterate over using forall:

	ctx := engine.NewContext()

	// Set an array of staking accounts for iteration
	ctx.SetEntityArray("accounts", []map[string]interface{}{
	    {"balance": "1000", "identity_url": "https://example.com/1"},
	    {"balance": "2000", "identity_url": "https://example.com/2"},
	    {"balance": "500", "identity_url": "https://example.com/3"},
	})

	// Execute table that iterates over the array
	result, err := engine.Execute("Calculate_Rewards", ctx)

The array name should be the plural form of the entity name.
The SDK automatically derives "account" from "accounts" when
creating entities. Common pluralization rules are handled:
accounts -> account, entities -> entity, boxes -> box.

Decision tables can iterate using forall in contexts:

	for all accounts

Or using forall in action expressions with postfix:

	{ account.balance /total b+ /total xdef } accounts forall

# Loading Options

Multiple loading options can be combined:

	// Load from embedded FS
	engine, _ := sdk.NewEngine("Rules",
	    sdk.WithFS(rulesFS, "rules"),
	)

	// Load from directory
	engine, _ := sdk.NewEngine("Rules",
	    sdk.WithDirectory("./rules"),
	)

	// Load from explicit readers
	eddFile, _ := os.Open("rules/edd.xml")
	dtFile, _ := os.Open("rules/dt.xml")
	engine, _ := sdk.NewEngine("Rules",
	    sdk.WithEDD(eddFile),
	    sdk.WithDT(dtFile),
	)

# File Naming Convention

The SDK automatically detects file types based on naming:

  - EDD files: *_edd.xml, edd_*.xml, edd.xml
  - DT files: *_dt.xml, dt_*.xml, decisiontables.xml

Files are loaded in order: EDD files first, then DT files.
*/
package sdk
