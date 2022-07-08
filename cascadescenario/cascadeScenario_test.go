package cascadescenario

import (
	"testing"
)
func TestReadConfigJSON(t *testing.T) {
	expectedModulesNum := 3
	filename := "./test/test_fail_first.json"
	modules := ReadConfigJSON(filename)
	if len(modules) != expectedModulesNum {
		t.Errorf("Output %q not equal to expected %q", len(modules), expectedModulesNum)
		return
	}
	t.Log("Number of modules is correct")
}

func TestReadConfigJSONBackoffLimit(t *testing.T) {
	expectedBackoffLimit := 0
	filename := "./test/test_fail_first.json"
	modules := ReadConfigJSON(filename)
	if *modules[0].BackoffLimit != int32(expectedBackoffLimit) {
		t.Errorf("Output %q not equal to expected %q", *modules[0].BackoffLimit, expectedBackoffLimit)
		return
	}
	t.Log("Scenario controller job backofflimit is correct")
}

func TestReadConfigJSONImageName(t *testing.T) {
	firstModuleImageName := "ghcr.io/randsw/grayscale:0.1.1"
	filename := "./test/test_fail_first.json"
	modules := ReadConfigJSON(filename)
	if modules[0].Template.Spec.Containers[0].Image != firstModuleImageName {
		t.Errorf("Output %q not equal to expected %q", *modules[0].BackoffLimit, firstModuleImageName)
		return
	}
	t.Log("First module image name is correct")
}