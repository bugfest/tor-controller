package tordaemon

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
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
			log.Println("starting tor...")

			t.cmd = exec.CommandContext(t.ctx,
				"tor",
				"-f", "/run/tor/torfile",
				// "--allow-missing-torrc",
			)

			t.cmd.Stdout = os.Stdout
			t.cmd.Stderr = os.Stderr

			err := t.cmd.Start()
			if err != nil {
				log.Print(err)
			}

			err = t.cmd.Wait()
			if err != nil {
				log.Print(err)
			}

			//nolint:gomnd // 3 seconds
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
		log.Println("reloading tor...")
		// https://manpages.debian.org/testing/tor/tor.1.en.html#SIGNALS
		// SIGHUP tells tor to reload the config
		err := t.cmd.Process.Signal(syscall.SIGHUP)
		if err != nil {
			log.Print("error sending SIGHUP to tor: ", err)
		}
	}
}
