// Copyright 2026 Paul Snow
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package dtrules_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DTRules/DTRules/pkg/dtrules"
	"github.com/DTRules/DTRules/pkg/dtrules/mapping"
	"github.com/DTRules/DTRules/pkg/dtrules/session"
)

// TestSinusitisTherapy exercises the "Agentic Medical Services" sample project
// (Challenge Apr-2026). It runs the entry-point table Determine_Therapy, which
// orchestrates the three loosely-coupled decision services:
//
//	1. Determine_Medication_And_Dosing  (drug of choice + dose)
//	2. Determine_Creatinine_Clearance   (CCr formula)
//	3. Check_Drug_Interactions          (conflict warnings)
func TestSinusitisTherapy(t *testing.T) {
	baseDir := findSampleProjectsDir(t)
	if baseDir == "" {
		t.Skip("Sample projects directory not found")
	}

	sampleDir := filepath.Join(baseDir, "SinusitisTherapy")
	if _, err := os.Stat(sampleDir); os.IsNotExist(err) {
		t.Skip("SinusitisTherapy project not found")
	}
	xmlDir := filepath.Join(sampleDir, "xml")

	rs := session.NewRuleSet("SinusitisTherapy")
	if err := rs.LoadFromDirectory(xmlDir); err != nil {
		t.Fatalf("Failed to load rules from directory: %v", err)
	}

	testCases := []struct {
		name            string
		file            string
		wantDrug        string
		wantDose        float64
		wantFreq        float64
		wantDuration    float64
		wantCCrLow      float64 // expected CCr window (inclusive)
		wantCCrHigh     float64
		wantRenalAdjust bool
		wantWarningSub  string // substring expected in at least one warning ("" = no warnings)
	}{
		{
			name:            "ChallengeExample",
			file:            "ChallengeExample/input.xml",
			wantDrug:        "Levofloxacin",
			wantDose:        200,
			wantFreq:        24,
			wantDuration:    14,
			wantCCrLow:      47.0,
			wantCCrHigh:     49.0,
			wantRenalAdjust: true,
			wantWarningSub:  "Coumadin",
		},
		{
			name:            "AdultStandard",
			file:            "AdultStandard/input.xml",
			wantDrug:        "Amoxicillin",
			wantDose:        500,
			wantFreq:        24,
			wantDuration:    14,
			wantCCrLow:      100.0,
			wantCCrHigh:     200.0,
			wantRenalAdjust: false,
			wantWarningSub:  "",
		},
		{
			name:            "PediatricPatient",
			file:            "PediatricPatient/input.xml",
			wantDrug:        "Cefuroxime",
			wantDose:        200,
			wantFreq:        24,
			wantDuration:    14,
			wantCCrLow:      90.0,
			wantCCrHigh:     200.0,
			wantRenalAdjust: false,
			wantWarningSub:  "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sess, err := rs.NewSession()
			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}

			mapFile, err := os.Open(filepath.Join(xmlDir, "sinusitis_map.xml"))
			if err != nil {
				t.Fatalf("Failed to open mapping: %v", err)
			}
			defer mapFile.Close()

			m := mapping.NewMapping(sess)
			if err := m.LoadMapping(mapFile); err != nil {
				t.Fatalf("Failed to load mapping: %v", err)
			}
			if err := m.Initialize(); err != nil {
				t.Fatalf("Failed to init mapping: %v", err)
			}

			testFile, err := os.Open(filepath.Join(sampleDir, "testfiles", "TestScenarios", tc.file))
			if err != nil {
				t.Fatalf("Failed to open test file %s: %v", tc.file, err)
			}
			defer testFile.Close()
			if err := m.LoadData(testFile); err != nil {
				t.Fatalf("Failed to map test data: %v", err)
			}

			ef := sess.GetEntityFactory()
			dt, err := ef.GetDecisionTable(dtrules.GetRName("Determine_Therapy"))
			if err != nil || dt == nil {
				t.Fatalf("Failed to get Determine_Therapy table: %v", err)
			}

			state := sess.GetState()
			if err := dt.Execute(state); err != nil {
				t.Fatalf("Failed to execute Determine_Therapy: %v", err)
			}

			result, err := state.FindEntity(dtrules.GetRName("result"))
			if err != nil || result == nil {
				t.Fatalf("Failed to find result entity: %v", err)
			}

			drug := getStringAttr(result, "recommended_drug")
			dose := getFloatAttr(result, "dose_mg")
			freq := getFloatAttr(result, "frequency_hours")
			duration := getFloatAttr(result, "duration_days")
			ccr := getFloatAttr(result, "ccr")
			renalAdjust := getBoolAttr(result, "renal_adjustment")
			warnings := getStringArrayAttr(result, "warnings")

			fmt.Printf("\n=== %s ===\n", tc.name)
			fmt.Printf("  Recommended drug: %s\n", drug)
			fmt.Printf("  Dose:             %.0f mg every %.0f h for %.0f days\n", dose, freq, duration)
			fmt.Printf("  CCr:              %.2f mL/min (renal adjustment: %v)\n", ccr, renalAdjust)
			if len(warnings) == 0 {
				fmt.Printf("  Warnings:         none\n")
			} else {
				fmt.Printf("  Warnings:\n")
				for _, w := range warnings {
					fmt.Printf("    - %s\n", w)
				}
			}

			if drug != tc.wantDrug {
				t.Errorf("recommended_drug = %q, want %q", drug, tc.wantDrug)
			}
			if dose != tc.wantDose {
				t.Errorf("dose_mg = %.0f, want %.0f", dose, tc.wantDose)
			}
			if freq != tc.wantFreq {
				t.Errorf("frequency_hours = %.0f, want %.0f", freq, tc.wantFreq)
			}
			if duration != tc.wantDuration {
				t.Errorf("duration_days = %.0f, want %.0f", duration, tc.wantDuration)
			}
			if ccr < tc.wantCCrLow || ccr > tc.wantCCrHigh {
				t.Errorf("ccr = %.2f, want within [%.1f, %.1f]", ccr, tc.wantCCrLow, tc.wantCCrHigh)
			}
			if renalAdjust != tc.wantRenalAdjust {
				t.Errorf("renal_adjustment = %v, want %v", renalAdjust, tc.wantRenalAdjust)
			}
			if tc.wantWarningSub == "" {
				if len(warnings) != 0 {
					t.Errorf("expected no warnings, got %d", len(warnings))
				}
			} else {
				found := false
				for _, w := range warnings {
					if strings.Contains(w, tc.wantWarningSub) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected a warning containing %q, got %v", tc.wantWarningSub, warnings)
				}
			}
		})
	}
}

func getStringAttr(entity dtrules.Entity, name string) string {
	obj, err := entity.Get(dtrules.GetRName(name))
	if err != nil || obj == nil {
		return ""
	}
	return obj.StringValue()
}

func getBoolAttr(entity dtrules.Entity, name string) bool {
	obj, err := entity.Get(dtrules.GetRName(name))
	if err != nil || obj == nil {
		return false
	}
	val, err := obj.BooleanValue()
	if err != nil {
		return false
	}
	return val
}

func getStringArrayAttr(entity dtrules.Entity, name string) []string {
	obj, err := entity.Get(dtrules.GetRName(name))
	if err != nil || obj == nil {
		return nil
	}
	arr, err := obj.ArrayValue()
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, v := range arr {
		out = append(out, v.StringValue())
	}
	return out
}
