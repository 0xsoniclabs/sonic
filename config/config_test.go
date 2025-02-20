package config

import (
	"strings"
	"testing"
)

func TestBootstrapNodes_AreValid(t *testing.T) {

	fakeIp := "1.2.3.4"
	fakeResolver := func(url string) (string, error) {
		return fakeIp, nil
	}

	for name, node := range Bootnodes {
		t.Run(name, func(t *testing.T) {
			for _, url := range node {
				t.Run(url, func(t *testing.T) {
					hostname, modified, err := resolveHostNameInEnodeURLInternal(url, fakeResolver)
					if err != nil {
						t.Fatalf("Failed to resolve hostname in enode URL: %v", err)
					}
					if !strings.Contains(url, hostname) {
						t.Fatalf("Hostname %q not found in URL", hostname)
					}
					if strings.Contains(modified, hostname) {
						t.Fatalf("failed to replace hostname in URL %q", modified)
					}
					if !strings.Contains(modified, fakeIp) {
						t.Fatalf("failed to insert IP in URL %q", modified)
					}
				})
			}
		})
	}
}
