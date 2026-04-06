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

package dtrules_test

import (
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// =============================================================================
// BigInt End-to-End Integration Tests
//
// These tests verify the full pipeline: EDD loading -> decision table execution
// -> BigInt arithmetic with very large numbers that exceed int64 range.
// =============================================================================

// TestBigIntEndToEndPipeline tests the full pipeline with BigInt support:
// 1. Load an EDD with bigint fields
// 2. Create a decision table that performs bigint arithmetic
// 3. Execute the decision table
// 4. Verify the results
func TestBigIntEndToEndPipeline(t *testing.T) {
	// Create EDD with bigint fields
	eddXML := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="budget" access="rw" comment="Budget entity with large number support">
		<field name="budget" type="entity" subtype="" access="r" input="" default_value="" comment="Self Reference"></field>
		<field name="supply_limit" type="bigint" subtype="" access="rw" input="" default_value="50000000000000000000" comment="Total supply limit (50 * 10^18)"></field>
		<field name="amount_issued" type="bigint" subtype="" access="rw" input="" default_value="0" comment="Amount already issued"></field>
		<field name="remaining" type="bigint" subtype="" access="rw" input="" default_value="0" comment="Remaining supply"></field>
	</entity>
</entity_data_dictionary>`

	// Create decision table that calculates: remaining = supply_limit - amount_issued
	// Using postfix syntax with xdef for assignment
	dtXML := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
	<decision_table>
		<table_name>Calculate_Remaining</table_name>
		<attribute_fields>
			<TABLE_NUMBER>1000</TABLE_NUMBER>
			<Type>FIRST</Type>
		</attribute_fields>
		<contexts></contexts>
		<initial_actions></initial_actions>
		<conditions>
			<condition_details>
				<condition_number>1</condition_number>
				<condition_dsl>Always execute</condition_dsl>
				<condition_postfix>true</condition_postfix>
				<condition_column column_number="1" column_value="Y"></condition_column>
			</condition_details>
		</conditions>
		<actions>
			<action_details>
				<action_number>1</action_number>
				<action_dsl>Calculate remaining = supply_limit - amount_issued</action_dsl>
				<action_postfix>budget.supply_limit budget.amount_issued b- /budget.remaining xdef</action_postfix>
				<action_column column_number="1" column_value="X"></action_column>
			</action_details>
		</actions>
		<policy_statements></policy_statements>
	</decision_table>
</decision_tables>`

	// Load EDD
	rs := session.NewRuleSet("bigint_test")
	err := rs.LoadEDD(strings.NewReader(eddXML))
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}
	t.Log("EDD loaded successfully")

	// Verify bigint entity was created
	names := rs.GetEntityNames()
	found := false
	for _, name := range names {
		if name.StringValue() == "budget" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("Expected 'budget' entity to be created")
	}

	// Load decision tables
	err = rs.LoadDecisionTables(strings.NewReader(dtXML))
	if err != nil {
		t.Fatalf("Failed to load decision tables: %v", err)
	}
	t.Log("Decision tables loaded successfully")

	// Verify decision table was created
	dtNames := rs.GetDecisionTableNames()
	foundDT := false
	for _, name := range dtNames {
		if name.StringValue() == "Calculate_Remaining" {
			foundDT = true
			break
		}
	}
	if !foundDT {
		t.Fatal("Expected 'Calculate_Remaining' decision table to be created")
	}

	// Create session
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	rsession := sess.(*session.RSession)

	// Create budget entity
	budgetEntity, err := rsession.CreateEntity(dtrules.GetRName("budget"))
	if err != nil {
		t.Fatalf("Failed to create budget entity: %v", err)
	}

	// Verify default values were set correctly
	supplyLimit, err := budgetEntity.Get(dtrules.GetRName("supply_limit"))
	if err != nil {
		t.Fatalf("Failed to get supply_limit: %v", err)
	}
	slBigInt, err := supplyLimit.RBigIntValue()
	if err != nil {
		t.Fatalf("supply_limit is not a BigInt: %v", err)
	}
	if slBigInt.StringValue() != "50000000000000000000" {
		t.Errorf("Expected supply_limit default '50000000000000000000', got '%s'", slBigInt.StringValue())
	}
	t.Logf("supply_limit default value: %s", slBigInt.StringValue())

	// Set amount_issued to a large value (10 * 10^18)
	amountIssued, err := dtrules.GetRBigIntFromString("10000000000000000000")
	if err != nil {
		t.Fatalf("Failed to create BigInt for amount_issued: %v", err)
	}
	err = budgetEntity.Put(dtrules.GetRName("amount_issued"), amountIssued)
	if err != nil {
		t.Fatalf("Failed to set amount_issued: %v", err)
	}
	t.Logf("amount_issued set to: %s", amountIssued.StringValue())

	// Push entity onto stack for decision table execution
	state := sess.GetState()
	state.EntityPush(budgetEntity)

	// Get and execute the decision table
	factory := sess.GetEntityFactory()
	dtObj, err := factory.GetDecisionTable(dtrules.GetRName("Calculate_Remaining"))
	if err != nil {
		t.Fatalf("Failed to get decision table: %v", err)
	}
	if dtObj == nil {
		t.Fatal("Decision table 'Calculate_Remaining' not found")
	}

	err = dtObj.Execute(state)
	if err != nil {
		t.Fatalf("Decision table execution failed: %v", err)
	}
	t.Log("Decision table executed successfully")

	// Verify remaining was calculated correctly: 50*10^18 - 10*10^18 = 40*10^18
	remaining, err := budgetEntity.Get(dtrules.GetRName("remaining"))
	if err != nil {
		t.Fatalf("Failed to get remaining: %v", err)
	}
	remainingBigInt, err := remaining.RBigIntValue()
	if err != nil {
		t.Fatalf("remaining is not a BigInt: %v", err)
	}

	expectedRemaining := "40000000000000000000"
	if remainingBigInt.StringValue() != expectedRemaining {
		t.Errorf("Expected remaining '%s', got '%s'", expectedRemaining, remainingBigInt.StringValue())
	} else {
		t.Logf("remaining correctly calculated: %s", remainingBigInt.StringValue())
	}

	// Pop the entity
	state.EntityPop()
}

// TestBigIntEDDLoading tests that EDD correctly loads bigint field types with defaults
func TestBigIntEDDLoading(t *testing.T) {
	eddXML := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="token" access="rw" comment="">
		<field name="token" type="entity" subtype="" access="r" input="" default_value="" comment="Self Reference"></field>
		<field name="total_supply" type="bigint" subtype="" access="rw" input="" default_value="1000000000000000000000000" comment="Total token supply"></field>
		<field name="burned" type="bigint" subtype="" access="rw" input="" default_value="0" comment="Amount burned"></field>
		<field name="circulating" type="bigint" subtype="" access="rw" input="" default_value="0" comment="Circulating supply"></field>
	</entity>
</entity_data_dictionary>`

	rs := session.NewRuleSet("bigint_edd_test")
	err := rs.LoadEDD(strings.NewReader(eddXML))
	if err != nil {
		t.Fatalf("Failed to load EDD with bigint fields: %v", err)
	}

	// Create session and entity
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	rsession := sess.(*session.RSession)

	entity, err := rsession.CreateEntity(dtrules.GetRName("token"))
	if err != nil {
		t.Fatalf("Failed to create token entity: %v", err)
	}

	// Verify total_supply has the large default value
	totalSupply, err := entity.Get(dtrules.GetRName("total_supply"))
	if err != nil {
		t.Fatalf("Failed to get total_supply: %v", err)
	}

	bi, err := totalSupply.RBigIntValue()
	if err != nil {
		t.Fatalf("total_supply is not a BigInt: %v", err)
	}

	expected := "1000000000000000000000000"
	if bi.StringValue() != expected {
		t.Errorf("Expected total_supply '%s', got '%s'", expected, bi.StringValue())
	}

	// Verify the value is larger than int64 max
	_, err = bi.LongValue()
	if err == nil {
		t.Error("Expected LongValue() to fail for value larger than int64 max")
	}
	t.Logf("Confirmed: total_supply (%s) exceeds int64 range", bi.StringValue())
}

// TestBigIntArithmeticOperators tests bigint arithmetic operators in decision tables
func TestBigIntArithmeticOperators(t *testing.T) {
	eddXML := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="calc" access="rw" comment="">
		<field name="calc" type="entity" subtype="" access="r" input="" default_value="" comment="Self Reference"></field>
		<field name="a" type="bigint" subtype="" access="rw" input="" default_value="0" comment=""></field>
		<field name="b" type="bigint" subtype="" access="rw" input="" default_value="0" comment=""></field>
		<field name="sum" type="bigint" subtype="" access="rw" input="" default_value="0" comment=""></field>
		<field name="diff" type="bigint" subtype="" access="rw" input="" default_value="0" comment=""></field>
		<field name="product" type="bigint" subtype="" access="rw" input="" default_value="0" comment=""></field>
		<field name="quotient" type="bigint" subtype="" access="rw" input="" default_value="0" comment=""></field>
	</entity>
</entity_data_dictionary>`

	// Decision table that tests all arithmetic operators
	dtXML := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
	<decision_table>
		<table_name>BigInt_Arithmetic</table_name>
		<attribute_fields>
			<TABLE_NUMBER>2000</TABLE_NUMBER>
			<Type>FIRST</Type>
		</attribute_fields>
		<contexts></contexts>
		<initial_actions></initial_actions>
		<conditions>
			<condition_details>
				<condition_number>1</condition_number>
				<condition_dsl>Always</condition_dsl>
				<condition_postfix>true</condition_postfix>
				<condition_column column_number="1" column_value="Y"></condition_column>
			</condition_details>
		</conditions>
		<actions>
			<action_details>
				<action_number>1</action_number>
				<action_dsl>sum = a + b</action_dsl>
				<action_postfix>calc.a calc.b b+ /calc.sum xdef</action_postfix>
				<action_column column_number="1" column_value="X"></action_column>
			</action_details>
			<action_details>
				<action_number>2</action_number>
				<action_dsl>diff = a - b</action_dsl>
				<action_postfix>calc.a calc.b b- /calc.diff xdef</action_postfix>
				<action_column column_number="1" column_value="X"></action_column>
			</action_details>
			<action_details>
				<action_number>3</action_number>
				<action_dsl>product = a * b</action_dsl>
				<action_postfix>calc.a calc.b b* /calc.product xdef</action_postfix>
				<action_column column_number="1" column_value="X"></action_column>
			</action_details>
			<action_details>
				<action_number>4</action_number>
				<action_dsl>quotient = a / b</action_dsl>
				<action_postfix>calc.a calc.b b/ /calc.quotient xdef</action_postfix>
				<action_column column_number="1" column_value="X"></action_column>
			</action_details>
		</actions>
		<policy_statements></policy_statements>
	</decision_table>
</decision_tables>`

	rs := session.NewRuleSet("bigint_arithmetic_test")
	if err := rs.LoadEDD(strings.NewReader(eddXML)); err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}
	if err := rs.LoadDecisionTables(strings.NewReader(dtXML)); err != nil {
		t.Fatalf("Failed to load DT: %v", err)
	}

	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	rsession := sess.(*session.RSession)

	entity, err := rsession.CreateEntity(dtrules.GetRName("calc"))
	if err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	// Set large values: a = 10^30, b = 10^20
	aValue, _ := dtrules.GetRBigIntFromString("1000000000000000000000000000000")
	bValue, _ := dtrules.GetRBigIntFromString("100000000000000000000")
	entity.Put(dtrules.GetRName("a"), aValue)
	entity.Put(dtrules.GetRName("b"), bValue)

	// Execute
	state := sess.GetState()
	state.EntityPush(entity)
	factory := sess.GetEntityFactory()
	dtObj, _ := factory.GetDecisionTable(dtrules.GetRName("BigInt_Arithmetic"))
	err = dtObj.Execute(state)
	if err != nil {
		t.Fatalf("DT execution failed: %v", err)
	}

	// Verify results
	// a = 10^30 = 1000000000000000000000000000000
	// b = 10^20 = 100000000000000000000
	tests := []struct {
		field    string
		expected string
	}{
		{"sum", "1000000000100000000000000000000"},     // 10^30 + 10^20
		{"diff", "999999999900000000000000000000"},     // 10^30 - 10^20
		{"product", "100000000000000000000000000000000000000000000000000"}, // 10^30 * 10^20 = 10^50
		{"quotient", "10000000000"},                    // 10^30 / 10^20 = 10^10
	}

	for _, tt := range tests {
		value, err := entity.Get(dtrules.GetRName(tt.field))
		if err != nil {
			t.Errorf("Failed to get %s: %v", tt.field, err)
			continue
		}
		bi, err := value.RBigIntValue()
		if err != nil {
			t.Errorf("%s is not a BigInt: %v", tt.field, err)
			continue
		}
		if bi.StringValue() != tt.expected {
			t.Errorf("%s: expected '%s', got '%s'", tt.field, tt.expected, bi.StringValue())
		} else {
			t.Logf("%s = %s (correct)", tt.field, bi.StringValue())
		}
	}

	state.EntityPop()
}

// TestBigIntComparisonOperators tests bigint comparison operators in decision tables
func TestBigIntComparisonOperators(t *testing.T) {
	eddXML := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="compare" access="rw" comment="">
		<field name="compare" type="entity" subtype="" access="r" input="" default_value="" comment="Self Reference"></field>
		<field name="balance" type="bigint" subtype="" access="rw" input="" default_value="0" comment=""></field>
		<field name="threshold" type="bigint" subtype="" access="rw" input="" default_value="0" comment=""></field>
		<field name="is_above_threshold" type="boolean" subtype="" access="rw" input="" default_value="false" comment=""></field>
		<field name="is_below_threshold" type="boolean" subtype="" access="rw" input="" default_value="false" comment=""></field>
		<field name="is_equal_threshold" type="boolean" subtype="" access="rw" input="" default_value="false" comment=""></field>
	</entity>
</entity_data_dictionary>`

	dtXML := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
	<decision_table>
		<table_name>BigInt_Compare</table_name>
		<attribute_fields>
			<TABLE_NUMBER>3000</TABLE_NUMBER>
			<Type>FIRST</Type>
		</attribute_fields>
		<contexts></contexts>
		<initial_actions></initial_actions>
		<conditions>
			<condition_details>
				<condition_number>1</condition_number>
				<condition_dsl>Always</condition_dsl>
				<condition_postfix>true</condition_postfix>
				<condition_column column_number="1" column_value="Y"></condition_column>
			</condition_details>
		</conditions>
		<actions>
			<action_details>
				<action_number>1</action_number>
				<action_dsl>Check balance > threshold</action_dsl>
				<action_postfix>compare.balance compare.threshold b> /compare.is_above_threshold xdef</action_postfix>
				<action_column column_number="1" column_value="X"></action_column>
			</action_details>
			<action_details>
				<action_number>2</action_number>
				<action_dsl>Check balance &lt; threshold</action_dsl>
				<action_postfix>compare.balance compare.threshold b&lt; /compare.is_below_threshold xdef</action_postfix>
				<action_column column_number="1" column_value="X"></action_column>
			</action_details>
			<action_details>
				<action_number>3</action_number>
				<action_dsl>Check balance == threshold</action_dsl>
				<action_postfix>compare.balance compare.threshold b== /compare.is_equal_threshold xdef</action_postfix>
				<action_column column_number="1" column_value="X"></action_column>
			</action_details>
		</actions>
		<policy_statements></policy_statements>
	</decision_table>
</decision_tables>`

	rs := session.NewRuleSet("bigint_compare_test")
	if err := rs.LoadEDD(strings.NewReader(eddXML)); err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}
	if err := rs.LoadDecisionTables(strings.NewReader(dtXML)); err != nil {
		t.Fatalf("Failed to load DT: %v", err)
	}

	tests := []struct {
		name      string
		balance   string
		threshold string
		above     bool
		below     bool
		equal     bool
	}{
		{
			name:      "balance > threshold",
			balance:   "999999999999999999999999999",
			threshold: "100000000000000000000000000",
			above:     true,
			below:     false,
			equal:     false,
		},
		{
			name:      "balance < threshold",
			balance:   "100000000000000000000000000",
			threshold: "999999999999999999999999999",
			above:     false,
			below:     true,
			equal:     false,
		},
		{
			name:      "balance == threshold",
			balance:   "123456789012345678901234567890",
			threshold: "123456789012345678901234567890",
			above:     false,
			below:     false,
			equal:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess, _ := rs.NewSession()
			rsession := sess.(*session.RSession)
			entity, _ := rsession.CreateEntity(dtrules.GetRName("compare"))

			balance, _ := dtrules.GetRBigIntFromString(tt.balance)
			threshold, _ := dtrules.GetRBigIntFromString(tt.threshold)
			entity.Put(dtrules.GetRName("balance"), balance)
			entity.Put(dtrules.GetRName("threshold"), threshold)

			state := sess.GetState()
			state.EntityPush(entity)
			factory := sess.GetEntityFactory()
			dtObj, _ := factory.GetDecisionTable(dtrules.GetRName("BigInt_Compare"))
			if err := dtObj.Execute(state); err != nil {
				t.Fatalf("DT execution failed: %v", err)
			}

			// Check results
			aboveVal, _ := entity.Get(dtrules.GetRName("is_above_threshold"))
			aboveBool, _ := aboveVal.BooleanValue()
			if aboveBool != tt.above {
				t.Errorf("is_above_threshold: expected %v, got %v", tt.above, aboveBool)
			}

			belowVal, _ := entity.Get(dtrules.GetRName("is_below_threshold"))
			belowBool, _ := belowVal.BooleanValue()
			if belowBool != tt.below {
				t.Errorf("is_below_threshold: expected %v, got %v", tt.below, belowBool)
			}

			equalVal, _ := entity.Get(dtrules.GetRName("is_equal_threshold"))
			equalBool, _ := equalVal.BooleanValue()
			if equalBool != tt.equal {
				t.Errorf("is_equal_threshold: expected %v, got %v", tt.equal, equalBool)
			}

			state.EntityPop()
		})
	}
}

// TestBigIntTypeConversion tests the cvbi (convert to bigint) operator
func TestBigIntTypeConversion(t *testing.T) {
	eddXML := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="convert" access="rw" comment="">
		<field name="convert" type="entity" subtype="" access="r" input="" default_value="" comment="Self Reference"></field>
		<field name="int_value" type="integer" subtype="" access="rw" input="" default_value="12345" comment=""></field>
		<field name="str_value" type="string" subtype="" access="rw" input="" default_value="99999999999999999999" comment=""></field>
		<field name="bigint_from_int" type="bigint" subtype="" access="rw" input="" default_value="0" comment=""></field>
		<field name="bigint_from_str" type="bigint" subtype="" access="rw" input="" default_value="0" comment=""></field>
	</entity>
</entity_data_dictionary>`

	dtXML := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
	<decision_table>
		<table_name>Convert_To_BigInt</table_name>
		<attribute_fields>
			<TABLE_NUMBER>4000</TABLE_NUMBER>
			<Type>FIRST</Type>
		</attribute_fields>
		<contexts></contexts>
		<initial_actions></initial_actions>
		<conditions>
			<condition_details>
				<condition_number>1</condition_number>
				<condition_dsl>Always</condition_dsl>
				<condition_postfix>true</condition_postfix>
				<condition_column column_number="1" column_value="Y"></condition_column>
			</condition_details>
		</conditions>
		<actions>
			<action_details>
				<action_number>1</action_number>
				<action_dsl>Convert int to bigint</action_dsl>
				<action_postfix>convert.int_value cvbi /convert.bigint_from_int xdef</action_postfix>
				<action_column column_number="1" column_value="X"></action_column>
			</action_details>
			<action_details>
				<action_number>2</action_number>
				<action_dsl>Convert string to bigint</action_dsl>
				<action_postfix>convert.str_value cvbi /convert.bigint_from_str xdef</action_postfix>
				<action_column column_number="1" column_value="X"></action_column>
			</action_details>
		</actions>
		<policy_statements></policy_statements>
	</decision_table>
</decision_tables>`

	rs := session.NewRuleSet("bigint_convert_test")
	if err := rs.LoadEDD(strings.NewReader(eddXML)); err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}
	if err := rs.LoadDecisionTables(strings.NewReader(dtXML)); err != nil {
		t.Fatalf("Failed to load DT: %v", err)
	}

	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	rsession := sess.(*session.RSession)

	entity, err := rsession.CreateEntity(dtrules.GetRName("convert"))
	if err != nil {
		t.Fatalf("Failed to create entity: %v", err)
	}

	state := sess.GetState()
	state.EntityPush(entity)
	factory := sess.GetEntityFactory()
	dtObj, _ := factory.GetDecisionTable(dtrules.GetRName("Convert_To_BigInt"))
	if err := dtObj.Execute(state); err != nil {
		t.Fatalf("DT execution failed: %v", err)
	}

	// Verify conversions
	fromInt, _ := entity.Get(dtrules.GetRName("bigint_from_int"))
	fromIntBI, err := fromInt.RBigIntValue()
	if err != nil {
		t.Fatalf("bigint_from_int is not a BigInt: %v", err)
	}
	if fromIntBI.StringValue() != "12345" {
		t.Errorf("bigint_from_int: expected '12345', got '%s'", fromIntBI.StringValue())
	}

	fromStr, _ := entity.Get(dtrules.GetRName("bigint_from_str"))
	fromStrBI, err := fromStr.RBigIntValue()
	if err != nil {
		t.Fatalf("bigint_from_str is not a BigInt: %v", err)
	}
	if fromStrBI.StringValue() != "99999999999999999999" {
		t.Errorf("bigint_from_str: expected '99999999999999999999', got '%s'", fromStrBI.StringValue())
	}

	// Verify that the string conversion gave a value larger than int64
	_, err = fromStrBI.LongValue()
	if err == nil {
		t.Error("Expected LongValue() to fail for bigint_from_str")
	}

	state.EntityPop()
}

// TestBigIntPipelineWithSession tests using Session.Execute() method
func TestBigIntPipelineWithSession(t *testing.T) {
	eddXML := `<?xml version="1.0" encoding="UTF-8"?>
<entity_data_dictionary version="2">
	<entity name="account" access="rw" comment="">
		<field name="account" type="entity" subtype="" access="r" input="" default_value="" comment="Self Reference"></field>
		<field name="balance" type="bigint" subtype="" access="rw" input="" default_value="0" comment=""></field>
		<field name="deposit" type="bigint" subtype="" access="rw" input="" default_value="0" comment=""></field>
		<field name="new_balance" type="bigint" subtype="" access="rw" input="" default_value="0" comment=""></field>
	</entity>
</entity_data_dictionary>`

	dtXML := `<?xml version="1.0" encoding="UTF-8"?>
<decision_tables>
	<decision_table>
		<table_name>Process_Deposit</table_name>
		<attribute_fields>
			<TABLE_NUMBER>5000</TABLE_NUMBER>
			<Type>FIRST</Type>
		</attribute_fields>
		<contexts></contexts>
		<initial_actions></initial_actions>
		<conditions>
			<condition_details>
				<condition_number>1</condition_number>
				<condition_dsl>Always</condition_dsl>
				<condition_postfix>true</condition_postfix>
				<condition_column column_number="1" column_value="Y"></condition_column>
			</condition_details>
		</conditions>
		<actions>
			<action_details>
				<action_number>1</action_number>
				<action_dsl>new_balance = balance + deposit</action_dsl>
				<action_postfix>account.balance account.deposit b+ /account.new_balance xdef</action_postfix>
				<action_column column_number="1" column_value="X"></action_column>
			</action_details>
		</actions>
		<policy_statements></policy_statements>
	</decision_table>
</decision_tables>`

	rs := session.NewRuleSet("session_execute_test")
	rs.LoadEDD(strings.NewReader(eddXML))
	rs.LoadDecisionTables(strings.NewReader(dtXML))

	sess, _ := rs.NewSession()
	rsession := sess.(*session.RSession)

	// Create and populate entity
	entity, _ := rsession.CreateEntity(dtrules.GetRName("account"))
	balance, _ := dtrules.GetRBigIntFromString("50000000000000000000")
	deposit, _ := dtrules.GetRBigIntFromString("25000000000000000000")
	entity.Put(dtrules.GetRName("balance"), balance)
	entity.Put(dtrules.GetRName("deposit"), deposit)

	// Push entity and execute using Session.Execute()
	state := sess.GetState()
	state.EntityPush(entity)

	err := rsession.Execute("Process_Deposit")
	if err != nil {
		t.Fatalf("Session.Execute failed: %v", err)
	}

	// Verify result: 50*10^18 + 25*10^18 = 75*10^18
	newBalance, _ := entity.Get(dtrules.GetRName("new_balance"))
	bi, _ := newBalance.RBigIntValue()
	expected := "75000000000000000000"
	if bi.StringValue() != expected {
		t.Errorf("new_balance: expected '%s', got '%s'", expected, bi.StringValue())
	} else {
		t.Logf("Session.Execute completed successfully: new_balance = %s", bi.StringValue())
	}

	state.EntityPop()
}
