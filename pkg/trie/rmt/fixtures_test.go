package rmt

import (
	"encoding/json"
	"os"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
)

type TransactionRootFixture struct {
	TestCases []struct {
		Description string `json:"description"`
		Input       struct {
			TransactionIDs []codec.Hex `json:"transactionIds"`
		} `json:"input"`
		Output struct {
			TransactionMerkleRoot codec.Hex `json:"transactionMerkleRoot"`
		} `json:"output"`
	} `json:"testCases"`
}

type ProofFixture struct {
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	TestCases []struct {
		Description string `json:"description"`
		Input       struct {
			Values      []codec.Hex `json:"values"`
			QueryHashes []codec.Hex `json:"queryHashes"`
		} `json:"input"`
		Output struct {
			MerkleRoot codec.Hex `json:"merkleRoot"`
			Proof      struct {
				Size          int         `json:"size,string"`
				SiblingHashes []codec.Hex `json:"siblingHashes"`
				Idxs          []string    `json:"idxs"`
			} `json:"proof"`
		} `json:"output"`
	} `json:"testCases"`
}

type UpdateLeavesFixture struct {
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	TestCases []struct {
		Description string `json:"description"`
		Input       struct {
			Values       []codec.Hex `json:"values"`
			UpdateValues []codec.Hex `json:"updateValues"`
			Proof        struct {
				Size          int         `json:"size,string"`
				SiblingHashes []codec.Hex `json:"siblingHashes"`
				Idxs          []string    `json:"indexes"`
			} `json:"proof"`
		} `json:"input"`
		Output struct {
			InitialMerkleRoot codec.Hex `json:"initialMerkleRoot"`
			FinalMerkleRoot   codec.Hex `json:"finalMerkleRoot"`
		} `json:"output"`
	} `json:"testCases"`
}

func loadFixture(path string, fixture interface{}) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(file, fixture)
}
