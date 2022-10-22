package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/LiskHQ/lisk-engine/pkg/framework/config"
)

/*
*
/{name}
- main.go
- app.go
- /modules
- /plugins.
*/
func GetInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize blockchain application",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Specify output directory",
			},
		},
		Action: func(c *cli.Context) error {
			output := c.String("output")
			outputPath, err := os.Getwd()
			if err != nil {
				return err
			}
			if output != "" {
				var err error
				outputPath, err = resolvePath(output)
				if err != nil {
					return err
				}
			}
			if err := os.MkdirAll(outputPath, 0755); err != nil {
				return err
			}
			{
				byteWriter := bytes.NewBuffer([]byte{})
				if err := generateMain(&mainTemplateData{}, byteWriter); err != nil {
					return err
				}
				writer, err := os.Create(filepath.Join(outputPath, "main.go"))
				if err != nil {
					return err
				}
				defer writer.Close()
				formattedSorce, err := format.Source(byteWriter.Bytes())
				if err != nil {
					return err
				}
				_, err = writer.Write(formattedSorce)
				if err != nil {
					return err
				}
			}
			{
				byteWriter := bytes.NewBuffer([]byte{})
				if err := generateApp(&appTemplateData{}, byteWriter); err != nil {
					return err
				}
				writer, err := os.Create(filepath.Join(outputPath, "app.go"))
				if err != nil {
					return err
				}
				defer writer.Close()
				formattedSorce, err := format.Source(byteWriter.Bytes())
				if err != nil {
					return err
				}
				_, err = writer.Write(formattedSorce)
				if err != nil {
					return err
				}
			}
			// Create sample genesis block
			genesis, err := getDefaultGenesisBlock()
			if err != nil {
				return err
			}
			genesisJSON, err := json.MarshalIndent(genesis, "", "  ")
			if err != nil {
				return err
			}
			folderPath := outputPath
			if err := os.WriteFile(filepath.Join(folderPath, "genesis_block.json"), genesisJSON, 0755); err != nil { //nolint:gosec // ok to write as 0755
				return err
			}
			// Create config
			appConfig := &config.ApplicationConfig{}
			if err := appConfig.InsertDefault(); err != nil {
				return err
			}
			appConfig.Generator.Keys.FromFile = "./config/dev-validators.json"

			configJSON, err := json.MarshalIndent(appConfig, "", "  ")
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(folderPath, "config.json"), configJSON, 0755); err != nil { //nolint:gosec // ok to write as 0755
				return err
			}
			// create module and plugin folders
			if err := os.MkdirAll(filepath.Join(folderPath, "modules"), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(folderPath, "modules", ".gitkeep"), []byte{}, 0755); err != nil { //nolint:gosec // ok to write as 0755
				return err
			}
			if err := os.MkdirAll(filepath.Join(folderPath, "plugins"), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(folderPath, "plugins", ".gitkeep"), []byte{}, 0755); err != nil { //nolint:gosec // ok to write as 0755
				return err
			}
			fmt.Printf("Generated new application at %s\n", folderPath)
			return nil
		},
	}
}

func GetGenerateCommand() *cli.Command {
	commonFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "output",
			Aliases: []string{"o"},
			Usage:   "Specify output directory",
		},
		&cli.StringFlag{
			Name:     "name",
			Aliases:  []string{"n"},
			Usage:    "Specify name",
			Required: true,
		},
	}
	return &cli.Command{
		Name:  "generate",
		Usage: "Code generation",
		Subcommands: []*cli.Command{
			{
				Name:  "module",
				Usage: "generate new module",
				Flags: append(commonFlags, &cli.IntFlag{
					Name:     "id",
					Usage:    "Specify id",
					Required: true,
				}),
				Action: func(c *cli.Context) error {
					name := c.String("name")
					id := uint32(c.Int("id"))
					moduleData := &moduleTemplateData{
						Name: name,
						ID:   id,
					}
					endpointData := &endpointTemplateData{
						Name: name,
					}
					apiData := &apiTemplateData{
						Name: name,
					}
					output := c.String("output")
					if output == "" {
						writer := os.Stdout
						if err := generateModule(moduleData, writer); err != nil {
							return err
						}
						if err := generateEndpoint(endpointData, writer); err != nil {
							return err
						}
						if err := generateAPI(apiData, writer); err != nil {
							return err
						}
						return nil
					}
					outputPath, err := resolvePath(output)
					if err != nil {
						return err
					}
					folderPath := filepath.Join(outputPath, strings.ToLower(name))
					if err := os.MkdirAll(folderPath, 0755); err != nil {
						return err
					}
					// Write module
					{
						byteWriter := bytes.NewBuffer([]byte{})
						if err := generateModule(moduleData, byteWriter); err != nil {
							return err
						}
						writer, err := os.Create(filepath.Join(folderPath, "module.go"))
						if err != nil {
							return err
						}
						defer writer.Close()
						formattedSorce, err := format.Source(byteWriter.Bytes())
						if err != nil {
							return err
						}
						_, err = writer.Write(formattedSorce)
						if err != nil {
							return err
						}
					}
					{
						byteWriter := bytes.NewBuffer([]byte{})
						if err := generateEndpoint(endpointData, byteWriter); err != nil {
							return err
						}
						writer, err := os.Create(filepath.Join(folderPath, "endpoint.go"))
						if err != nil {
							return err
						}
						defer writer.Close()
						formattedSorce, err := format.Source(byteWriter.Bytes())
						if err != nil {
							return err
						}
						_, err = writer.Write(formattedSorce)
						if err != nil {
							return err
						}
					}
					{
						byteWriter := bytes.NewBuffer([]byte{})
						if err := generateAPI(apiData, byteWriter); err != nil {
							return err
						}
						writer, err := os.Create(filepath.Join(folderPath, "api.go"))
						if err != nil {
							return err
						}
						defer writer.Close()
						formattedSorce, err := format.Source(byteWriter.Bytes())
						if err != nil {
							return err
						}
						_, err = writer.Write(formattedSorce)
						if err != nil {
							return err
						}
					}
					fmt.Printf("Generated module %s at %s\n", name, folderPath)
					return nil
				},
			},
			{
				Name:  "command",
				Usage: "generate new command",
				Flags: append(commonFlags, &cli.IntFlag{
					Name:     "id",
					Usage:    "Specify id",
					Required: true,
				}, &cli.StringFlag{
					Name:     "module-name",
					Usage:    "module name",
					Required: true,
				}),
				Action: func(c *cli.Context) error {
					name := c.String("name")
					moduleName := c.String("module-name")
					id := uint32(c.Int("id"))
					commandData := &commandTemplateData{
						ModuleName: moduleName,
						Name:       name,
						ID:         id,
					}
					output := c.String("output")
					if output == "" {
						writer := os.Stdout
						if err := generateCommand(commandData, writer); err != nil {
							return err
						}
						return nil
					}
					outputPath, err := resolvePath(output)
					if err != nil {
						return err
					}
					folderPath := filepath.Join(outputPath, strings.ToLower(moduleName))
					if err := os.MkdirAll(folderPath, 0755); err != nil {
						return err
					}
					// Write module
					byteWriter := bytes.NewBuffer([]byte{})
					if err := generateCommand(commandData, byteWriter); err != nil {
						return err
					}
					commandFileName := fmt.Sprintf("%s_command.go", name)
					writer, err := os.Create(filepath.Join(folderPath, commandFileName))
					if err != nil {
						return err
					}
					defer writer.Close()
					formattedSorce, err := format.Source(byteWriter.Bytes())
					if err != nil {
						return err
					}
					_, err = writer.Write(formattedSorce)
					if err != nil {
						return err
					}
					fmt.Printf("Generated command %s at %s\n", commandFileName, folderPath)
					return nil
				},
			},
			{
				Name:  "plugin",
				Usage: "generate new plugin",
				Flags: commonFlags,
				Action: func(c *cli.Context) error {
					name := c.String("name")
					pluginData := &pluginTemplateData{
						Name: name,
					}
					output := c.String("output")
					if output == "" {
						writer := os.Stdout
						if err := generatePlugin(pluginData, writer); err != nil {
							return err
						}
						return nil
					}
					outputPath, err := resolvePath(output)
					if err != nil {
						return err
					}
					folderPath := filepath.Join(outputPath, strings.ToLower(name))
					if err := os.MkdirAll(folderPath, 0755); err != nil {
						return err
					}
					// Write module
					byteWriter := bytes.NewBuffer([]byte{})
					if err := generatePlugin(pluginData, byteWriter); err != nil {
						return err
					}
					commandFileName := fmt.Sprintf("%s_plugin.go", name)
					writer, err := os.Create(filepath.Join(folderPath, commandFileName))
					if err != nil {
						return err
					}
					defer writer.Close()
					formattedSorce, err := format.Source(byteWriter.Bytes())
					if err != nil {
						return err
					}
					_, err = writer.Write(formattedSorce)
					if err != nil {
						return err
					}
					fmt.Printf("Generated plugin %s at %s\n", commandFileName, folderPath)
					return nil
				},
			},
		},
	}
}

func resolvePath(output string) (string, error) {
	if filepath.IsAbs(output) {
		return output, nil
	}
	return filepath.Abs(output)
}
