package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tyler-smith/go-bip39"
	"github.com/urfave/cli/v2"

	"github.com/LiskHQ/lisk-engine/pkg/codec"
	"github.com/LiskHQ/lisk-engine/pkg/collection/bytes"
	"github.com/LiskHQ/lisk-engine/pkg/crypto"
)

//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen

type keys struct {
	Passphrase string                   `json:"passphrase"`
	Password   string                   `json:"password,omitempty"`
	Address    codec.Lisk32             `json:"address"`
	KeyPath    string                   `json:"keyPath"`
	PublicKey  codec.Hex                `json:"publicKey"`
	PrivateKey codec.Hex                `json:"privateKey"`
	Plain      *plainKeys               `json:"plain"`
	Encrypted  *crypto.EncryptedMessage `json:"encrptedMessage"`
}

type plainKeys struct {
	GeneratorKeyPath     string    `json:"generatorKeyPath"`
	GeneratorKey         codec.Hex `json:"generatorPrivateKey" fieldNumber:"1"`
	GeneratorPrivateKey  codec.Hex `json:"generatorPublicKey" fieldNumber:"2"`
	BLSKeyPath           string    `json:"blsKeyPath"`
	BLSKey               codec.Hex `json:"blsPublicKey" fieldNumber:"3"`
	BLSSecretKey         codec.Hex `json:"blsPrivateKey" fieldNumber:"4"`
	BLSProofOfPossession codec.Hex `json:"blsProofOfPossession"`
}

func (a *keys) String() string {
	val := `
	Passphrase: %s
	Password: %s

	Address: %s
	KeyDerivation: %s
	PublicKey: %s
	PrivateKey: %s

	GeneratorKeyDerivation: %s
	GeneratorKey: %s
	PrivateKey: %s

	BLSKeyDerivation: %s
	BLSKey: %s
	BLSSecretKey: %s
	BLSProofOfPossession: %s

	EncryptedPassphrase: %s
	`
	return fmt.Sprintf(
		val,
		a.Passphrase,
		a.Password,
		a.Address,
		a.KeyPath,
		a.PublicKey,
		a.PrivateKey,
		a.Plain.GeneratorKeyPath,
		a.Plain.GeneratorKey,
		a.Plain.GeneratorPrivateKey,
		a.Plain.BLSKeyPath,
		a.Plain.BLSKey,
		a.Plain.BLSSecretKey,
		a.Plain.BLSProofOfPossession,
		a.Encrypted,
	)
}

func getDefaultKeyPath(offset int) string {
	return fmt.Sprintf("m/44'/134'/%d'", offset)
}

func getDefaultGeneratorKeyPath(chainID []byte, offset int) string {
	chainIDNum := bytes.ToUint32(chainID)
	return fmt.Sprintf("m/25519'/134'/%d'/%d'", chainIDNum, offset)
}

func getDefaultBLSKeyPath(chainID []byte, offset int) string {
	chainIDNum := bytes.ToUint32(chainID)
	return fmt.Sprintf("m/12381/134/%d/%d", chainIDNum, offset)
}

func generatePassphrase() (string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}
	passphrase, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", err
	}
	return passphrase, nil
}

func generateKeys(passphrase, password string, chainID codec.Hex, offset int) (*keys, error) {
	defaultKeyPath := getDefaultKeyPath(offset)
	defaultGeneratorKeyPath := getDefaultGeneratorKeyPath(chainID, offset)
	defaultBLSKeyPath := getDefaultBLSKeyPath(chainID, offset)

	privateKey, err := crypto.DeriveEd25519Key(passphrase, defaultKeyPath)
	if err != nil {
		return nil, err
	}
	publicKey := crypto.GetEdPublicKey(privateKey)

	generatorPrivateKey, err := crypto.DeriveEd25519Key(passphrase, defaultGeneratorKeyPath)
	if err != nil {
		return nil, err
	}
	generatorPublicKey := crypto.GetEdPublicKey(generatorPrivateKey)

	address := crypto.GetAddress(publicKey)

	blsPrivateKey, err := crypto.DeriveBLSKey(passphrase, defaultBLSKeyPath)
	if err != nil {
		return nil, err
	}
	blsPublicKey := crypto.BLSSKToPK(blsPrivateKey)
	blsPoP := crypto.BLSPopProve(blsPrivateKey)
	account := &keys{
		Passphrase: passphrase,
		Address:    address,
		KeyPath:    defaultKeyPath,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Plain: &plainKeys{
			GeneratorKeyPath:     defaultGeneratorKeyPath,
			GeneratorPrivateKey:  generatorPrivateKey,
			GeneratorKey:         generatorPublicKey,
			BLSKeyPath:           defaultBLSKeyPath,
			BLSKey:               blsPublicKey,
			BLSSecretKey:         blsPublicKey,
			BLSProofOfPossession: blsPoP,
		},
	}
	if password != "" {
		encodedKeys, err := account.Plain.Encode()
		if err != nil {
			return nil, err
		}
		encryptedMsg, err := crypto.EncryptMessageWithPassword(encodedKeys, password, crypto.DefaultEncryptOptions())
		if err != nil {
			return nil, err
		}
		account.Encrypted = encryptedMsg
		account.Password = password
	}

	return account, nil
}

func GetKeysCommand() *cli.Command {
	return &cli.Command{
		Name:  "account",
		Usage: "Account related commands",
		Subcommands: []*cli.Command{
			{
				Name:  "generate",
				Usage: "generate new account",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "passphrase",
						Usage: "Passphrase to use for keys",
					},
					&cli.StringFlag{
						Name:  "password",
						Usage: "Password to use for encryption",
					},
					&cli.StringFlag{
						Name:  "chain-id",
						Usage: "ChainID to use for keys",
					},
					&cli.IntFlag{
						Name:        "offset",
						Usage:       "Offset to use for key derivation",
						DefaultText: "0",
					},
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Print in JSON format",
					},
				},
				Action: func(c *cli.Context) error {
					passphrase := c.String("passphrase")
					if passphrase == "" {
						var err error
						passphrase, err = generatePassphrase()
						if err != nil {
							return err
						}
					}
					password := c.String("password")
					chainIDStr := c.String("chain-id")
					chainID := []byte{0, 0, 0, 0}
					if chainIDStr != "" {
						if len(chainIDStr) != 8 {
							return fmt.Errorf("invalid chain-id. chain-id must be 4 bytes hex string")
						}
						var err error
						chainID, err = hex.DecodeString(chainIDStr)
						if err != nil {
							return err
						}
					}
					offset := c.Int("offset")
					account, err := generateKeys(passphrase, password, chainID, offset)
					if err != nil {
						return err
					}
					if !c.Bool("json") {
						fmt.Println(account)
						return nil
					}
					jsonAccount, err := json.MarshalIndent(account, "", "  ")
					if err != nil {
						return err
					}
					fmt.Println(string(jsonAccount))
					return nil
				},
			},
			{
				Name:  "verify",
				Usage: "verify address",
				Action: func(c *cli.Context) error {
					if c.Args().Len() != 1 {
						return errors.New("must specify address to verify")
					}
					address := c.Args().First()

					if err := codec.ValidateLisk32(address); err != nil {
						return err
					}
					fmt.Printf("Address %s is valid \n", address)
					return nil
				},
			},
		},
	}
}
