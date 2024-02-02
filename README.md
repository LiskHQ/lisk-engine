![Logo](./docs/assets/banner_engine.png)

# Lisk Engine

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](http://www.apache.org/licenses/LICENSE-2.0)
[![Build status](https://github.com/LiskHQ/lisk-engine/actions/workflows/pr.yaml/badge.svg)](https://github.com/LiskHQ/lisk-engine/actions/workflows/pr.yaml)
![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/liskhq/lisk-engine)
![GitHub repo size](https://img.shields.io/github/repo-size/liskhq/lisk-engine)
![GitHub issues](https://img.shields.io/github/issues-raw/liskhq/lisk-engine)
![GitHub closed issues](https://img.shields.io/github/issues-closed-raw/liskhq/lisk-engine)

Lisk Engine is a consensus engine written in Golang implementing the Lisk v4 protocol.

## Requirements

- Go v1.21 or above

## Development

### Linting

[golangci-lint](https://golangci-lint.run/) is used for linting. To setup the local environment, refer to [Editor Integration](https://golangci-lint.run/usage/integrations/) in the usage guide.

### Documentation

Execute the folllowing command to view the generated documentation locally:

```bash
make godocs
```

### Usage

Execute the folllowing command to run Lisk Engine against a known ABI server:

```bash
make run.lengine PATH_TO_ABI_SERVER PATH_TO_CONFIG

# Example
# make run.lengine path=~/.lisk/dpos-mainchain/tmp/sockets/abi.ipc config=./cmd/debug/app/config.json
```

### Codecs

To generate a new codec:

1. Define a new struct with a tag on each property `fieldNumber: "n"`
2. Add `//go:generate go run github.com/LiskHQ/lisk-engine/pkg/codec/gen` at the top of the file
3. Call `make generate.codec`

### Tests

To run the tests:

* `make test` - To run all the tests and show a summary
* `make test.coverage` - To run all the tests and show a detailed coverage report
* `make test.coverage.html` - To run all the tests and produce a HTML coverage report

## Contributors

https://github.com/LiskHQ/lisk-engine/graphs/contributors

## Disclaimer

> [!WARNING]
> By using the source code of Lisk Engine, you acknowledge and agree that you have an adequate understanding of the risks associated with the use of the source code of Lisk Engine and that it is provided on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. To the fullest extent permitted by law, in no event shall the Lisk Foundation or other parties involved in the development of Lisk Engine have any liability whatsoever to any person for any direct or indirect loss, liability, cost, claim, expense or damage of any kind, whether in contract or in tort, including negligence, or otherwise, arising out of or related to the use of all or part of the source code of Lisk Engine.

## License

Copyright 2016-2024 Lisk Foundation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.