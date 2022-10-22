package crypto

import (
	"encoding/hex"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type signFixture struct {
	Input struct {
		PrivKey string `yaml:"privkey"`
		Message string `yaml:"message"`
	} `yaml:"input"`
	Output string `yaml:"output"`
}

type verifyFixture struct {
	Input struct {
		PubKey    string `yaml:"pubkey"`
		Message   string `yaml:"message"`
		Signature string `yaml:"signature"`
	} `yaml:"input"`
	Output bool `yaml:"output"`
}

type popVerifyFixture struct {
	Input struct {
		PrivKey string `yaml:"pk"`
		Proof   string `yaml:"proof"`
	} `yaml:"input"`
	Output bool `yaml:"output"`
}

type hexInOutFixture struct {
	Input  string `yaml:"input"`
	Output string `yaml:"output"`
}

type createAggSigFixture struct {
	Input struct {
		KeyList     []string `yaml:"key_list"`
		KeySigPairs []struct {
			PublicKey string `yaml:"pk"`
			Signature string `yaml:"signature"`
		} `yaml:"key_sig_pairs"`
	} `yaml:"input"`
	Output struct {
		AggrigationBits string `yaml:"aggregation_bits"`
		Signature       string `yaml:"signature"`
	} `yaml:"output"`
}

type verifyAggSigFixture struct {
	Input struct {
		KeyList         []string `yaml:"key_list"`
		AggrigationBits string   `yaml:"aggregation_bits"`
		Signature       string   `yaml:"signature"`
		Tag             string   `yaml:"tag"`
		NetID           string   `yaml:"netId"`
		Message         string   `yaml:"message"`
	} `yaml:"input"`
	Output bool `yaml:"output"`
}

type verifyWeightedAggSigFixture struct {
	Input struct {
		KeyList         []string `yaml:"key_list"`
		AggrigationBits string   `yaml:"aggregation_bits"`
		Signature       string   `yaml:"signature"`
		Tag             string   `yaml:"tag"`
		NetID           string   `yaml:"netId"`
		Message         string   `yaml:"message"`
		Weights         []uint64 `yaml:"weights"`
		Threshold       uint64   `yaml:"threshold"`
	} `yaml:"input"`
	Output bool `yaml:"output"`
}

func strToHex(str string) []byte {
	if len(str) == 0 {
		return []byte{}
	}
	res, err := hex.DecodeString(str[2:])
	if err != nil {
		panic(err)
	}
	return res
}

func loadYaml(path string, fixture interface{}) {
	file, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	if err := yaml.Unmarshal(file, fixture); err != nil {
		panic(err)
	}
}

func readFolder(path string) []string {
	files, err := os.ReadDir(path)
	if err != nil {
		panic(err)
	}
	res := []string{}
	for _, file := range files {
		if !file.IsDir() {
			res = append(res, filepath.Join(path, file.Name()))
		}
	}
	return res
}
