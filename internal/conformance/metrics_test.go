package conformance

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type entryMetricFixture struct {
	Version         int                 `json:"version"`
	MeasurementKind string              `json:"measurement_kind"`
	Corpus          string              `json:"corpus"`
	Samples         []EntryMetricSample `json:"samples"`
	Expected        EntryMetric         `json:"expected"`
}

func TestW24GEntryMetricFixtureRecordsDeferredDecision(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "conformance", "w24g-entry-metric.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture entryMetricFixture
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Version != 1 || fixture.MeasurementKind != "provider_free_conformance" || fixture.Corpus != "w24i-incident-corpus" {
		t.Fatalf("unexpected fixture identity: %+v", fixture)
	}
	got := MeasureEntryMetric(fixture.Samples)
	if got.SampleCount != fixture.Expected.SampleCount || got.FirstPassValid != fixture.Expected.FirstPassValid || got.FirstPassInvalid != fixture.Expected.FirstPassInvalid || got.ProviderInvocationP95 != fixture.Expected.ProviderInvocationP95 || got.EntryConditionMet != fixture.Expected.EntryConditionMet || got.Decision != fixture.Expected.Decision {
		t.Fatalf("entry metric mismatch: got=%+v expected=%+v", got, fixture.Expected)
	}
	if got.EntryConditionMet {
		t.Fatalf("W24G must remain deferred for the recorded provider-free corpus: %+v", got)
	}
}

func TestW24GEntryMetricTriggersEachThreshold(t *testing.T) {
	tests := []struct {
		name    string
		samples []EntryMetricSample
		want    string
	}{
		{
			name:    "first pass",
			samples: []EntryMetricSample{{FirstPassValid: false, ProviderInvocations: 1}},
			want:    "first_pass_success_below_95_percent",
		},
		{
			name:    "repair rate",
			samples: []EntryMetricSample{{FirstPassValid: true, OtherwiseValid: true, ProviderRepair: true, ProviderInvocations: 1}, {FirstPassValid: true, OtherwiseValid: true, ProviderInvocations: 1}},
			want:    "otherwise_valid_provider_repair_above_10_percent",
		},
		{
			name:    "mechanical p95",
			samples: []EntryMetricSample{{FirstPassValid: true, OtherwiseValid: true, MechanicalIssue: true, ProviderInvocations: 3}},
			want:    "p95_provider_invocations_above_two_for_mechanical_failures",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metric := MeasureEntryMetric(test.samples)
			if !metric.EntryConditionMet || len(metric.TriggeredConditions) != 1 || metric.TriggeredConditions[0] != test.want {
				t.Fatalf("unexpected threshold result: %+v", metric)
			}
			if metric.Decision != "start_w24g_design" {
				t.Fatalf("triggered metric must select W24G: %+v", metric)
			}
		})
	}
}
