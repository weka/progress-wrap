package runner

import (
	"bytes"
	"io"
	"os"
	"os/exec"
)

// Run executes name with args, streaming its stdout to os.Stdout while also
// capturing it. Stderr is passed through to os.Stderr but not captured.
// Returns captured stdout bytes, the process exit code, and any exec error.
// A non-zero exit code is NOT returned as an error.
func Run(name string, args []string) (stdout []byte, exitCode int, err error) {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &buf)
	cmd.Stderr = os.Stderr

	if runErr := cmd.Run(); runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return buf.Bytes(), exitErr.ExitCode(), nil
		}
		return nil, -1, runErr
	}
	return buf.Bytes(), 0, nil
}
