package prompt

import (
	"testing"
)

func TestUserPrompt_IsNotNil(t *testing.T) {
	if UserPrompt == nil {
		t.Fatal("expected UserPrompt to be non-nil")
	}
}

func TestUserPrompter_InterfaceCompliance(t *testing.T) {
	// Verify that UserPrompt implements UserPrompter interface
	var _ UserPrompter = UserPrompt
}
