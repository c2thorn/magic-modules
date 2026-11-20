package test

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/terraform-google-conversion/v7/pkg/cai2hcl"
	"github.com/GoogleCloudPlatform/terraform-google-conversion/v7/pkg/caiasset"

	"go.uber.org/zap"
)

// The CAI asset that the nightly tests metadata has for a step of a test is not always
// a snapshot of the resource at that step:
//   - The asset can be changed after the step is checked. It is then a snapshot of the resource
//     at a later step. The update time of such an asset is later than the CAI read time of the step.
//   - The asset can be without the changes of the step. It is then a snapshot of the resource
//     at an earlier step.
//
// In both cases, fields in the raw config of the step can be missing from the asset
// although CAI has them and the conversion supports them.
type snapshotChecker struct {
	// Returns the keys of a raw config that have to be in the export config.
	requiredKeys func(rawConfig map[string]any) []string
	// Converts the CAI assets of a resource into the attributes of the resource in the export config.
	exportAttributes func(assets []caiasset.Asset) (map[string]any, error)
}

// Returns the reason why the missing keys of a resource at a step are caused by a CAI asset
// that is a snapshot of the resource at another step of the test.
// Returns an empty string if this is not shown for all of the missing keys.
//
// A missing key is caused by such a CAI asset in two cases:
//  1. The CAI asset was changed after the step was checked, and the raw config of the later step
//     that the asset is a snapshot of does not need the key.
//  2. The key is in the export config that is converted from the CAI asset of another step
//     of the test. This shows that CAI has the field and that the conversion supports it.
func (c snapshotChecker) explainMissingKeys(stepData *StepTestData, resourceData ResourceTestData, missingKeys []string) string {
	if stepData == nil || len(stepData.Steps) < 2 || len(missingKeys) == 0 {
		return ""
	}

	stepNumbers := make([]int, 0, len(stepData.Steps))
	for stepNumber := range stepData.Steps {
		stepNumbers = append(stepNumbers, stepNumber)
	}
	sort.Ints(stepNumbers)

	address := resourceData.ResourceAddress
	unexplained := make(map[string]struct{}, len(missingKeys))
	for _, key := range missingKeys {
		unexplained[key] = struct{}{}
	}
	var reasons []string

	// Case 1
	if laterStep, updateTime := laterStepOfSnapshot(stepData, resourceData, stepNumbers); laterStep != 0 {
		if rawConfig := rawConfigOfResource(stepData.Steps[laterStep], address); rawConfig != nil {
			required := make(map[string]struct{})
			for _, key := range c.requiredKeys(rawConfig) {
				required[key] = struct{}{}
			}
			var keys []string
			for _, key := range missingKeys {
				if _, ok := required[key]; !ok {
					keys = append(keys, key)
					delete(unexplained, key)
				}
			}
			if len(keys) > 0 {
				reasons = append(reasons, fmt.Sprintf(
					"the asset was updated at %s, after the step was checked at %s, and the raw config of step %d does not have %v",
					updateTime.UTC().Format(time.RFC3339), stepData.Steps[stepData.StepNumber].CaiReadTime.UTC().Format(time.RFC3339), laterStep, keys,
				))
			}
		}
	}

	// Case 2
	for _, stepNumber := range stepNumbers {
		if len(unexplained) == 0 {
			break
		}
		if stepNumber == stepData.StepNumber {
			continue
		}
		attributes := c.exportAttributesOfStep(stepData.Steps[stepNumber], address)
		var keys []string
		for _, key := range missingKeys {
			if _, ok := unexplained[key]; !ok {
				continue
			}
			if _, ok := attributes[key]; ok {
				keys = append(keys, key)
				delete(unexplained, key)
			}
		}
		if len(keys) > 0 {
			reasons = append(reasons, fmt.Sprintf("the asset of step %d has %v", stepNumber, keys))
		}
	}

	if len(unexplained) > 0 {
		return ""
	}
	return fmt.Sprintf("the CAI asset of %s at step %d is a snapshot of the resource at another step: %s",
		address, stepData.StepNumber, strings.Join(reasons, "; "))
}

// Returns the later step that the CAI asset of the resource is a snapshot of, and the update time
// of the asset. This is the first step that was checked after the asset was changed.
// Returns 0 if the asset was not changed after the step was checked, or if no step was checked
// after the asset was changed.
func laterStepOfSnapshot(stepData *StepTestData, resourceData ResourceTestData, stepNumbers []int) (int, time.Time) {
	var updateTime time.Time
	for _, caiData := range resourceData.Cai {
		if caiData == nil {
			continue
		}
		if t := caiData.CaiAsset.UpdateTime.Time(); t.After(updateTime) {
			updateTime = t
		}
	}

	readTime := stepData.Steps[stepData.StepNumber].CaiReadTime
	if updateTime.IsZero() || readTime.IsZero() || !updateTime.After(readTime) {
		return 0, updateTime
	}

	for _, stepNumber := range stepNumbers {
		if stepNumber <= stepData.StepNumber {
			continue
		}
		if t := stepData.Steps[stepNumber].CaiReadTime; !t.IsZero() && !t.Before(updateTime) {
			return stepNumber, updateTime
		}
	}
	return 0, updateTime
}

// Returns the attributes of the resource in the export config that is converted
// from the CAI assets of the resource at the given step.
// Returns nil if the step does not have the CAI assets, or if they cannot be converted.
func (c snapshotChecker) exportAttributesOfStep(stepMetadata TgcMetadataPayload, address string) map[string]any {
	resourceMetadata := stepMetadata.ResourceMetadata[address]
	if resourceMetadata == nil || len(resourceMetadata.Cai) == 0 {
		return nil
	}

	assets := make([]caiasset.Asset, 0, len(resourceMetadata.Cai))
	for _, caiData := range resourceMetadata.Cai {
		if caiData == nil || caiData.CaiAsset.Type == "" {
			return nil
		}
		assets = append(assets, caiData.CaiAsset.Asset)
	}

	attributes, err := c.exportAttributes(assets)
	if err != nil {
		return nil
	}
	return attributes
}

// Returns the attributes of the resource in the raw config of the given step.
// Returns nil if the raw config does not have the resource, or if it cannot be parsed.
func rawConfigOfResource(stepMetadata TgcMetadataPayload, address string) map[string]any {
	resources, err := parseHCLBytesSafely([]byte(stepMetadata.RawConfig), fmt.Sprintf("step%d.tf", stepMetadata.StepNumber))
	if err != nil {
		return nil
	}
	return resources[address]
}

// Converts the CAI assets of a resource into the attributes of the resource in the export config.
func exportAttributes(assets []caiasset.Asset, resourceType string, logger *zap.Logger) (map[string]any, error) {
	// Perform the deep copy in case the assets are transformed in cai2hcl
	var assetsCopy []caiasset.Asset
	if err := DeepCopyMap(assets, &assetsCopy); err != nil {
		return nil, err
	}

	exportConfigData, err := cai2hcl.Convert(assetsCopy, &cai2hcl.Options{
		ErrorLogger: logger,
	})
	if err != nil {
		return nil, err
	}

	resources, err := parseHCLBytesSafely(exportConfigData, "export.tf")
	if err != nil {
		return nil, err
	}
	for address, attributes := range resources {
		// The resource type in the export config can be different from the resource type of the test.
		if len(resources) == 1 || strings.HasPrefix(address, resourceType+".") {
			return attributes, nil
		}
	}
	return nil, nil
}

// Parses HCL and returns an error instead of panicking on the expressions that parseHCLBytes does not support.
func parseHCLBytesSafely(src []byte, filePath string) (resources map[string]map[string]any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("parse HCL %s: %v", filePath, r)
		}
	}()
	return parseHCLBytes(src, filePath)
}
