package onionbalancedaemon

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
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
			log.Println("starting onionbalance...")

			t.cmd = exec.CommandContext(t.ctx,
				"onionbalance",
				"--config", "/run/onionbalance/config.yaml",
				// "--verbosity", "debug",
				"--ip", "127.0.0.1",
				"--port", "9051",
				"--hs-version", "v3",
			)
			t.cmd.Stdout = os.Stdout
			t.cmd.Stderr = os.Stderr

			err := t.cmd.Start()
			if err != nil {
				log.Print("error starting onionbalance: ", err)
			}

			err = t.cmd.Wait()
			if err != nil {
				log.Print("error running onionbalance: ", err)
			}

			//nolint:gomnd // just seconds
			time.Sleep(time.Second * 3)
		}
	}()
}

func (t *OnionBalance) IsRunning() bool {
	return t.cmd != nil && (t.cmd.ProcessState == nil || !t.cmd.ProcessState.Exited())
}

func (t *OnionBalance) EnsureRunning() {
	if !t.IsRunning() {
		log.Println("onionbalance is not running...")
		t.Start()
	}
}

func (t *OnionBalance) Reload() {
	log.Println("reloading onionbalance...")

	if t.IsRunning() {
		log.Println("stopping existing onionbalance...")

		err := t.cmd.Process.Signal(syscall.SIGHUP)
		if err != nil {
			log.Println()
		}

		err = t.cmd.Wait()
		if err != nil {
			log.Println("error stopping onionbalance: ", err)
		}
	}

	t.Start()
}
