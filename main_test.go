package main

import (
	"bytes"
	"encoding/json"
	iofs "io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/pandorasNox/lettr/pkg/session"
)

func TestGoVersionMatchesGoMod(t *testing.T) {
	// Get runtime version (e.g., "go1.21.3")
	runtimeVersion := strings.TrimPrefix(runtime.Version(), "go")
	runtimeMajorMinor := majorMinorPatch(runtimeVersion)

	// Get go.mod version using `go mod edit -json`
	cmd := exec.Command("go", "mod", "edit", "-json")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run `go mod edit -json`: %v", err)
	}

	var mod struct {
		Go string
	}
	if err := json.Unmarshal(out, &mod); err != nil {
		t.Fatalf("Failed to parse go.mod JSON: %v", err)
	}

	if runtimeMajorMinor != mod.Go {
		t.Errorf("Go version mismatch: runtime is %s, go.mod declares %s", runtimeMajorMinor, mod.Go)
	}
}

func majorMinorPatch(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return version
	}
	return parts[0] + "." + parts[1] + "." + parts[2]
}

// todo: test for ???:
//   files, err := getAllFilenames(staticFS)
//   log.Printf("  debug fsys:\n    %v\n    %s\n", files, err)

func Test_ExpectedEmbededFiles(t *testing.T) {
	expectedFiles := []string{
		"web/static/generated/main.js",
		"web/static/generated/output.css",
	}

	embededFiles, err := getAllFilenames(embedFs)
	if err != nil {
		t.Errorf("getAllFilenames() error = %v", err)
	}

	for _, expectedFile := range expectedFiles {
		if !slices.Contains(embededFiles, expectedFile) {
			t.Errorf("expected embeded files, got '%v', want '%v' but was not found", embededFiles, expectedFile)
		}
	}

}

func getAllFilenames(efs iofs.FS) (files []string, err error) {
	if err := iofs.WalkDir(efs, ".", func(path string, d iofs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		files = append(files, path)

		return nil
	}); err != nil {
		return nil, err
	}

	return files, nil
}

func Test_run(t *testing.T) {
	type args struct {
		sigChan          chan os.Signal
		shutdownDoneChan chan bool
		server           *http.Server
		sessions         session.Sessions
	}
	tests := []struct {
		name       string
		args       args
		wantOutput string
	}{
		// test cases
		{
			args: args{
				sigChan:          make(chan os.Signal, 1),
				shutdownDoneChan: make(chan bool, 2),
				server:           &http.Server{},
				sessions:         session.NewSessions(),
			},
			wantOutput: "Graceful shutdown complete.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var buf bytes.Buffer
			log.SetOutput(&buf)
			originalOutput := log.Writer()
			defer func() { log.SetOutput(originalOutput) }()

			tt.args.sigChan <- syscall.SIGTERM // keep in mind: this bypasses "signal.Notify" relay behaviour, on theory we can send anything here at this point
			run(tt.args.sigChan, tt.args.shutdownDoneChan, tt.args.server, tt.args.sessions)

			time.Sleep(1 * time.Second)

			output := buf.String()
			if !strings.Contains(output, tt.wantOutput) {
				t.Errorf("shutdown not complete, got '%v', want '%v' values", output, tt.wantOutput)
			}
		})
	}
}
