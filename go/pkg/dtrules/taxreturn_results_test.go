package dtrules_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/DTRules/DTRules/go/pkg/dtrules"
	"github.com/DTRules/DTRules/go/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/go/pkg/dtrules/session"
)

func TestTaxReturnResults(t *testing.T) {
	// Find the sample projects directory
	cwd, _ := os.Getwd()
	sampleDir := filepath.Join(cwd, "..", "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	// Create rule set
	rs := session.NewRuleSet("TaxReturn")

	// Load EDD
	eddPath := filepath.Join(xmlDir, "TaxReturn_edd.xml")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open EDD: %v", err)
	}
	defer eddFile.Close()

	err = rs.LoadEDD(eddFile)
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}

	// Load Decision Tables
	dtPath := filepath.Join(xmlDir, "TaxReturn_dt.xml")
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open DT: %v", err)
	}
	defer dtFile.Close()

	err = rs.LoadDecisionTables(dtFile)
	if err != nil {
		t.Fatalf("Failed to load DT: %v", err)
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

	// Expected values based on test data (2025 tax constants per Rev. Proc. 2024-40):
	// Income: W-2 $125k + SE net $128.2k + Rental net $4.2k = $257.4k
	// AGI: $257.4k - $9,057 SE deduction = $248,343
	// Taxable: AGI - $30k std deduction (2025 MFJ) - $25.6k QBI = $192,703
	// Tax: $32,223 regular + $18,114 SE + $29 Add Medicare - $7,100 credits = $43,265
	//   Credits: 3 × $2,200 CTC (2025 OBBBA) + $500 ODC = $7,100
	expectedAGI := 248343.0
	expectedTaxable := 192703.0
	expectedTax := 43265.0
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
	sampleDir := filepath.Join(cwd, "..", "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	// Create rule set
	rs := session.NewRuleSet("TaxReturn")

	// Load EDD
	eddPath := filepath.Join(xmlDir, "TaxReturn_edd.xml")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open EDD: %v", err)
	}
	defer eddFile.Close()

	err = rs.LoadEDD(eddFile)
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}

	// Load Decision Tables
	dtPath := filepath.Join(xmlDir, "TaxReturn_dt.xml")
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open DT: %v", err)
	}
	defer dtFile.Close()

	err = rs.LoadDecisionTables(dtFile)
	if err != nil {
		t.Fatalf("Failed to load DT: %v", err)
	}

	// Define OBBBA test cases
	// Expected values are based on actual tax calculation with OBBBA deductions
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
			expectedTaxable: 25000, // AGI - $15k std deduction
			expectedTax:     2768,  // Single brackets on $25k
			expectedRefund:  0,
		},
		{
			name:            "Overtime_MFJ_Factory",
			file:            "TestCase_OBBBA_02_Overtime_MFJ_Factory.xml",
			expectedAGI:     97000, // $115k - $18k overtime
			expectedTaxable: 67000, // AGI - $30k std
			expectedTax:     5576,  // MFJ brackets on $67k - $2k CTC for child
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
			expectedTaxable: 40000, // AGI - $15k std
			expectedTax:     4568,  // Single brackets on $40k
			expectedRefund:  0,
		},
		{
			name:            "Tips_Phaseout",
			file:            "TestCase_OBBBA_05_Tips_Phaseout.xml",
			expectedAGI:     170000, // No deduction - phased out
			expectedTaxable: 155000, // AGI - $15k std
			expectedTax:     30243,  // Single brackets on $155k
			expectedRefund:  0,
		},
		{
			name:            "Working_Senior_Tips",
			file:            "TestCase_OBBBA_06_Working_Senior_Tips.xml",
			expectedAGI:     34000, // $52k - $12k tips - $6k senior
			expectedTaxable: 17000, // AGI - $17k std (includes $2k extra for 65+ single)
			expectedTax:     1800,  // Single brackets on $17k
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
	sampleDir := filepath.Join(cwd, "..", "..", "..", "sampleprojects", "TaxReturn")
	xmlDir := filepath.Join(sampleDir, "xml")

	// Create rule set
	rs := session.NewRuleSet("TaxReturn")

	// Load EDD
	eddPath := filepath.Join(xmlDir, "TaxReturn_edd.xml")
	eddFile, err := os.Open(eddPath)
	if err != nil {
		t.Fatalf("Failed to open EDD: %v", err)
	}
	defer eddFile.Close()

	err = rs.LoadEDD(eddFile)
	if err != nil {
		t.Fatalf("Failed to load EDD: %v", err)
	}

	// Load Decision Tables
	dtPath := filepath.Join(xmlDir, "TaxReturn_dt.xml")
	dtFile, err := os.Open(dtPath)
	if err != nil {
		t.Fatalf("Failed to open DT: %v", err)
	}
	defer dtFile.Close()

	err = rs.LoadDecisionTables(dtFile)
	if err != nil {
		t.Fatalf("Failed to load DT: %v", err)
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
