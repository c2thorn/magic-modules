package test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/terraform-google-conversion/v7/pkg/caiasset"
)

const snapshotTestAddress = "google_test_resource.default"

var snapshotTestTime = time.Date(2026, 11, 19, 8, 0, 0, 0, time.UTC)

// Returns the metadata of a step. The step is checked readOffset after snapshotTestTime.
// The CAI asset of the step is changed updateOffset after snapshotTestTime and has the given data.
// The asset is missing if data is nil.
func snapshotTestStep(stepNumber int, readOffset time.Duration, rawConfig string, updateOffset time.Duration, data map[string]any) TgcMetadataPayload {
	metadata := &ResourceMetadata{
		ResourceType:    "google_test_resource",
		ResourceAddress: snapshotTestAddress,
	}
	if data != nil {
		updateTime := snapshotTestTime.Add(updateOffset)
		metadata.Cai = map[string]*CaiData{
			"//test.googleapis.com/resources/default": {
				CaiAsset: NightlyAsset{
					Asset: caiasset.Asset{
						Type:     "test.googleapis.com/Resource",
						Resource: &caiasset.AssetResource{Data: data},
					},
					UpdateTime: &AssetUpdateTime{Seconds: updateTime.Unix()},
				},
			},
		}
	}
	return TgcMetadataPayload{
		StepNumber:       stepNumber,
		RawConfig:        rawConfig,
		ResourceMetadata: map[string]*ResourceMetadata{snapshotTestAddress: metadata},
		PrimaryResource:  snapshotTestAddress,
		CaiReadTime:      snapshotTestTime.Add(readOffset),
	}
}

func TestExplainMissingKeys(t *testing.T) {
	t.Parallel()

	const (
		enabledConfig = `
resource "google_test_resource" "default" {
  name    = "test"
  enabled = true
}
`
		disabledConfig = `
resource "google_test_resource" "default" {
  name    = "test"
  enabled = false
}
`
		unsetConfig = `
resource "google_test_resource" "default" {
  name = "test"
}
`
		enabledWithRegionConfig = `
resource "google_test_resource" "default" {
  name    = "test"
  enabled = true
  region  = "us"
}
`
		disabledWithRegionConfig = `
resource "google_test_resource" "default" {
  name    = "test"
  enabled = false
  region  = "us"
}
`
		otherResourceConfig = `
resource "google_test_other_resource" "default" {
  name = "test"
}
`
	)
	withoutEnabled := map[string]any{"name": "test"}
	withEnabled := map[string]any{"name": "test", "enabled": true}
	withEnabledAndRegion := map[string]any{"name": "test", "enabled": true, "region": "us"}

	// The export config of the test has the same keys as the data of the CAI asset.
	checker := snapshotChecker{
		requiredKeys: func(rawConfig map[string]any) []string {
			return compareHCLFields(rawConfig, map[string]any{}, map[string]any{}, nil)
		},
		exportAttributes: func(assets []caiasset.Asset) (map[string]any, error) {
			return assets[0].Resource.Data, nil
		},
	}

	cases := []struct {
		name        string
		steps       []TgcMetadataPayload
		stepNumber  int
		missingKeys []string
		// A part of the expected reason. The missing keys are not explained if it is empty.
		wantReason string
	}{
		{
			name: "asset is changed after the step is checked and the next step unsets the field",
			steps: []TgcMetadataPayload{
				snapshotTestStep(1, 0, enabledConfig, 4*time.Second, withoutEnabled),
				snapshotTestStep(3, 15*time.Second, disabledConfig, 19*time.Second, withoutEnabled),
			},
			stepNumber:  1,
			missingKeys: []string{"enabled"},
			wantReason:  "the raw config of step 3 does not have [enabled]",
		},
		{
			name: "asset is changed after the step is checked and the next step removes the field",
			steps: []TgcMetadataPayload{
				snapshotTestStep(1, 0, enabledConfig, 4*time.Second, withoutEnabled),
				snapshotTestStep(3, 15*time.Second, unsetConfig, 19*time.Second, withoutEnabled),
			},
			stepNumber:  1,
			missingKeys: []string{"enabled"},
			wantReason:  "the raw config of step 3 does not have [enabled]",
		},
		{
			name: "asset is changed after the step is checked but the next step also sets the field",
			steps: []TgcMetadataPayload{
				snapshotTestStep(1, 0, enabledConfig, 4*time.Second, withoutEnabled),
				snapshotTestStep(3, 15*time.Second, enabledWithRegionConfig, 19*time.Second, withoutEnabled),
			},
			stepNumber:  1,
			missingKeys: []string{"enabled"},
		},
		{
			name: "asset is changed after the last step is checked",
			steps: []TgcMetadataPayload{
				snapshotTestStep(1, 0, disabledConfig, 4*time.Second, withoutEnabled),
				snapshotTestStep(3, 15*time.Second, enabledConfig, 19*time.Second, withoutEnabled),
			},
			stepNumber:  3,
			missingKeys: []string{"enabled"},
		},
		{
			name: "asset is changed after two steps are checked",
			steps: []TgcMetadataPayload{
				snapshotTestStep(1, 0, enabledConfig, 20*time.Second, withoutEnabled),
				snapshotTestStep(3, 15*time.Second, enabledWithRegionConfig, 20*time.Second, withoutEnabled),
				snapshotTestStep(5, 30*time.Second, disabledConfig, 34*time.Second, withoutEnabled),
			},
			stepNumber:  1,
			missingKeys: []string{"enabled"},
			wantReason:  "the raw config of step 5 does not have [enabled]",
		},
		{
			name: "asset is changed after the step is checked and the resource is not in the next step",
			steps: []TgcMetadataPayload{
				snapshotTestStep(1, 0, enabledConfig, 4*time.Second, withoutEnabled),
				snapshotTestStep(3, 15*time.Second, otherResourceConfig, 19*time.Second, nil),
			},
			stepNumber:  1,
			missingKeys: []string{"enabled"},
		},
		{
			name: "asset does not have the changes of the step and the asset of a later step has the field",
			steps: []TgcMetadataPayload{
				snapshotTestStep(1, 0, unsetConfig, -5*time.Second, withoutEnabled),
				snapshotTestStep(3, 200*time.Second, enabledConfig, 120*time.Second, withoutEnabled),
				snapshotTestStep(5, 500*time.Second, enabledWithRegionConfig, 495*time.Second, withEnabled),
			},
			stepNumber:  3,
			missingKeys: []string{"enabled"},
			wantReason:  "the asset of step 5 has [enabled]",
		},
		{
			name: "asset does not have the changes of the step and the asset of an earlier step has the field",
			steps: []TgcMetadataPayload{
				snapshotTestStep(1, 0, enabledConfig, -5*time.Second, withEnabled),
				snapshotTestStep(3, 200*time.Second, unsetConfig, 195*time.Second, withoutEnabled),
				snapshotTestStep(5, 500*time.Second, enabledConfig, 195*time.Second, withoutEnabled),
			},
			stepNumber:  5,
			missingKeys: []string{"enabled"},
			wantReason:  "the asset of step 1 has [enabled]",
		},
		{
			name: "field is not in the asset of any step",
			steps: []TgcMetadataPayload{
				snapshotTestStep(1, 0, unsetConfig, -5*time.Second, withoutEnabled),
				snapshotTestStep(3, 200*time.Second, enabledConfig, 195*time.Second, withoutEnabled),
				snapshotTestStep(5, 500*time.Second, enabledWithRegionConfig, 495*time.Second, withoutEnabled),
			},
			stepNumber:  3,
			missingKeys: []string{"enabled"},
		},
		{
			name: "other steps do not have the asset",
			steps: []TgcMetadataPayload{
				snapshotTestStep(1, 0, unsetConfig, 0, nil),
				snapshotTestStep(3, 200*time.Second, enabledConfig, 195*time.Second, withoutEnabled),
				snapshotTestStep(5, 500*time.Second, enabledConfig, 0, nil),
			},
			stepNumber:  3,
			missingKeys: []string{"enabled"},
		},
		{
			name: "all of the missing fields are not in the next step",
			steps: []TgcMetadataPayload{
				snapshotTestStep(1, 0, enabledWithRegionConfig, 4*time.Second, withoutEnabled),
				snapshotTestStep(3, 15*time.Second, disabledConfig, 19*time.Second, withoutEnabled),
			},
			stepNumber:  1,
			missingKeys: []string{"enabled", "region"},
			wantReason:  "the raw config of step 3 does not have [enabled region]",
		},
		{
			name: "one missing field is not in the next step and the other is in the asset of another step",
			steps: []TgcMetadataPayload{
				snapshotTestStep(1, 0, enabledWithRegionConfig, 4*time.Second, withoutEnabled),
				snapshotTestStep(3, 15*time.Second, disabledWithRegionConfig, 19*time.Second, withoutEnabled),
				snapshotTestStep(5, 30*time.Second, enabledWithRegionConfig, 25*time.Second, withEnabledAndRegion),
			},
			stepNumber:  1,
			missingKeys: []string{"enabled", "region"},
			wantReason:  "the raw config of step 3 does not have [enabled]; the asset of step 5 has [region]",
		},
		{
			name: "one of the missing fields is not explained",
			steps: []TgcMetadataPayload{
				snapshotTestStep(1, 0, unsetConfig, -5*time.Second, withoutEnabled),
				snapshotTestStep(3, 200*time.Second, enabledWithRegionConfig, 120*time.Second, withoutEnabled),
				snapshotTestStep(5, 500*time.Second, enabledWithRegionConfig, 495*time.Second, withEnabled),
			},
			stepNumber:  3,
			missingKeys: []string{"enabled", "region"},
		},
		{
			name: "test has one step",
			steps: []TgcMetadataPayload{
				snapshotTestStep(1, 0, enabledConfig, 4*time.Second, withoutEnabled),
			},
			stepNumber:  1,
			missingKeys: []string{"enabled"},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			stepData := &StepTestData{
				StepNumber: tc.stepNumber,
				Steps:      make(map[int]TgcMetadataPayload),
			}
			for _, step := range tc.steps {
				stepData.Steps[step.StepNumber] = step
			}
			resourceData := ResourceTestData{
				ResourceMetadata: *stepData.Steps[tc.stepNumber].ResourceMetadata[snapshotTestAddress],
			}

			got := checker.explainMissingKeys(stepData, resourceData, tc.missingKeys)
			if tc.wantReason == "" {
				if got != "" {
					t.Errorf("the missing keys %v are explained, want not explained: %s", tc.missingKeys, got)
				}
				return
			}
			if !strings.Contains(got, tc.wantReason) {
				t.Errorf("got reason %q, want it to contain %q", got, tc.wantReason)
			}
		})
	}
}

func TestExplainMissingKeys_noUpdateTime(t *testing.T) {
	t.Parallel()

	const config = `
resource "google_test_resource" "default" {
  enabled = true
}
`
	steps := []TgcMetadataPayload{
		snapshotTestStep(1, 0, config, 4*time.Second, map[string]any{}),
		snapshotTestStep(3, 15*time.Second, `resource "google_test_resource" "default" {}`, 19*time.Second, map[string]any{}),
	}
	// Metadata without the update time of the asset or without the CAI read time of the step
	for _, caiData := range steps[0].ResourceMetadata[snapshotTestAddress].Cai {
		caiData.CaiAsset.UpdateTime = nil
	}
	steps[1].CaiReadTime = time.Time{}

	checker := snapshotChecker{
		requiredKeys: func(rawConfig map[string]any) []string {
			return compareHCLFields(rawConfig, map[string]any{}, map[string]any{}, nil)
		},
		exportAttributes: func(assets []caiasset.Asset) (map[string]any, error) {
			return assets[0].Resource.Data, nil
		},
	}
	stepData := &StepTestData{
		StepNumber: 1,
		Steps:      map[int]TgcMetadataPayload{1: steps[0], 3: steps[1]},
	}
	resourceData := ResourceTestData{
		ResourceMetadata: *steps[0].ResourceMetadata[snapshotTestAddress],
	}
	if got := checker.explainMissingKeys(stepData, resourceData, []string{"enabled"}); got != "" {
		t.Errorf("the missing keys are explained, want not explained: %s", got)
	}
	if got := checker.explainMissingKeys(nil, resourceData, []string{"enabled"}); got != "" {
		t.Errorf("the missing keys are explained without the data of the steps, want not explained: %s", got)
	}
}

func TestNightlyAssetJSON(t *testing.T) {
	t.Parallel()

	// The format of the CAI asset in the nightly tests metadata
	src := `{
  "cai_asset": {
    "ancestors": ["projects/123", "organizations/456"],
    "asset_type": "networksecurity.googleapis.com/FirewallEndpoint",
    "name": "//networksecurity.googleapis.com/projects/p/locations/us-central1-a/firewallEndpoints/e",
    "resource": {
      "data": {"state": "ACTIVE"},
      "version": "v1"
    },
    "update_time": {"nanos": 961110000, "seconds": 1795076494}
  }
}`
	var caiData CaiData
	if err := json.Unmarshal([]byte(src), &caiData); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := caiData.CaiAsset.Type, "networksecurity.googleapis.com/FirewallEndpoint"; got != want {
		t.Errorf("got asset type %q, want %q", got, want)
	}
	if caiData.CaiAsset.Resource == nil || caiData.CaiAsset.Resource.Data["state"] != "ACTIVE" {
		t.Errorf("got resource %+v, want the data of the asset", caiData.CaiAsset.Resource)
	}
	want := time.Unix(1795076494, 961110000)
	if got := caiData.CaiAsset.UpdateTime.Time(); !got.Equal(want) {
		t.Errorf("got update time %s, want %s", got, want)
	}

	// The update time is kept when the metadata is written to a local file and read again.
	b, err := json.Marshal(caiData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var caiDataCopy CaiData
	if err := json.Unmarshal(b, &caiDataCopy); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := caiDataCopy.CaiAsset.UpdateTime.Time(); !got.Equal(want) {
		t.Errorf("got update time %s after writing and reading, want %s", got, want)
	}
	if got, want := caiDataCopy.CaiAsset.Name, caiData.CaiAsset.Name; got != want {
		t.Errorf("got asset name %q after writing and reading, want %q", got, want)
	}

	// An asset that could not be fetched
	var missing CaiData
	if err := json.Unmarshal([]byte(`{"cai_asset": null}`), &missing); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if missing.CaiAsset.Type != "" || !missing.CaiAsset.UpdateTime.Time().IsZero() {
		t.Errorf("got %+v, want an empty asset", missing.CaiAsset)
	}
}
