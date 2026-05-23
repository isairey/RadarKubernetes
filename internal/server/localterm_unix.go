//go:build !windows

package server

import (
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/creack/pty"
)

// unixPty wraps the master end of a Unix pseudo-terminal
type unixPty struct {
	f *os.File
}

func (p *unixPty) Read(b []byte) (int, error)  { return p.f.Read(b) }
func (p *unixPty) Write(b []byte) (int, error) { return p.f.Write(b) }
func (p *unixPty) Close() error                { return p.f.Close() }

func (p *unixPty) Resize(cols, rows uint16) error {
	return pty.Setsize(p.f, &pty.Winsize{Rows: rows, Cols: cols})
}

func getDefaultShell() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	if runtime.GOOS == "darwin" {
		return "/bin/zsh"
	}
	return "/bin/bash"
}

func startShell(shell string, env []string, dir string) (*shellProcess, error) {
	cmd := exec.Command(shell, "-l")
	cmd.Env = env
	cmd.Dir = dir

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}
	return &shellProcess{
		pty:     &unixPty{f: ptmx},
		process: cmd.Process,
	}, nil
}

func signalProcess(p *os.Process) {
	p.Signal(syscall.SIGHUP)
}

func waitProcess(proc *shellProcess) {
	// On Unix, we need to wait for the child process to avoid zombies.
	// The process was started via exec.Cmd, but we only hold *os.Process here.
	proc.process.Wait()
}
