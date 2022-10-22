package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBLSKeyGen(t *testing.T) {
	keys := BLSKeyGen([]byte("sampleaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	assert.NotNil(t, keys.PublicKey)
	assert.NotNil(t, keys.PrivateKey)
	keys = BLSRandomKeyGen()
	assert.NotNil(t, keys.PublicKey)
	assert.NotNil(t, keys.PrivateKey)
}

func TestBLSSignVerify(t *testing.T) {
	keys := BLSRandomKeyGen()
	signature := BLSSign([]byte("some msg"), keys.PrivateKey)
	assert.NotNil(t, signature)
	valid := BLSVerify([]byte("some msg"), signature, keys.PublicKey)
	assert.True(t, valid)
}

func TestSKToPK(t *testing.T) {
	fixtureFiles := readFolder("./fixtures/bls_specs/sk_to_pk")
	for _, fixtureFile := range fixtureFiles {
		fixture := &hexInOutFixture{}
		loadYaml(fixtureFile, fixture)
		output := BLSSKToPK(strToHex(fixture.Input))
		assert.Equal(t, strToHex(fixture.Output), output)
	}
}

func TestBLSPopProve(t *testing.T) {
	fixtureFiles := readFolder("./fixtures/bls_specs/pop_prove")
	for _, fixtureFile := range fixtureFiles {
		t.Log("Testing", fixtureFile)
		fixture := &hexInOutFixture{}
		loadYaml(fixtureFile, fixture)
		output := BLSPopProve(strToHex(fixture.Input))
		assert.Equal(t, strToHex(fixture.Output), output)
	}
}

func TestBLSPopVerify(t *testing.T) {
	fixtureFiles := readFolder("./fixtures/bls_specs/pop_verify")
	for _, fixtureFile := range fixtureFiles {
		t.Log("Testing", fixtureFile)
		fixture := &popVerifyFixture{}
		loadYaml(fixtureFile, fixture)
		output := BLSPopVerify(strToHex(fixture.Input.PrivKey), strToHex(fixture.Input.Proof))
		assert.Equal(t, fixture.Output, output)
	}
}

func TestBLSSign(t *testing.T) {
	fixtureFiles := readFolder("./fixtures/bls_specs/sign")
	for _, fixtureFile := range fixtureFiles {
		t.Log("Testing", fixtureFile)
		fixture := &signFixture{}
		loadYaml(fixtureFile, fixture)
		output := BLSSign(strToHex(fixture.Input.Message), strToHex(fixture.Input.PrivKey))
		assert.Equal(t, strToHex(fixture.Output), output)
	}
}

func TestBLSVerify(t *testing.T) {
	fixtureFiles := readFolder("./fixtures/bls_specs/verify")
	for _, fixtureFile := range fixtureFiles {
		t.Log("Testing", fixtureFile)
		fixture := &verifyFixture{}
		loadYaml(fixtureFile, fixture)
		output := BLSVerify(strToHex(fixture.Input.Message), strToHex(fixture.Input.Signature), strToHex(fixture.Input.PubKey))
		assert.Equal(t, fixture.Output, output)
	}
}

func TestCreateAggSig(t *testing.T) {
	fixtureFiles := readFolder("./fixtures/bls_specs/create_agg_sig")
	for _, fixtureFile := range fixtureFiles {
		t.Log("Testing", fixtureFile)
		fixture := &createAggSigFixture{}
		loadYaml(fixtureFile, fixture)
		keylist := make([][]byte, len(fixture.Input.KeyList))
		for i, val := range fixture.Input.KeyList {
			keylist[i] = strToHex(val)
		}
		keypair := make([]*BLSPublicKeySignaturePair, len(fixture.Input.KeySigPairs))
		for i, val := range fixture.Input.KeySigPairs {
			keypair[i] = &BLSPublicKeySignaturePair{
				PublicKey: strToHex(val.PublicKey),
				Signature: strToHex(val.Signature),
			}
		}
		aggBits, signature := BLSCreateAggSig(keylist, keypair)
		assert.Equal(t, strToHex(fixture.Output.AggrigationBits), aggBits)
		assert.Equal(t, strToHex(fixture.Output.Signature), signature)
	}
}

func TestVerifyAggSig(t *testing.T) {
	fixtureFiles := readFolder("./fixtures/bls_specs/verify_agg_sig")
	for _, fixtureFile := range fixtureFiles {
		t.Log("Testing", fixtureFile)
		fixture := &verifyAggSigFixture{}
		loadYaml(fixtureFile, fixture)
		keylist := make([][]byte, len(fixture.Input.KeyList))
		for i, val := range fixture.Input.KeyList {
			keylist[i] = strToHex(val)
		}
		mesage := append([]byte(fixture.Input.Tag), strToHex(fixture.Input.NetID)...)
		mesage = append(mesage, strToHex(fixture.Input.Message)...)
		output := BLSVerifyAggSig(
			keylist,
			strToHex(fixture.Input.AggrigationBits),
			strToHex(fixture.Input.Signature),
			mesage,
		)
		assert.Equal(t, fixture.Output, output)
	}
}

func TestVerifyWeightedAggSig(t *testing.T) {
	fixtureFiles := readFolder("./fixtures/bls_specs/verify_weighted_agg_sig")
	for _, fixtureFile := range fixtureFiles {
		t.Log("Testing", fixtureFile)
		fixture := &verifyWeightedAggSigFixture{}
		loadYaml(fixtureFile, fixture)
		keylist := make([][]byte, len(fixture.Input.KeyList))
		for i, val := range fixture.Input.KeyList {
			keylist[i] = strToHex(val)
		}
		mesage := append([]byte(fixture.Input.Tag), strToHex(fixture.Input.NetID)...)
		mesage = append(mesage, strToHex(fixture.Input.Message)...)
		output := BLSVerifyWeightedAggSig(
			keylist,
			strToHex(fixture.Input.AggrigationBits),
			strToHex(fixture.Input.Signature),
			fixture.Input.Weights,
			fixture.Input.Threshold,
			mesage,
		)
		assert.Equal(t, fixture.Output, output)
	}
}
