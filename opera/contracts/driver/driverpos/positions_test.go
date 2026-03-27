package driverpos

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestTopics_NotZero(t *testing.T) {
	zero := common.Hash{}
	if Topics.UpdateValidatorWeight == zero {
		t.Fatal("UpdateValidatorWeight topic should not be zero")
	}
	if Topics.UpdateValidatorPubkey == zero {
		t.Fatal("UpdateValidatorPubkey topic should not be zero")
	}
	if Topics.UpdateNetworkRules == zero {
		t.Fatal("UpdateNetworkRules topic should not be zero")
	}
	if Topics.UpdateNetworkVersion == zero {
		t.Fatal("UpdateNetworkVersion topic should not be zero")
	}
	if Topics.AdvanceEpochs == zero {
		t.Fatal("AdvanceEpochs topic should not be zero")
	}
}

func TestTopics_CorrectHashes(t *testing.T) {
	expected := crypto.Keccak256Hash([]byte("UpdateValidatorWeight(uint256,uint256)"))
	if Topics.UpdateValidatorWeight != expected {
		t.Fatal("UpdateValidatorWeight hash mismatch")
	}

	expected = crypto.Keccak256Hash([]byte("AdvanceEpochs(uint256)"))
	if Topics.AdvanceEpochs != expected {
		t.Fatal("AdvanceEpochs hash mismatch")
	}
}

func TestTopics_AllUnique(t *testing.T) {
	topics := []common.Hash{
		Topics.UpdateValidatorWeight,
		Topics.UpdateValidatorPubkey,
		Topics.UpdateNetworkRules,
		Topics.UpdateNetworkVersion,
		Topics.AdvanceEpochs,
	}
	seen := make(map[common.Hash]bool)
	for _, topic := range topics {
		if seen[topic] {
			t.Fatalf("duplicate topic: %s", topic.Hex())
		}
		seen[topic] = true
	}
}
