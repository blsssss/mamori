package avprobe

import "testing"

// scans the AMSI sample in memory, nothing is written to disk; the result depends on the machine,
// so it is only logged
func TestAMSIScanLive(t *testing.T) {
	if testing.Short() {
		t.Skip("touches the live system")
	}
	result, err := amsiScan()
	if err != nil {
		t.Skipf("amsi unavailable: %v", err)
	}
	t.Logf("amsi result %d, detected %v", result, classifyAMSI(result) == detected)
}
