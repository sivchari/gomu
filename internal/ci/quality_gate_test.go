package ci

import (
	"math"
	"testing"

	"github.com/sivchari/gomu/internal/report"
)

func TestQualityGateEvaluator_Evaluate(t *testing.T) {
	testCases := []struct {
		name           string
		enabled        bool
		minScore       float64
		summary        *report.Summary
		expectedPass   bool
		expectedScore  float64
		expectedReason string
	}{
		{
			name:     "disabled quality gate",
			enabled:  false,
			minScore: 80.0,
			summary: &report.Summary{
				TotalMutants:  100,
				KilledMutants: 60,
				Statistics: report.Statistics{
					Score: 60.0,
				},
			},
			expectedPass:   true,
			expectedScore:  60.0,
			expectedReason: "Quality gate disabled",
		},
		{
			name:     "pass quality gate",
			enabled:  true,
			minScore: 80.0,
			summary: &report.Summary{
				TotalMutants:  100,
				KilledMutants: 85,
				Statistics: report.Statistics{
					Score: 85.0,
				},
			},
			expectedPass:   true,
			expectedScore:  85.0,
			expectedReason: "Mutation score meets minimum threshold",
		},
		{
			name:     "fail quality gate",
			enabled:  true,
			minScore: 80.0,
			summary: &report.Summary{
				TotalMutants:  100,
				KilledMutants: 70,
				Statistics: report.Statistics{
					Score: 70.0,
				},
			},
			expectedPass:   false,
			expectedScore:  70.0,
			expectedReason: "Mutation score 70.0% is below minimum threshold of 80.0%",
		},
		{
			name:     "zero mutants",
			enabled:  true,
			minScore: 80.0,
			summary: &report.Summary{
				TotalMutants:  0,
				KilledMutants: 0,
				Statistics: report.Statistics{
					Score: 0.0,
				},
			},
			expectedPass:   false,
			expectedScore:  0.0,
			expectedReason: "No mutants generated",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			evaluator := NewQualityGateEvaluator(tc.enabled, tc.minScore)
			result := evaluator.Evaluate(tc.summary)

			if result.Pass != tc.expectedPass {
				t.Errorf("Expected pass=%v, got %v", tc.expectedPass, result.Pass)
			}

			if math.Abs(result.MutationScore-tc.expectedScore) > 0.01 {
				t.Errorf("Expected score=%f, got %f", tc.expectedScore, result.MutationScore)
			}

			if result.Reason != tc.expectedReason {
				t.Errorf("Expected reason='%s', got '%s'", tc.expectedReason, result.Reason)
			}
		})
	}
}

func TestQualityGateEvaluator_Evaluate_EdgeCases(t *testing.T) {
	evaluator := NewQualityGateEvaluator(true, 80.0)

	// Test with nil summary
	result := evaluator.Evaluate(nil)
	if result.Pass {
		t.Error("Expected false for nil summary")
	}

	if result.Reason != "No mutants generated" {
		t.Errorf("Expected 'No mutants generated', got '%s'", result.Reason)
	}

	// Test exact threshold
	summary := &report.Summary{
		TotalMutants:  100,
		KilledMutants: 80,
		Statistics: report.Statistics{
			Score: 80.0,
		},
	}

	result = evaluator.Evaluate(summary)
	if !result.Pass {
		t.Error("Expected true for exact threshold match")
	}
}
