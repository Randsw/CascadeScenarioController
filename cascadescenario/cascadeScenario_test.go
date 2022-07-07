package cascadescenario

import (
	//"encoding/json"
	//"fmt"
	"testing"
)


func TestReadConfigJSON(t *testing.T) {
	expectedModulesNum := 3
	filename := "./test/test_fail_first.json"
	modules := ReadConfigJSON(filename)
	if len(modules) != expectedModulesNum {
		t.Errorf("Output %q not equal to expected %q", len(modules), expectedModulesNum)
	}
	t.Log("Number of modules is correct")
}