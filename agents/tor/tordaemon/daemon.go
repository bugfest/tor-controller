package tordaemon

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type Tor struct {
	cmd *exec.Cmd
	ctx context.Context
}

func (t *Tor) SetContext(ctx context.Context) {
	t.ctx = ctx
}

func (t *Tor) Start() {
	go func() {
		for {
			fmt.Println("starting tor...")
			t.cmd = exec.CommandContext(t.ctx,
				"tor",
				"-f", "/run/tor/torfile",
				// "--allow-missing-torrc",
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

func (t *Tor) Reload() {

	if t.cmd == nil || (t.cmd.ProcessState != nil && t.cmd.ProcessState.Exited()) {
		// tor is not running
		t.Start()
	} else {

		// restart if already running
		fmt.Println("reloading tor...")
		// https://manpages.debian.org/testing/tor/tor.1.en.html#SIGNALS
		// SIGHUP tells tor to reload the config
		t.cmd.Process.Signal(syscall.SIGHUP)
	}
}
