# Plan: ping

This test plan is a simple usage of `testground` for `lisk-engine/pkg/p2p`.

## Installation & Setup

Please follow the instructions to install `testground` [here](https://docs.testground.ai/v/master/table-of-contents/getting-started) and run the example of the `testcase` to prepare Docker container for current `testplan`. Make sure that you have `testground-goproxy` in your Docker container. 

### Requirements
After installing the `testground` you need to have running `testground daemon` by the following command:
```bash
$ testground deamon
```

## Import testplan
The `testground` client will look for `testplans` in `TESTGROUND_HOME/plans` (default path). You have two options to import a new `testplan` as described below:
 - The `testplan` will draw a chart of result and if you want to see them, please `copy` from `your-path/lisk-engine/test/test-plans/ping` and `paste` it into `TESTGROUND_HOME/plans` and go to the next section.
 - To **import** a new`testplan` run the following command:
    ```bash
    testground plan import --from your-path/lisk-engine/test/test-plans/ping --name ping
    ```

## Running Tests
### Build and Run without Docker
Please follow the below steps to prepare your machine to run the `testcase`:
1. Install `ssh` client on you local machine.
2. In case you want to make a new `ssh key` run: `ssh-keygen -b 4096 -t rsa`
3. When you have `ssh key`, then run the following commands:
	```bash
	eval "$(ssh-agent -s)"
	ssh-add ~/.ssh/your-key
	```
4. Then you need to config `git` to use `ssh` instead of `https`, so run the following command for that:
	```bash
	git config  --global url.ssh://git@github.com/.insteadOf https://github.com/
	```
5. To make sure everything goes well the result of the following command should be `PTY allocation request failed on channel 0` :
	```bash
	ssh -t git@github.com
	```
6. Now you can run the following commands to fetch the private repository of the `lisk-engine`:
	```bash
	export GOPRIVATE=*
	go get -v github.com/LiskHQ/lisk-engine/pkg/p2p/v2@c1dfe1a32115d24cbf94bd5b5522aaa06cd17fcd
	```
7. After fetching the `pkg` you should turn `git config` to use `https`, for this you can remove the `~/.gitconfig` or comment the lines that you recently added it, put `;` at the beginning of the each lines.

Now you can run the `testcase` by the following command in your local machine:
```bash
testground run single --plan=ping --testcase=ping --runner=local:exec --builder=exec:go --instances=10
```

### Build and Run with Docker
- Make sure that you have `testground-goproxy` in your container list by the following command:
	```bash
	docker container ls
	```
- If you do not have `testground-goproxy` you should go back to **Installation & Setup** section.
- If you want to use your own `ssh key` for docker container, you can run the following command in your local terminal machine: `docker cp path/to/key testground-goproxy:~/.ssh`
- To be able to use `docker` as a builder you need to login into `testground-goproxy` container first and then install `ssh` client by the following command:
  ```bash
  apk add openssh
  ```
- After installation follow the steps of `2 to 7` in **Build and Run without Docker** section to prepare the container.

When your configuration is finished, run the following command to run the `testplan`:
```bash
testground run single --plan=ping --testcase=ping --runner=docker:docker --builder=docker:go --instances=10
```

## Test parameters
|Name|Type|Description|Default|
|:----|:----:|:----|:----:|
|secure_channel|enum|secure channel used|noise|
|max_latency_ms|int|maximum value for random link latency| 1000|
|iterations|int|number of ping iterations we will run against each peer| 10|
#### The following values are available for the `secure_channel`:
 - noise
 - tls
 - none

To change the default parameters, you can pass the parameters like below command:
```bash
testground run single --plan=ping --testcase=ping --runner=local:exec --builder=exec:go --instances=10 --test-param `param-name`=`value`
```

## Result and Chart
After running the `testcase` you can navigate to `testground daemon` `dashboard` to see a `chart`. Default address is `http://localhost:8042/tasks`.
