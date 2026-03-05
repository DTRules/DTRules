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

	// Expected values based on test data:
	// Income: W-2 $125k + SE net $128.2k + Rental net $4.2k = $257.4k
	// AGI: $257.4k - $9,057 SE deduction = $248,343
	// Taxable: AGI - $30k std deduction - $25.6k QBI = $192,703
	// Tax: $32,501 regular + $18,114 SE - $6,500 credits = $44,115
	expectedAGI := 248343.0
	expectedTaxable := 192703.0
	expectedTax := 44115.0
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
