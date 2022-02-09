package onionbalancedaemon

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type OnionBalance struct {
	cmd *exec.Cmd
	ctx context.Context
}

func (t *OnionBalance) SetContext(ctx context.Context) {
	t.ctx = ctx
}

func (t *OnionBalance) Start() {
	go func() {
		for {
			fmt.Println("starting onionbalance...")
			t.cmd = exec.CommandContext(t.ctx,
				"onionbalance",
				"--config", "/run/onionbalance/config.yaml",
				// "--verbosity", "debug",
				"--ip", "127.0.0.1",
				"--port", "6666",
				"--hs-version", "v3",
			)
			t.cmd.Stdout = os.Stdout
			t.cmd.Stderr = os.Stderr

			err := t.cmd.Start()
			if err != nil {
				fmt.Print(err)
			}
			t.cmd.Wait()
			time.Sleep(time.Second * 3)
		}
	}()
}

func (t *OnionBalance) Reload() {
	fmt.Println("reloading onionbalance...")

	// start if not already running
	if t.cmd == nil || (t.cmd.ProcessState != nil && t.cmd.ProcessState.Exited()) {
		t.Start()
	} else {
		t.cmd.Process.Signal(syscall.SIGHUP)
	}
}
