# Plan: gossipsub

This test plan designed to evaluate the performance of p2p.Gossipsub under various attack scenarios.

## Installation & Setup

We're using python to generate testground composition files, and we shell out to a few 
external commands, so there's some environment setup to do.

### Requirements

#### Hardware

While no special hardware is needed to run the tests, running with a lot of test instances requires considerable CPU and
RAM, and will likely exceed the capabilities of a single machine.

A large workstation with many CPU cores can reasonably run a few hundred instances using the testground
`local:docker` runner, although the exact limit will require some trial and error. In early testing we were able to
run 500 containers on a 56 core Xeon W-3175X with 124 GiB of RAM, although it's possible we could run more
now that we've optimized things a bit.

It's useful to run with the `local:docker` or `local:exec` runners during test development, so long as you use
fairly small instance counts (~22 or so works fine on a 2021 iMac with 16 GB RAM).

When using the `local:docker` runner, it's a good idea to periodically garbage collect the docker images created by
testground using `docker system prune` to reclaim disk space.

To create your own cluster, follow the directions in the [testground/infra repo](https://github.com/testground/infra)
to provision a cluster on AWS, and configure your tests to use the `cluster:k8s` test runner.

The testground daemon process can be running on your local machine, as long as it has access to the k8s cluster.
The machine running the daemon must have Docker installed, and the user account must have permission to use
docker and have the correct AWS credentials to connect to the cluster. 
When running tests on k8s, the machine running the testground daemon doesn't need a ton of resources,
but ideally it should have a fast internet connection to push the Docker images to the cluster.

### Testground

You'll need to have the [testground](https://github.com/testground/testground#getting-started) binary built and accessible
on your `$PATH`. As of the time of writing, we're running from `master`.

Default `testground` home is deprecated. Please change it with the following command or use your own path:
```bash
echo "export TESTGROUND_HOME='your-home-path/testground'" >> ~/.bashrc
source ~/.bashrc
```

#### Private repo setup

For the private repo you need to setup `ssh key` and `git` in your local machine or docker, please follow the instructions based on your choice [here](https://github.com/LiskHQ/lisk-engine/blob/main/test/test-plans/ping/README.md#running-tests).
**Important:** You have to change the commit version please run the following command after your setup:
//TODO commit version should remove after merge with main branch.
```bash
go get -v github.com/LiskHQ/lisk-engine/pkg/p2p@040fbea90b89ccce8fcdc95ba3ec301386020bec
```

#### Import testplan

To **import** a new`testplan` run the following command:
```bash
testground plan import --from your-path/lisk-engine/test/test-plans/gossipsub --name gossipsub
```

## Running Tests
You can modify the parameters in `composition.toml` to make your own config, after the please run the following commands to run the test-plan:
```bash
testground run composition -f your-path/lisk-engine/test/test-plans/gossipsub/composition.toml  --collect -o path-to-output
```
