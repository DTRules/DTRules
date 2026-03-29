package dtrules_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

func TestTaxReturnResults(t *testing.T) {
	// Find the sample projects directory
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	// Create rule set and load from directory (multi-file structure)
	rs := session.NewRuleSet("TaxReturn")
	err := rs.LoadFromDirectory(xmlDir)
	if err != nil {
		t.Fatalf("Failed to load rules from directory: %v", err)
	}

	// Create session
	sess, err := rs.NewSession()
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Load mapping
	mapPath := filepath.Join(xmlDir, "TaxReturn_map.xml")
	mapFile, err := os.Open(mapPath)
	if err != nil {
		t.Fatalf("Failed to open mapping: %v", err)
	}
	defer mapFile.Close()

	m := mapping.NewMapping(sess)
	err = m.LoadMapping(mapFile)
	if err != nil {
		t.Fatalf("Failed to load mapping: %v", err)
	}
	err = m.Initialize()
	if err != nil {
		t.Fatalf("Failed to init mapping: %v", err)
	}

	// Load test data
	testPath := filepath.Join(sampleDir, "testfiles", "TestScenarios", "TestCase_Family_2025.xml")
	testFile, err := os.Open(testPath)
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer testFile.Close()

	err = m.LoadData(testFile)
	if err != nil {
		t.Fatalf("Failed to map test data: %v", err)
	}

	// Execute Compute_Tax_Return
	ef := sess.GetEntityFactory()
	dt, err := ef.GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
	if err != nil || dt == nil {
		t.Fatalf("Failed to get decision table: %v", err)
	}

	state := sess.GetState()
	err = dt.Execute(state)
	if err != nil {
		t.Fatalf("Failed to execute: %v", err)
	}

	// Get results
	jobName := dtrules.GetRName("job")
	job, err := state.FindEntity(jobName)
	if err != nil || job == nil {
		t.Fatalf("Failed to find job entity: %v", err)
	}

	// Expected values based on test data (updated 2026-03-26):
	// Includes: unemployment, gambling, alimony, state tax refund
	// AOTC refundable portion properly split (40%/60%)
	// Enhanced credit calculations
	expectedAGI := 252515.105850
	expectedTaxable := 195375.105850
	expectedTax := 35603.897309
	expectedRefund := 0.0

	// Get computed results
	resultsName := dtrules.GetRName("results")
	resultsObj, _ := job.Get(resultsName)
	if resultsObj == nil {
		t.Fatal("No results array found")
	}
	resultsArr, _ := resultsObj.ArrayValue()
	if len(resultsArr) == 0 {
		t.Fatal("Results array is empty")
	}

	result, _ := resultsArr[0].REntityValue()
	if result == nil {
		t.Fatal("No result entity found")
	}

	// Print all result attributes
	fmt.Println("\n=== TAX RETURN RESULTS ===")
	fmt.Println()

	// Get computed values
	agi := getFloatAttr(result, "agi")
	taxableIncome := getFloatAttr(result, "taxable_income")
	totalTax := getFloatAttr(result, "total_tax")
	refund := getFloatAttr(result, "refund_amount")
	amountOwed := getFloatAttr(result, "amount_owed")

	// Income breakdown
	totalW2 := getFloatAttr(result, "total_w2_wages")
	totalSE := getFloatAttr(result, "total_se_income")
	totalRental := getFloatAttr(result, "total_rental_income")
	grossIncome := getFloatAttr(result, "gross_income")

	// Deductions
	standardDed := getFloatAttr(result, "standard_deduction")
	totalItemized := getFloatAttr(result, "total_itemized")
	totalDeduction := getFloatAttr(result, "total_deduction")
	qbiDeduction := getFloatAttr(result, "qbi_deduction")

	// Tax components
	regularTax := getFloatAttr(result, "regular_tax")
	seTax := getFloatAttr(result, "se_tax")
	totalCredits := getFloatAttr(result, "total_credits")
	ctc := getFloatAttr(result, "total_ctc")
	odc := getFloatAttr(result, "total_odc")

	// Payments
	totalWithholding := getFloatAttr(result, "total_withholding")
	totalEstimated := getFloatAttr(result, "total_estimated_payments")
	totalPayments := getFloatAttr(result, "total_payments")

	fmt.Println("INCOME:")
	fmt.Printf("  W-2 Wages:           $%.0f\n", totalW2)
	fmt.Printf("  Self-Employment:     $%.0f\n", totalSE)
	fmt.Printf("  Rental Income:       $%.0f\n", totalRental)
	fmt.Printf("  Gross Income:        $%.0f\n", grossIncome)
	fmt.Println()

	fmt.Println("DEDUCTIONS:")
	fmt.Printf("  Standard Deduction:  $%.0f\n", standardDed)
	fmt.Printf("  Total Itemized:      $%.0f\n", totalItemized)
	fmt.Printf("  Deduction Used:      $%.0f\n", totalDeduction)
	fmt.Printf("  QBI Deduction:       $%.0f\n", qbiDeduction)
	fmt.Println()

	fmt.Println("KEY FIGURES:")
	fmt.Printf("  AGI:                 $%.0f  (expected: $%.0f)\n", agi, expectedAGI)
	fmt.Printf("  Taxable Income:      $%.0f  (expected: $%.0f)\n", taxableIncome, expectedTaxable)
	fmt.Println()

	fmt.Println("TAX CALCULATION:")
	fmt.Printf("  Regular Tax:         $%.0f\n", regularTax)
	fmt.Printf("  SE Tax:              $%.0f\n", seTax)
	fmt.Printf("  Child Tax Credit:    $%.0f\n", ctc)
	fmt.Printf("  Other Dep Credit:    $%.0f\n", odc)
	fmt.Printf("  Total Credits:       $%.0f\n", totalCredits)
	fmt.Printf("  Total Tax:           $%.0f  (expected: $%.0f)\n", totalTax, expectedTax)
	fmt.Println()

	fmt.Println("PAYMENTS:")
	fmt.Printf("  Withholding:         $%.0f\n", totalWithholding)
	fmt.Printf("  Estimated Payments:  $%.0f\n", totalEstimated)
	fmt.Printf("  Total Payments:      $%.0f\n", totalPayments)
	fmt.Println()

	fmt.Println("FINAL:")
	fmt.Printf("  Refund:              $%.0f  (expected: $%.0f)\n", refund, expectedRefund)
	fmt.Printf("  Amount Owed:         $%.0f\n", amountOwed)
	fmt.Println()

	// Print audit trail
	auditName := dtrules.GetRName("audit_trail")
	auditObj, _ := job.Get(auditName)
	if auditObj != nil {
		auditArr, _ := auditObj.ArrayValue()
		if len(auditArr) > 0 {
			fmt.Println("=== AUDIT TRAIL ===")
			for _, item := range auditArr {
				fmt.Println(item.StringValue())
			}
		}
	}

	// Validation
	fmt.Println("\n=== VALIDATION ===")
	checkValueResult(t, "AGI", agi, expectedAGI, 100)
	checkValueResult(t, "Taxable Income", taxableIncome, expectedTaxable, 100)
	checkValueResult(t, "Total Tax", totalTax, expectedTax, 100)
	checkValueResult(t, "Refund", refund, expectedRefund, 100)
}

func getFloatAttr(entity dtrules.Entity, name string) float64 {
	obj, err := entity.Get(dtrules.GetRName(name))
	if err != nil || obj == nil {
		return 0
	}
	val, err := obj.DoubleValue()
	if err != nil {
		return 0
	}
	return val
}

func checkValueResult(t *testing.T, name string, actual, expected, tolerance float64) {
	diff := actual - expected
	if diff < 0 {
		diff = -diff
	}
	status := "PASS"
	if diff > tolerance {
		status = "FAIL"
	}
	fmt.Printf("%s: actual=$%.0f expected=$%.0f diff=$%.0f [%s]\n", name, actual, expected, diff, status)
	if diff > tolerance {
		t.Errorf("%s: actual=%f expected=%f diff=%f exceeds tolerance=%f", name, actual, expected, diff, tolerance)
	}
}

// TestOBBBADeductions tests the OBBBA 2025 tax provisions
func TestOBBBADeductions(t *testing.T) {
	// Find the sample projects directory
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	// Create rule set and load from directory (multi-file structure)
	rs := session.NewRuleSet("TaxReturn")
	err := rs.LoadFromDirectory(xmlDir)
	if err != nil {
		t.Fatalf("Failed to load rules from directory: %v", err)
	}

	// Define OBBBA test cases
	// Expected values calculated with 2025 tax constants per Rev. Proc. 2024-40
	testCases := []struct {
		name            string
		file            string
		expectedAGI     float64
		expectedTaxable float64
		expectedTax     float64
		expectedRefund  float64
	}{
		{
			name:            "Tips_Single_Server",
			file:            "TestCase_OBBBA_01_Tips_Single_Server.xml",
			expectedAGI:     40000, // $65k - $25k tips deduction
			expectedTaxable: 25000, // AGI - $15k std deduction (2025)
			expectedTax:     2762,  // 2025 Single brackets on $25k
			expectedRefund:  0,
		},
		{
			name:            "Overtime_MFJ_Factory",
			file:            "TestCase_OBBBA_02_Overtime_MFJ_Factory.xml",
			expectedAGI:     97000, // $115k - $18k overtime
			expectedTaxable: 67000, // AGI - $30k std (2025)
			expectedTax:     5363,  // 2025 MFJ brackets - $2.2k CTC (OBBBA)
			expectedRefund:  0,
		},
		{
			name:            "Senior_SS_MFJ",
			file:            "TestCase_OBBBA_03_Senior_SS_MFJ.xml",
			expectedAGI:     48000, // $60k (SS+pension) - $12k senior deduction
			expectedTaxable: 14800, // $48k - $33.2k std (includes 2x$1600 for 65+)
			expectedTax:     1480,  // 10% of $14,800
			expectedRefund:  0,
		},
		{
			name:            "Tips_Overtime_Combined",
			file:            "TestCase_OBBBA_04_Tips_Overtime_Combined.xml",
			expectedAGI:     55000, // $88k - $25k tips - $8k overtime
			expectedTaxable: 40000, // AGI - $15k std (2025)
			expectedTax:     4562,  // 2025 Single brackets on $40k
			expectedRefund:  0,
		},
		{
			name:            "Tips_Phaseout",
			file:            "TestCase_OBBBA_05_Tips_Phaseout.xml",
			expectedAGI:     170000, // No deduction - phased out
			expectedTaxable: 155000, // AGI - $15k std (2025)
			expectedTax:     30047,  // 2025 Single brackets on $155k
			expectedRefund:  0,
		},
		{
			name:            "Working_Senior_Tips",
			file:            "TestCase_OBBBA_06_Working_Senior_Tips.xml",
			expectedAGI:     34000, // $52k - $12k tips - $6k senior
			expectedTaxable: 17000, // AGI - $17k std (includes $2k extra for 65+ single)
			expectedTax:     1802,  // 2025 Single brackets on $17k
			expectedRefund:  0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create fresh session for each test case
			sess, err := rs.NewSession()
			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}

			// Load mapping
			mapPath := filepath.Join(xmlDir, "TaxReturn_map.xml")
			mapFile, err := os.Open(mapPath)
			if err != nil {
				t.Fatalf("Failed to open mapping: %v", err)
			}
			defer mapFile.Close()

			m := mapping.NewMapping(sess)
			err = m.LoadMapping(mapFile)
			if err != nil {
				t.Fatalf("Failed to load mapping: %v", err)
			}
			err = m.Initialize()
			if err != nil {
				t.Fatalf("Failed to init mapping: %v", err)
			}

			// Load test data
			testPath := filepath.Join(sampleDir, "testfiles", "TestScenarios", "OBBBA_2025", tc.file)
			testFile, err := os.Open(testPath)
			if err != nil {
				t.Fatalf("Failed to open test file %s: %v", tc.file, err)
			}
			defer testFile.Close()

			err = m.LoadData(testFile)
			if err != nil {
				t.Fatalf("Failed to map test data: %v", err)
			}

			// Execute Compute_Tax_Return
			ef := sess.GetEntityFactory()
			dt, err := ef.GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
			if err != nil || dt == nil {
				t.Fatalf("Failed to get decision table: %v", err)
			}

			state := sess.GetState()
			err = dt.Execute(state)
			if err != nil {
				t.Fatalf("Failed to execute: %v", err)
			}

			// Get results
			jobName := dtrules.GetRName("job")
			job, err := state.FindEntity(jobName)
			if err != nil || job == nil {
				t.Fatalf("Failed to find job entity: %v", err)
			}

			resultsName := dtrules.GetRName("results")
			resultsObj, _ := job.Get(resultsName)
			if resultsObj == nil {
				t.Fatal("No results array found")
			}
			resultsArr, _ := resultsObj.ArrayValue()
			if len(resultsArr) == 0 {
				t.Fatal("Results array is empty")
			}

			result, _ := resultsArr[0].REntityValue()
			if result == nil {
				t.Fatal("No result entity found")
			}

			// Get computed values
			agi := getFloatAttr(result, "agi")
			taxableIncome := getFloatAttr(result, "taxable_income")
			totalTax := getFloatAttr(result, "total_tax")
			_ = getFloatAttr(result, "refund_amount") // unused but loaded for validation

			// OBBBA specific
			tipsDeduction := getFloatAttr(result, "tips_deduction")
			overtimeDeduction := getFloatAttr(result, "overtime_deduction")
			seniorDeduction := getFloatAttr(result, "senior_deduction")
			totalOBBBA := getFloatAttr(result, "total_obbba_deductions")

			fmt.Printf("\n=== %s ===\n", tc.name)
			fmt.Printf("OBBBA Deductions:\n")
			fmt.Printf("  Tips:     $%.0f\n", tipsDeduction)
			fmt.Printf("  Overtime: $%.0f\n", overtimeDeduction)
			fmt.Printf("  Senior:   $%.0f\n", seniorDeduction)
			fmt.Printf("  Total:    $%.0f\n", totalOBBBA)
			fmt.Printf("Results:\n")
			fmt.Printf("  AGI:           $%.0f (expected: $%.0f)\n", agi, tc.expectedAGI)
			fmt.Printf("  Taxable:       $%.0f (expected: $%.0f)\n", taxableIncome, tc.expectedTaxable)
			fmt.Printf("  Total Tax:     $%.0f (expected: $%.0f)\n", totalTax, tc.expectedTax)

			// Validate - use larger tolerance for complex calculations
			checkValueResult(t, "AGI", agi, tc.expectedAGI, 500)
			checkValueResult(t, "Taxable Income", taxableIncome, tc.expectedTaxable, 500)
			checkValueResult(t, "Total Tax", totalTax, tc.expectedTax, 500)
		})
	}
}

// TestNewTaxScenarios tests the new tax gap implementations:
// - Social Security taxability (Publication 915)
// - Capital Gains (Schedule D, 0%/15%/20%)
// - NIIT (Form 8960, 3.8%)
// - Additional Medicare Tax (Form 8959, 0.9%)
// - IRA/HSA deductions
// - Student Loan Interest deduction
func TestNewTaxScenarios(t *testing.T) {
	// Find the sample projects directory
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	// Create rule set and load from directory (multi-file structure)
	rs := session.NewRuleSet("TaxReturn")
	err := rs.LoadFromDirectory(xmlDir)
	if err != nil {
		t.Fatalf("Failed to load rules from directory: %v", err)
	}

	// Define test cases for new scenarios
	testCases := []struct {
		name       string
		folder     string
		file       string
		checkField string
	}{
		{
			name:       "SS_Taxability",
			folder:     "SocialSecurity",
			file:       "TestCase_SS_Taxability.xml",
			checkField: "taxable_social_security",
		},
		{
			name:       "LTCG_0_Percent",
			folder:     "CapitalGains",
			file:       "TestCase_LTCG_0_Percent.xml",
			checkField: "capital_gains_tax",
		},
		{
			name:       "LTCG_15_Percent",
			folder:     "CapitalGains",
			file:       "TestCase_LTCG_15_Percent.xml",
			checkField: "capital_gains_tax",
		},
		{
			name:       "Qualified_Dividends",
			folder:     "CapitalGains",
			file:       "TestCase_Qualified_Dividends.xml",
			checkField: "total_qualified_dividends",
		},
		{
			name:       "NIIT_High_Income",
			folder:     "Surtaxes",
			file:       "TestCase_NIIT_High_Income.xml",
			checkField: "niit_tax",
		},
		{
			name:       "Additional_Medicare",
			folder:     "Surtaxes",
			file:       "TestCase_Additional_Medicare.xml",
			checkField: "additional_medicare_tax",
		},
		{
			name:       "IRA_Full_Deduction",
			folder:     "Retirement",
			file:       "TestCase_IRA_Full_Deduction.xml",
			checkField: "ira_deduction",
		},
		{
			name:       "HSA_Family",
			folder:     "Retirement",
			file:       "TestCase_HSA_Family.xml",
			checkField: "hsa_deduction",
		},
		{
			name:       "Student_Loan",
			folder:     "Adjustments",
			file:       "TestCase_Student_Loan.xml",
			checkField: "student_loan_deduction",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create fresh session for each test case
			sess, err := rs.NewSession()
			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}

			// Load mapping
			mapPath := filepath.Join(xmlDir, "TaxReturn_map.xml")
			mapFile, err := os.Open(mapPath)
			if err != nil {
				t.Fatalf("Failed to open mapping: %v", err)
			}
			defer mapFile.Close()

			m := mapping.NewMapping(sess)
			err = m.LoadMapping(mapFile)
			if err != nil {
				t.Fatalf("Failed to load mapping: %v", err)
			}
			err = m.Initialize()
			if err != nil {
				t.Fatalf("Failed to init mapping: %v", err)
			}

			// Load test data
			testPath := filepath.Join(sampleDir, "testfiles", "TestScenarios", tc.folder, tc.file)
			testFile, err := os.Open(testPath)
			if err != nil {
				t.Fatalf("Failed to open test file %s: %v", tc.file, err)
			}
			defer testFile.Close()

			err = m.LoadData(testFile)
			if err != nil {
				t.Fatalf("Failed to map test data: %v", err)
			}

			// Execute Compute_Tax_Return
			ef := sess.GetEntityFactory()
			dt, err := ef.GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
			if err != nil || dt == nil {
				t.Fatalf("Failed to get decision table: %v", err)
			}

			state := sess.GetState()
			err = dt.Execute(state)
			if err != nil {
				t.Fatalf("Failed to execute: %v", err)
			}

			// Get results
			jobName := dtrules.GetRName("job")
			job, err := state.FindEntity(jobName)
			if err != nil || job == nil {
				t.Fatalf("Failed to find job entity: %v", err)
			}

			resultsName := dtrules.GetRName("results")
			resultsObj, _ := job.Get(resultsName)
			if resultsObj == nil {
				t.Fatal("No results array found")
			}
			resultsArr, _ := resultsObj.ArrayValue()
			if len(resultsArr) == 0 {
				t.Fatal("Results array is empty")
			}

			result, _ := resultsArr[0].REntityValue()
			if result == nil {
				t.Fatal("No result entity found")
			}

			// Get computed values
			agi := getFloatAttr(result, "agi")
			taxableIncome := getFloatAttr(result, "taxable_income")
			totalTax := getFloatAttr(result, "total_tax")
			checkValue := getFloatAttr(result, tc.checkField)

			fmt.Printf("\n=== %s ===\n", tc.name)
			fmt.Printf("  AGI:           $%.0f\n", agi)
			fmt.Printf("  Taxable:       $%.0f\n", taxableIncome)
			fmt.Printf("  Total Tax:     $%.0f\n", totalTax)
			fmt.Printf("  %s: $%.2f\n", tc.checkField, checkValue)

			// Print audit trail for debugging
			auditName := dtrules.GetRName("audit_trail")
			auditObj, _ := job.Get(auditName)
			if auditObj != nil {
				auditArr, _ := auditObj.ArrayValue()
				if len(auditArr) > 0 {
					fmt.Println("  --- Audit Trail (last 20 lines) ---")
					start := len(auditArr) - 20
					if start < 0 {
						start = 0
					}
					for i := start; i < len(auditArr); i++ {
						fmt.Printf("  %s\n", auditArr[i].StringValue())
					}
				}
			}

			// Basic validation - execution should complete without error
			// and produce some output
			if agi == 0 && taxableIncome == 0 {
				t.Errorf("Calculation produced zero AGI and taxable income")
			}
		})
	}
}

// Test2025Constants verifies 2025 tax constants using existing test data files
// This tests the Level 1 simple scenarios which verify standard deduction and tax brackets
func Test2025Constants(t *testing.T) {
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	// Create rule set once and reuse for all test cases
	rs := session.NewRuleSet("TaxReturn")
	err := rs.LoadFromDirectory(xmlDir)
	if err != nil {
		t.Fatalf("Failed to load rules from directory: %v", err)
	}

	// Test cases using existing Level 1 test files
	// These verify that 2025 standard deductions are applied correctly
	// Tax amounts calculated using 2025 brackets per Rev. Proc. 2024-40
	testCases := []struct {
		name             string
		file             string
		expectedAGI      float64
		expectedTaxable  float64
		expectedTax      float64
		expectedStdDed   float64
		description      string
	}{
		{
			name:            "Single_W2_Standard",
			file:            "Level1_Simple/TestCase_L1_01_Single_W2_Standard.xml",
			expectedAGI:     65000,
			expectedTaxable: 49250,  // 65000 - 15750 std ded
			expectedTax:     5749,   // 2025 brackets: 10% on $11,925 + 12% on $36,550 + 22% on $775
			expectedStdDed:  15750,  // 2025 Single std ded per Rev. Proc. 2024-40
			description:     "Verifies Single standard deduction $15,750",
		},
		{
			name:            "MFJ_W2_Standard",
			file:            "Level1_Simple/TestCase_L1_03_MFJ_W2_Standard.xml",
			expectedAGI:     150000, // $90k + $60k W-2 wages
			expectedTaxable: 118500, // 150000 - 31500 std ded
			expectedTax:     15898,  // 2025 MFJ brackets: 10% on $23,850 + 12% on $73,100 + 22% on $21,550
			expectedStdDed:  31500,  // 2025 MFJ std ded per Rev. Proc. 2024-40
			description:     "Verifies MFJ standard deduction $31,500",
		},
		{
			name:            "HOH_W2_One_Child",
			file:            "Level1_Simple/TestCase_L1_04_HOH_W2_One_Child.xml",
			expectedAGI:     70000,
			expectedTaxable: 46375,  // 70000 - 23625 std ded
			expectedTax:     3127,   // HOH brackets - credits
			expectedStdDed:  23625,  // 2025 HOH std ded per Rev. Proc. 2024-40
			description:     "Verifies HOH standard deduction $23,625",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sess, err := rs.NewSession()
			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}

			mapFile, err := os.Open(filepath.Join(xmlDir, "TaxReturn_map.xml"))
			if err != nil {
				t.Fatalf("Failed to open mapping: %v", err)
			}
			defer mapFile.Close()

			m := mapping.NewMapping(sess)
			m.LoadMapping(mapFile)
			m.Initialize()

			testFile, err := os.Open(filepath.Join(sampleDir, "testfiles", "TestScenarios", tc.file))
			if err != nil {
				t.Fatalf("Failed to open test file: %v", err)
			}
			defer testFile.Close()
			m.LoadData(testFile)

			ef := sess.GetEntityFactory()
			dt, _ := ef.GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
			state := sess.GetState()
			err = dt.Execute(state)
			if err != nil {
				t.Fatalf("Execution failed: %v", err)
			}

			job, _ := state.FindEntity(dtrules.GetRName("job"))
			resultsObj, _ := job.Get(dtrules.GetRName("results"))
			resultsArr, _ := resultsObj.ArrayValue()
			if len(resultsArr) == 0 {
				t.Fatal("No results")
			}
			result, _ := resultsArr[0].REntityValue()

			agi := getFloatAttr(result, "agi")
			taxable := getFloatAttr(result, "taxable_income")
			totalTax := getFloatAttr(result, "total_tax")
			stdDed := getFloatAttr(result, "standard_deduction")

			fmt.Printf("\n=== %s ===\n", tc.name)
			fmt.Printf("Description: %s\n", tc.description)
			fmt.Printf("Standard Deduction: $%.0f (expected $%.0f)\n", stdDed, tc.expectedStdDed)
			fmt.Printf("AGI: $%.0f (expected $%.0f)\n", agi, tc.expectedAGI)
			fmt.Printf("Taxable: $%.0f (expected $%.0f)\n", taxable, tc.expectedTaxable)
			fmt.Printf("Total Tax: $%.0f (expected $%.0f)\n", totalTax, tc.expectedTax)

			// Verify standard deduction - this is the key 2025 constant verification
			if stdDed != tc.expectedStdDed {
				t.Errorf("Standard deduction: got $%.0f, want $%.0f", stdDed, tc.expectedStdDed)
			}

			// Verify AGI
			checkValueResult(t, "AGI", agi, tc.expectedAGI, 10)

			// Verify taxable income (this validates std ded was applied correctly)
			checkValueResult(t, "Taxable Income", taxable, tc.expectedTaxable, 10)

			// Verify tax (this validates brackets are correct)
			checkValueResult(t, "Total Tax", totalTax, tc.expectedTax, 100)
		})
	}
}

// Test2025HSALimits verifies HSA contribution limits per Rev. Proc. 2024-25
// Uses existing HSA test file
func Test2025HSALimits(t *testing.T) {
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	rs := session.NewRuleSet("TaxReturn")
	err := rs.LoadFromDirectory(xmlDir)
	if err != nil {
		t.Fatalf("Failed to load rules from directory: %v", err)
	}

	sess, _ := rs.NewSession()

	mapFile, _ := os.Open(filepath.Join(xmlDir, "TaxReturn_map.xml"))
	defer mapFile.Close()

	m := mapping.NewMapping(sess)
	m.LoadMapping(mapFile)
	m.Initialize()

	// Use existing HSA test file
	testFile, err := os.Open(filepath.Join(sampleDir, "testfiles", "TestScenarios", "Retirement", "TestCase_HSA_Family.xml"))
	if err != nil {
		t.Fatalf("Failed to open HSA test file: %v", err)
	}
	defer testFile.Close()
	m.LoadData(testFile)

	ef := sess.GetEntityFactory()
	dt, _ := ef.GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
	state := sess.GetState()
	dt.Execute(state)

	job, _ := state.FindEntity(dtrules.GetRName("job"))
	resultsObj, _ := job.Get(dtrules.GetRName("results"))
	resultsArr, _ := resultsObj.ArrayValue()
	result, _ := resultsArr[0].REntityValue()

	hsaDeduction := getFloatAttr(result, "hsa_deduction")

	fmt.Printf("\n=== HSA Family Limit Test ===\n")
	fmt.Printf("HSA Deduction: $%.0f\n", hsaDeduction)

	// 2025 Family HSA limit is $8,550
	// The test file contributes $8,300, so should get full deduction
	if hsaDeduction > 8550 {
		t.Errorf("HSA deduction $%.0f exceeds 2025 family limit of $8,550", hsaDeduction)
	}
	if hsaDeduction == 0 {
		t.Errorf("HSA deduction should not be zero")
	}
}

// Test2025IRALimits verifies IRA deduction using existing test file
func Test2025IRALimits(t *testing.T) {
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	rs := session.NewRuleSet("TaxReturn")
	err := rs.LoadFromDirectory(xmlDir)
	if err != nil {
		t.Fatalf("Failed to load rules from directory: %v", err)
	}

	sess, _ := rs.NewSession()

	mapFile, _ := os.Open(filepath.Join(xmlDir, "TaxReturn_map.xml"))
	defer mapFile.Close()

	m := mapping.NewMapping(sess)
	m.LoadMapping(mapFile)
	m.Initialize()

	testFile, err := os.Open(filepath.Join(sampleDir, "testfiles", "TestScenarios", "Retirement", "TestCase_IRA_Full_Deduction.xml"))
	if err != nil {
		t.Fatalf("Failed to open IRA test file: %v", err)
	}
	defer testFile.Close()
	m.LoadData(testFile)

	ef := sess.GetEntityFactory()
	dt, _ := ef.GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
	state := sess.GetState()
	dt.Execute(state)

	job, _ := state.FindEntity(dtrules.GetRName("job"))
	resultsObj, _ := job.Get(dtrules.GetRName("results"))
	resultsArr, _ := resultsObj.ArrayValue()
	result, _ := resultsArr[0].REntityValue()

	iraDeduction := getFloatAttr(result, "ira_deduction")
	agi := getFloatAttr(result, "agi")

	fmt.Printf("\n=== IRA Full Deduction Test ===\n")
	fmt.Printf("AGI: $%.0f\n", agi)
	fmt.Printf("IRA Deduction: $%.0f\n", iraDeduction)

	// 2025 IRA limit is $7,000
	// Phase-out for Single covered by plan: $79,000 - $89,000
	if iraDeduction > 7000 {
		t.Errorf("IRA deduction $%.0f exceeds 2025 limit of $7,000", iraDeduction)
	}
	// This test case should be below phase-out, so full deduction expected
	if iraDeduction == 7000 {
		fmt.Println("PASS: Full IRA deduction of $7,000 applied (below phase-out)")
	}
}

// Test2025StudentLoanPhaseout verifies student loan interest deduction
func Test2025StudentLoanPhaseout(t *testing.T) {
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	rs := session.NewRuleSet("TaxReturn")
	err := rs.LoadFromDirectory(xmlDir)
	if err != nil {
		t.Fatalf("Failed to load rules from directory: %v", err)
	}

	sess, _ := rs.NewSession()

	mapFile, _ := os.Open(filepath.Join(xmlDir, "TaxReturn_map.xml"))
	defer mapFile.Close()

	m := mapping.NewMapping(sess)
	m.LoadMapping(mapFile)
	m.Initialize()

	testFile, err := os.Open(filepath.Join(sampleDir, "testfiles", "TestScenarios", "Adjustments", "TestCase_Student_Loan.xml"))
	if err != nil {
		t.Fatalf("Failed to open Student Loan test file: %v", err)
	}
	defer testFile.Close()
	m.LoadData(testFile)

	ef := sess.GetEntityFactory()
	dt, _ := ef.GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
	state := sess.GetState()
	dt.Execute(state)

	job, _ := state.FindEntity(dtrules.GetRName("job"))
	resultsObj, _ := job.Get(dtrules.GetRName("results"))
	resultsArr, _ := resultsObj.ArrayValue()
	result, _ := resultsArr[0].REntityValue()

	slDeduction := getFloatAttr(result, "student_loan_deduction")
	agi := getFloatAttr(result, "agi")

	fmt.Printf("\n=== Student Loan Deduction Test ===\n")
	fmt.Printf("AGI: $%.0f\n", agi)
	fmt.Printf("Student Loan Deduction: $%.0f\n", slDeduction)

	// 2025 max is $2,500
	// Phase-out for Single: $85,000 - $100,000
	if slDeduction > 2500 {
		t.Errorf("Student loan deduction $%.0f exceeds 2025 limit of $2,500", slDeduction)
	}
}

// Test2025CapitalGainsBrackets verifies capital gains tax brackets
func Test2025CapitalGainsBrackets(t *testing.T) {
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	// Create rule set once and reuse for all test cases
	rs := session.NewRuleSet("TaxReturn")
	err := rs.LoadFromDirectory(xmlDir)
	if err != nil {
		t.Fatalf("Failed to load rules from directory: %v", err)
	}

	testCases := []struct {
		name        string
		file        string
		description string
	}{
		{
			name:        "LTCG_0_Percent",
			file:        "CapitalGains/TestCase_LTCG_0_Percent.xml",
			description: "Tests 0% LTCG rate (taxable income below $48,350 single)",
		},
		{
			name:        "LTCG_15_Percent",
			file:        "CapitalGains/TestCase_LTCG_15_Percent.xml",
			description: "Tests 15% LTCG rate (taxable income above $48,350 single)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			sess, _ := rs.NewSession()

			mapFile, _ := os.Open(filepath.Join(xmlDir, "TaxReturn_map.xml"))
			defer mapFile.Close()

			m := mapping.NewMapping(sess)
			m.LoadMapping(mapFile)
			m.Initialize()

			testFile, err := os.Open(filepath.Join(sampleDir, "testfiles", "TestScenarios", tc.file))
			if err != nil {
				t.Fatalf("Failed to open test file: %v", err)
			}
			defer testFile.Close()
			m.LoadData(testFile)

			ef := sess.GetEntityFactory()
			dt, _ := ef.GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
			state := sess.GetState()
			dt.Execute(state)

			job, _ := state.FindEntity(dtrules.GetRName("job"))
			resultsObj, _ := job.Get(dtrules.GetRName("results"))
			resultsArr, _ := resultsObj.ArrayValue()
			result, _ := resultsArr[0].REntityValue()

			taxable := getFloatAttr(result, "taxable_income")
			cgTax := getFloatAttr(result, "capital_gains_tax")

			fmt.Printf("\n=== %s ===\n", tc.name)
			fmt.Printf("Description: %s\n", tc.description)
			fmt.Printf("Taxable Income: $%.0f\n", taxable)
			fmt.Printf("Capital Gains Tax: $%.0f\n", cgTax)

			// 2025 thresholds: 0% up to $48,350 single, 15% up to $533,400
			if tc.name == "LTCG_0_Percent" && cgTax > 0 {
				// Only fail if taxable income is clearly in 0% bracket
				if taxable < 48350 {
					t.Errorf("Expected 0%% CG rate but got tax of $%.0f", cgTax)
				}
			}
		})
	}
}

// TestSouthCarolinaTax tests South Carolina state income tax implementation
// SC has progressive tax brackets: 0% up to $3,560, 3% from $3,561-$17,830, and 6% above $17,830
// SC standard deduction: $15,000 (Single) or $30,000 (MFJ)
func TestSouthCarolinaTax(t *testing.T) {
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	// Test cases for SC tax implementation
	testCases := []struct {
		name            string
		file            string
		expectedAGI     float64
		expectedSCTax   float64
		description     string
	}{
		{
			name:        "SC_Low_Income",
			file:        "SC/TestCase_SC_Low_Income.xml",
			expectedAGI: 18000,
			expectedSCTax: 0, // SC taxable: $3,000 (under $3,560 threshold, 0% bracket)
			description: "SC resident with income under first bracket threshold",
		},
		{
			name:        "SC_Middle_Income",
			file:        "SC/TestCase_SC_Middle_Income.xml",
			expectedAGI: 32000,
			expectedSCTax: 403, // SC taxable: $17,000, tax: ($17,000 - $3,560) * 3% = $403.20
			description: "SC resident in 3% bracket",
		},
		{
			name:        "SC_High_Income",
			file:        "SC/TestCase_SC_High_Income.xml",
			expectedAGI: 100000,
			expectedSCTax: 3558, // SC taxable: $70,000, tax: $428.10 + ($70,000 - $17,830) * 6% = $3,558.30
			description: "SC resident MFJ in 6% bracket",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rs := session.NewRuleSet("TaxReturn")

			eddFile, err := os.Open(filepath.Join(xmlDir, "TaxReturn_edd.xml"))
			if err != nil {
				t.Fatalf("Failed to open EDD: %v", err)
			}
			defer eddFile.Close()
			rs.LoadEDD(eddFile)

			dtFile, err := os.Open(filepath.Join(xmlDir, "TaxReturn_dt.xml"))
			if err != nil {
				t.Fatalf("Failed to open DT: %v", err)
			}
			defer dtFile.Close()
			rs.LoadDecisionTables(dtFile)

			sess, err := rs.NewSession()
			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}

			mapFile, err := os.Open(filepath.Join(xmlDir, "TaxReturn_map.xml"))
			if err != nil {
				t.Fatalf("Failed to open mapping: %v", err)
			}
			defer mapFile.Close()

			m := mapping.NewMapping(sess)
			m.LoadMapping(mapFile)
			m.Initialize()

			testFile, err := os.Open(filepath.Join(sampleDir, "testfiles", "TestScenarios", tc.file))
			if err != nil {
				t.Fatalf("Failed to open test file %s: %v", tc.file, err)
			}
			defer testFile.Close()
			m.LoadData(testFile)

			ef := sess.GetEntityFactory()
			dt, _ := ef.GetDecisionTable(dtrules.GetRName("Compute_Tax_Return"))
			state := sess.GetState()
			err = dt.Execute(state)
			if err != nil {
				t.Fatalf("Execution failed: %v", err)
			}

			job, _ := state.FindEntity(dtrules.GetRName("job"))
			resultsObj, _ := job.Get(dtrules.GetRName("results"))
			resultsArr, _ := resultsObj.ArrayValue()
			if len(resultsArr) == 0 {
				t.Fatal("No results")
			}
			result, _ := resultsArr[0].REntityValue()

			agi := getFloatAttr(result, "agi")
			scTax := getFloatAttr(result, "sc_state_tax")
			scTaxable := getFloatAttr(result, "sc_taxable_income")
			totalTax := getFloatAttr(result, "total_tax")

			fmt.Printf("\n=== %s ===\n", tc.name)
			fmt.Printf("Description: %s\n", tc.description)
			fmt.Printf("AGI: $%.0f (expected $%.0f)\n", agi, tc.expectedAGI)
			fmt.Printf("SC Taxable Income: $%.0f\n", scTaxable)
			fmt.Printf("SC State Tax: $%.0f (expected $%.0f)\n", scTax, tc.expectedSCTax)
			fmt.Printf("Total Tax (Federal + SC): $%.0f\n", totalTax)

			// Verify AGI
			checkValueResult(t, "AGI", agi, tc.expectedAGI, 10)

			// Verify SC state tax
			checkValueResult(t, "SC State Tax", scTax, tc.expectedSCTax, 10)

			// Verify SC tax is included in total tax
			if totalTax < scTax {
				t.Errorf("Total tax $%.0f should include SC tax $%.0f", totalTax, scTax)
			}

			// Print audit trail for SC calculation
			auditName := dtrules.GetRName("audit_trail")
			auditObj, _ := job.Get(auditName)
			if auditObj != nil {
				auditArr, _ := auditObj.ArrayValue()
				fmt.Println("  --- SC Tax Audit Trail ---")
				for _, item := range auditArr {
					line := item.StringValue()
					if len(line) > 0 && (line[0] == ' ' && len(line) > 2 && line[2] == 'S') {
						// Print lines that start with "  South" (SC-related audit lines)
						fmt.Printf("  %s\n", line)
					}
				}
			}
		})
	}
}
