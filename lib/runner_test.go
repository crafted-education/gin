package gin_test

import (
	"bytes"
	"os"
	"runtime"
	"testing"
	"time"

	gin "github.com/crafted-education/gin/lib"
)

func Test_NewRunner(t *testing.T) {
	bin := "writing_output"
	if runtime.GOOS == "windows" {
		bin += ".bat"
	}
	wd := "test_fixtures"

	runner := gin.NewRunner(wd, bin, false)

	fi, _ := runner.Info()
	expect(t, fi.Name(), bin)
}

func Test_Runner_Run(t *testing.T) {
	wd := "test_fixtures"
	bin := "writing_output"
	if runtime.GOOS == "windows" {
		bin += ".bat"
	}
	runner := gin.NewRunner(wd, bin, false)

	cmd, err := runner.Run()
	expect(t, err, nil)
	expect(t, cmd.Process == nil, false)
}

// func Test_Runner_SettingEnvironment(t *testing.T) {
// }

func Test_Runner_Kill(t *testing.T) {
	wd := "test_fixtures"
	bin := "writing_output"
	if runtime.GOOS == "windows" {
		bin += ".bat"
	}

	runner := gin.NewRunner(wd, bin, false)

	cmd1, err := runner.Run()
	expect(t, err, nil)

	_, err = runner.Run()
	expect(t, err, nil)

	time.Sleep(time.Second * 1)
	os.Chtimes(bin, time.Now(), time.Now())
	if err != nil {
		t.Fatal("Error with Chtimes")
	}

	cmd3, err := runner.Run()
	expect(t, err, nil)

	if runtime.GOOS != "windows" {
		// does not seem to work as expected on windows
		refute(t, cmd1, cmd3)
	}
}

func Test_Runner_SetWriter(t *testing.T) {
	buff := bytes.NewBufferString("")
	expect(t, buff.String(), "")

	wd := "test_fixtures"
	bin := "writing_output"
	if runtime.GOOS == "windows" {
		bin += ".bat"
	}

	runner := gin.NewRunner(wd, bin, false)
	runner.SetWriter(buff)

	cmd, err := runner.Run()
	cmd.Wait()
	expect(t, err, nil)

	if runtime.GOOS == "windows" {
		expect(t, buff.String(), "Hello world\r\n")
	} else {
		expect(t, buff.String(), "Hello world\n")
	}
}
