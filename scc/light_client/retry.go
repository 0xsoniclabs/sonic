// Copyright 2025 Sonic Operations Ltd
// This file is part of the Sonic Client
//
// Sonic is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Sonic is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Sonic. If not, see <http://www.gnu.org/licenses/>.

package light_client

import (
	"errors"
	"fmt"
	"time"

	"github.com/0xsoniclabs/carmen/go/carmen"
	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
	"github.com/ethereum/go-ethereum/common"
)

// retryProvider is used to wrap another provider and add a retry mechanism.
// It is a provider that maxRetries provider methods, that returned an error, a
// specified number of times with some delay between maxRetries up to a maximum timeout.
//
// Fields:
// - provider: The underlying provider to retry requests on.
// - maxRetries: The maximum number of attempt to ask for certificates.
// - timeout: The time max time willing to wait for a request without error.
type retryProvider struct {
	provider   provider
	maxRetries uint
	timeout    time.Duration
}

// newRetry creates a new retryProvider provider with the given provider and maximum
// number of maxRetries and total timeout.
//
// Parameters:
//   - provider: The underlying provider to wrap with retry logic.
//   - maxRetries: The maximum number of attempt to ask for certificates. If it
//     is zero value, then maxRetries is 1024.
//   - timeout: The time max time willing to wait. if it is zero value,
//     then timeout is 10 seconds.
//
// Returns:
// - *retryProvider: A new retryProvider provider instance.
func newRetry(provider provider, maxRetries uint, timeout time.Duration) *retryProvider {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	if maxRetries == 0 {
		maxRetries = 1024
	}
	return &retryProvider{
		provider:   provider,
		maxRetries: maxRetries,
		timeout:    timeout,
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
	return retry(r.maxRetries, r.timeout, func() ([]cert.CommitteeCertificate, error) {
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
	return retry(r.maxRetries, r.timeout, func() ([]cert.BlockCertificate, error) {
		return r.provider.getBlockCertificates(first, maxResults)
	})
}

// GetAccountProof returns the account proof corresponding to the
// given address at the given height.
//
// Parameters:
// - address: The address of the account.
// - height: The block height of the state.
//
// Returns:
// - AccountProof: The proof of the account at the given height.
// - error: Not nil if the provider failed to obtain the requested account proof.
func (r retryProvider) getAccountProof(address common.Address, height idx.Block) (carmen.WitnessProof, error) {
	return retry(r.maxRetries, r.timeout, func() (carmen.WitnessProof, error) {
		return r.provider.getAccountProof(address, height)
	})
}

// retry executes the given function a number of times equal to maxRetries, unless
// one it returns a nil error, with incremental delays, waiting up to a max of timeout.
//
// Parameters:
// - fn: The function to execute and retry if failed.
//
// Returns:
//   - C: The result of the function if it succeeded.
//   - error: Nil if at least one execution of fn returned without error.
//     The joined error of all failed maxRetries if all calls to fn failed.
func retry[C any](maxRetries uint, timeout time.Duration, fn func() (C, error)) (C, error) {
	var errs []error
	now := time.Now()
	delay := 200 * time.Millisecond
	maxDelay := time.Second
	for i := uint(0); i < maxRetries; i++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		errs = append(errs, err)
		delay = 2 * delay
		if delay > maxDelay {
			delay = maxDelay
		}
		time.Sleep(delay)
		if time.Since(now) >= timeout {
			errs = append(errs, fmt.Errorf("exceeded timeout of %v", timeout))
			break
		}
	}

	var c C
	return c, errors.Join(fmt.Errorf("all maxRetries failed: "), errors.Join(errs...))
}
