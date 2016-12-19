package repo

import "testing"

func TestGetBetSummary(t *testing.T) {
	_, err := GetBetSummary(1)
	if err == nil {
		t.Fatal("error is expected")
	}
	//TODO first implement adding functionality, add bet here, then check with this function
}
