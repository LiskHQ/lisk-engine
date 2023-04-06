package main

import (
	"context"

	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
)

var testcases = map[string]interface{}{
	"evaluate": run.InitializedTestCaseFn(runPubsub),
}

func main() {
	run.InvokeMap(testcases)
}

func runPubsub(runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	params := parseParams(runenv)

	totalTime := params.setup + params.runtime + params.warmup + params.cooldown
	ctx, cancel := context.WithTimeout(context.Background(), totalTime)
	defer cancel()

	initCtx.MustWaitAllInstancesInitialized(ctx)

	return RunSimulation(ctx, initCtx, runenv, params)
}
