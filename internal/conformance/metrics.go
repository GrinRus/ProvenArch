package conformance

import "github.com/GrinRus/ProvenArch/internal/runtime/providercommon"

const (
	FirstPassSuccessThreshold  = 0.95
	ProviderRepairRateLimit    = 0.10
	ProviderInvocationP95Limit = 2
)

// EntryMetricSample is a provider-free observation used to decide whether the
// conditional W24G mechanical-envelope slice should start. The sample keeps
// semantic validity separate from mechanical failures: an otherwise-valid
// payload that needed a provider repair is the denominator required by the
// backlog entry condition.
type EntryMetricSample struct {
	ID                  string `json:"id"`
	FirstPassValid      bool   `json:"first_pass_valid"`
	OtherwiseValid      bool   `json:"otherwise_valid"`
	ProviderRepair      bool   `json:"provider_repair"`
	MechanicalIssue     bool   `json:"mechanical_issue"`
	ProviderInvocations int    `json:"provider_invocations"`
}

type EntryMetric struct {
	Version                    int      `json:"version"`
	MeasurementKind            string   `json:"measurement_kind"`
	SampleCount                int      `json:"sample_count"`
	FirstPassValid             int      `json:"first_pass_valid"`
	FirstPassInvalid           int      `json:"first_pass_invalid"`
	FirstPassSuccessRate       float64  `json:"first_pass_success_rate"`
	OtherwiseValidCount        int      `json:"otherwise_valid_count"`
	ProviderRepairCount        int      `json:"provider_repair_count"`
	ProviderRepairRate         float64  `json:"provider_repair_rate"`
	MechanicalContractFailures int      `json:"mechanical_contract_failures"`
	ProviderInvocationP95      int      `json:"provider_invocation_p95"`
	TriggeredConditions        []string `json:"triggered_conditions"`
	EntryConditionMet          bool     `json:"entry_condition_met"`
	Decision                   string   `json:"decision"`
}

// MeasureEntryMetric applies the exact W24G entry thresholds to a fixed set
// of observations. It is intentionally provider-free and does not alter the
// W24H process-start budget or any release matrix.
func MeasureEntryMetric(samples []EntryMetricSample) EntryMetric {
	metric := EntryMetric{
		Version:         1,
		MeasurementKind: "provider_free_conformance",
		Decision:        "defer_w24g",
	}
	invocations := make([]int, 0, len(samples))
	for _, sample := range samples {
		if sample.FirstPassValid {
			metric.FirstPassValid++
		} else {
			metric.FirstPassInvalid++
		}
		if sample.OtherwiseValid {
			metric.OtherwiseValidCount++
			if sample.ProviderRepair {
				metric.ProviderRepairCount++
			}
		}
		if sample.MechanicalIssue {
			metric.MechanicalContractFailures++
		}
		if sample.ProviderInvocations > 0 {
			invocations = append(invocations, sample.ProviderInvocations)
		}
	}
	metric.SampleCount = len(samples)
	if metric.SampleCount > 0 {
		metric.FirstPassSuccessRate = float64(metric.FirstPassValid) / float64(metric.SampleCount)
	}
	if metric.OtherwiseValidCount > 0 {
		metric.ProviderRepairRate = float64(metric.ProviderRepairCount) / float64(metric.OtherwiseValidCount)
	}
	metric.ProviderInvocationP95 = providercommon.ProviderInvocationP95(invocations)
	if metric.FirstPassSuccessRate < FirstPassSuccessThreshold {
		metric.TriggeredConditions = append(metric.TriggeredConditions, "first_pass_success_below_95_percent")
	}
	if metric.ProviderRepairRate > ProviderRepairRateLimit {
		metric.TriggeredConditions = append(metric.TriggeredConditions, "otherwise_valid_provider_repair_above_10_percent")
	}
	if metric.ProviderInvocationP95 > ProviderInvocationP95Limit && metric.MechanicalContractFailures > 0 {
		metric.TriggeredConditions = append(metric.TriggeredConditions, "p95_provider_invocations_above_two_for_mechanical_failures")
	}
	metric.EntryConditionMet = len(metric.TriggeredConditions) > 0
	if metric.EntryConditionMet {
		metric.Decision = "start_w24g_design"
	}
	return metric
}
