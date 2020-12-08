package gin

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

type Runner interface {
	Run() (*exec.Cmd, error)
	StartDebugServer() (*exec.Cmd, error)
	StopDebugServer() error
	Info() (os.FileInfo, error)
	SetWriter(io.Writer)
	Kill() error
}

type runner struct {
	wd                 string
	bin                string
	args               []string
	writer             io.Writer
	appCommand         *exec.Cmd
	debugServerCommand *exec.Cmd
	debugServerPort    int
	starttime          time.Time
}

func NewRunner(wd string, bin string, debugServerPort int, args ...string) Runner {

	return &runner{
		wd:              wd,
		bin:             bin,
		args:            args,
		debugServerPort: debugServerPort,
		writer:          ioutil.Discard,
		starttime:       time.Now(),
	}
}

func (r *runner) Run() (*exec.Cmd, error) {
	if r.needsRefresh() {
		r.Kill()
	}

	if r.appCommand == nil || r.Exited() {
		err := r.runBin()
		if err != nil {
			log.Print("Error running: ", err)
		}
		time.Sleep(250 * time.Millisecond)
		return r.appCommand, err
	} else {
		return r.appCommand, nil
	}

}

func (r *runner) StartDebugServer() (*exec.Cmd, error) {
	if r.debugServerCommand != nil {
		if r.debugServerCommand.Process != nil {
			r.debugServerCommand.Process.Kill()
		}
		r.debugServerCommand = nil
	}

	if r.appCommand == nil && r.appCommand.Process == nil {
		return nil, nil
	}

	err := r.runDebugServer()

	return r.debugServerCommand, err
}

func (r *runner) StopDebugServer() error {
	if r.debugServerCommand != nil {
		if r.debugServerCommand.Process != nil {
			r.debugServerCommand.Process.Kill()
		}
		r.debugServerCommand = nil
	}

	return nil
}

func (r *runner) Info() (os.FileInfo, error) {
	return os.Stat(filepath.Join(r.wd, r.bin))
}

func (r *runner) SetWriter(writer io.Writer) {
	r.writer = writer
}

func (r *runner) Kill() error {
	if r.appCommand != nil && r.appCommand.Process != nil {
		done := make(chan error)
		go func() {
			r.appCommand.Wait()
			close(done)
		}()

		if r.debugServerCommand != nil {
			if r.debugServerCommand.Process != nil {
				r.debugServerCommand.Process.Kill()
			}
			r.debugServerCommand = nil
		}

		//Trying a "soft" kill first
		if runtime.GOOS == "windows" {
			if err := r.appCommand.Process.Kill(); err != nil {
				return err
			}
		} else if err := r.appCommand.Process.Signal(os.Interrupt); err != nil {
			return err
		}

		//Wait for our process to die before we return or hard kill after 3 sec
		select {
		case <-time.After(3 * time.Second):
			if err := r.appCommand.Process.Kill(); err != nil {
				log.Println("failed to kill: ", err)
			}
		case <-done:
		}
		r.appCommand = nil
	}

	return nil
}

func (r *runner) Exited() bool {
	return r.appCommand != nil && r.appCommand.ProcessState != nil && r.appCommand.ProcessState.Exited()
}

func (r *runner) runBin() error {
	r.appCommand = exec.Command(filepath.Join(r.wd, r.bin))
	stdout, err := r.appCommand.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := r.appCommand.StderrPipe()
	if err != nil {
		return err
	}

	err = r.appCommand.Start()
	if err != nil {
		return err
	}

	r.starttime = time.Now()

	go io.Copy(r.writer, stdout)
	go io.Copy(r.writer, stderr)
	go r.appCommand.Wait()

	return nil
}

func (r *runner) runDebugServer() error {
	if r.appCommand == nil || r.appCommand.Process == nil {
		return errors.New("app not running for debug server")
	}

	appPid := r.appCommand.Process.Pid
	if appPid == 0 {
		return errors.New("app pid is zero")
	}

	r.debugServerCommand = exec.Command("dlv", "attach", fmt.Sprint(appPid), r.wd, "--listen=:"+fmt.Sprint(r.debugServerPort), "--headless=true", "--continue", "--accept-multiclient", "--only-same-user=false", "--api-version=2", "--log")
	stdout, err := r.debugServerCommand.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := r.debugServerCommand.StderrPipe()
	if err != nil {
		return err
	}

	err = r.debugServerCommand.Start()
	if err != nil {
		return err
	}

	r.starttime = time.Now()

	go io.Copy(r.writer, stdout)
	go io.Copy(r.writer, stderr)
	go r.debugServerCommand.Wait()

	return nil
}

func (r *runner) needsRefresh() bool {
	info, err := r.Info()
	if err != nil {
		return false
	} else {
		return info.ModTime().After(r.starttime)
	}
}
