package light_client

import (
	"errors"
	"fmt"
	"time"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// retryProvider is used to wrap other providers to add a retry mechanism.
// It is a provider that retries provider methods, which return an error, a
// specified number of times with some delay between retries.
//
// Fields:
// - provider: The underlying provider to retry requests on.
// - retries: The maximum number of attempt to ask for certificates.
// - delay: The time to wait between retries.
type retryProvider struct {
	provider provider
	retries  uint
	delay    time.Duration
}

// newRetry creates a new retryProvider provider with the given provider and maximum
// number of retries and delay between retries.
//
// Parameters:
// - provider: The underlying provider to wrap with retry logic.
// - retries: The maximum number of attempt to ask for certificates.
// - delay: The time to wait between retries.
//
// Returns:
// - *retryProvider: A new retryProvider provider instance.
func newRetry(provider provider, retries uint, delay time.Duration) *retryProvider {
	return &retryProvider{
		provider: provider,
		retries:  retries,
		delay:    delay,
	}
}

// Close closes the retryProvider.
// Closing an already closed provider has no effect.
func (r *retryProvider) close() {
	r.provider.close()
}

// getCommitteeCertificates returns up to `maxResults` consecutive committee
// certificates starting from the given period.
//
// Parameters:
// - first: The starting period for which to retrieve committee certificates.
// - maxResults: The maximum number of committee certificates to retrieve.
//
// Returns:
// - []cert.CommitteeCertificate: A slice of committee certificates.
// - error: An error if the provider failed to obtain the requested certificates.
func (r retryProvider) getCommitteeCertificates(first scc.Period, maxResults uint64) ([]cert.CommitteeCertificate, error) {
	return retry(r.retries, r.delay, func() ([]cert.CommitteeCertificate, error) {
		return r.provider.getCommitteeCertificates(first, maxResults)
	})
}

// getBlockCertificates returns up to `maxResults` consecutive block
// certificates starting from the given block number.
//
// Parameters:
// - first: The starting block number for which to retrieve the block certificate.
// - maxResults: The maximum number of block certificates to retrieve.
//
// Returns:
//   - []cert.BlockCertificate: The block certificates for the given block number
//     and the following blocks.
//   - error: An error if the provider failed to obtain the requested certificates.
func (r retryProvider) getBlockCertificates(first idx.Block, maxResults uint64) ([]cert.BlockCertificate, error) {
	return retry(r.retries, r.delay, func() ([]cert.BlockCertificate, error) {
		return r.provider.getBlockCertificates(first, maxResults)
	})
}

// retry executes the given function a number of times equal to retries+1, unless
// one it returns a nil error, waiting the specified delay between retries.
//
// Parameters:
// - fn: The function to execute and retry if failed.
//
// Returns:
//   - C: The result of the function if it succeeded.
//   - error: Nil if at least one execution of fn returned without error.
//     The joined error of all failed retries if all calls to fn failed.
func retry[C any](retries uint, delay time.Duration, fn func() (C, error)) (C, error) {
	var errs []error
	for i := uint(0); i < retries+1; i++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		errs = append(errs, err)
		time.Sleep(delay)
	}

	var c C
	return c, errors.Join(fmt.Errorf("all retries failed: "), errors.Join(errs...))
}
