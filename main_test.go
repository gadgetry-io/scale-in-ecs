package main

import "testing"

func TestScaleInECS(t *testing.T) {
	res, err := scaleInECS("uat", "80")

	if res != "" {
		t.Errorf("Response was incorrect, got: %v, want empty string.", res)
	}

	if err != nil {
		t.Errorf("Err was not nil, got: %v", err.Error())
	}
}
