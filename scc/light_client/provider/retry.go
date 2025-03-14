package provider

import (
	"fmt"
	"time"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
)

// Retry is a provider that retries requests a maximum number of times.
// It is used to wrap other providers to add a retry mechanism.
type Retry struct {
	// provider is the provider to retry requests on.
	provider Provider
	// maxRetries is the maximum number of retries to attempt.
	maxRetries uint
	// delay is the time to wait between retries.
	delay time.Duration
}

// NewRetry creates a new Retry provider with the given provider and maximum
func NewRetry(provider Provider, maxRetries uint, delay time.Duration) *Retry {
	return &Retry{
		provider:   provider,
		maxRetries: maxRetries,
		delay:      delay,
	}
}

// Close closes the Server.
// Closing an already closed Server has no effect
func (s *Retry) Close() {
	s.provider.Close()
}

// GetCommitteeCertificates returns up to `maxResults` consecutive committee
// certificates starting from the given period.
//
// Parameters:
// - first: The starting period for which to retrieve committee certificates.
// - maxResults: The maximum number of committee certificates to retrieve.
//
// Returns:
//   - []cert.CommitteeCertificate: A slice of committee certificates.
//   - error: Not nil if the provider failed to obtain the requested certificates.
func (s Retry) GetCommitteeCertificates(first scc.Period, maxResults uint64) ([]cert.CommitteeCertificate, error) {
	for i := uint(0); i < s.maxRetries; i++ {
		certs, err := s.provider.GetCommitteeCertificates(first, maxResults)
		if err == nil {
			return certs, nil
		}
		time.Sleep(s.delay)
	}
	return nil, fmt.Errorf("failed to get committee certificates after %d retries", s.maxRetries)
}
