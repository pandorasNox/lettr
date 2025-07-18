package playwright

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type PlaywrightContainer struct {
	ctx       context.Context
	container tc.Container
	reportDir string
}

// Setup the container
func NewPlaywrightContainer(ctx context.Context, testDir string, reportDir string, netName string) (*PlaywrightContainer, error) {
	if !filepath.IsAbs(testDir) {
		return nil, fmt.Errorf("passed path for testDir (dir='%s') is not absolute", testDir)
	}

	if !filepath.IsAbs(reportDir) {
		return nil, fmt.Errorf("passed path for reportDir (dir='%s') is not absolute", reportDir)
	}

	req := tc.ContainerRequest{
		Image:  "mcr.microsoft.com/playwright:v1.54.1-noble",
		Mounts: tc.Mounts(tc.BindMount(testDir, "/tests")),
		// Files: ,
		WorkingDir: "/tests",
		WaitingFor: wait.ForAll(
			wait.ForLog("Running").WithStartupTimeout(60 * time.Second),
		),
		ExposedPorts: []string{},
		LogConsumerCfg: &tc.LogConsumerConfig{
			Opts:      []tc.LogProductionOption{tc.WithLogProductionTimeout(10 * time.Second)},
			Consumers: []tc.LogConsumer{&tc.StdoutLogConsumer{}},
		},
		Env: map[string]string{
			"CI": "true",
		},
		Entrypoint: []string{"/bin/bash"},
		Cmd:        []string{"-c", "echo 'Running...'; while true; do sleep $((60*60*24)); done"},
	}

	if netName != "" {
		req.Networks = append(req.Networks, netName)
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		if container == nil {
			return nil, fmt.Errorf("couldn't create playwright container error='%s'", err)
		}

		logReadCloser, logErr := container.Logs(ctx)
		if logErr != nil {
			return nil, fmt.Errorf("couldn't create playwright container error='%s', logError='%s'", err, logErr)
		}

		log, logReadErr := io.ReadAll(logReadCloser)
		if logReadErr != nil {
			return nil, fmt.Errorf("couldn't create playwright container error='%s', logReadError='%s'", err, logReadErr)
		}

		return nil, fmt.Errorf("couldn't create playwright container error='%s', containerLogs='%s'", err, log)
	}

	return &PlaywrightContainer{
		ctx:       ctx,
		container: container,
		reportDir: reportDir,
	}, nil
}

// Execute the tests
func (pc *PlaywrightContainer) RunTests() (logs string, err error) {
	log.Println("run playwright tests...")

	exitCode, outputReader, err := pc.container.Exec(pc.ctx, []string{
		// "npx", "playwright", "test", "--reporter=html",
		// "echo", "start...", "&&", "npm", "ci", "&&", "npx", "playwright", "test", "--reporter=html", "&&", "echo", "done",
		// "/usr/bin/env", "sh", "-c", "echo start; echo done;",
		"/usr/bin/env", "sh", "-c", "echo start && npm ci && npx playwright test && echo done",
	})
	if err != nil {
		return "", fmt.Errorf("test run playwright - container exec err: %s", err)
	}

	outputBytes, err := io.ReadAll(outputReader)
	if err != nil && exitCode != 0 {
		return "", fmt.Errorf("test run playwright - faild with exit code='%d' & could not read exec logs: %s", exitCode, err)
	}
	if err != nil && exitCode == 0 {
		return "", fmt.Errorf("test run playwright - could not read exec logs: %s", err)
	}

	execLogs := string(outputBytes)

	if exitCode != 0 {
		return execLogs, fmt.Errorf("test run playwright - failed with exit code='%d'", exitCode)
	}

	return execLogs, nil
}

// SaveReport copies a single file (e.g. HTML report) from the container to the host machine
func (pc *PlaywrightContainer) SaveReport(containerPath string, hostPath string) error {
	log.Println("saving playwright reports...")

	// Copy file from container
	reader, err := pc.container.CopyFileFromContainer(pc.ctx, containerPath)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Create destination file on host
	outFile, err := os.Create(hostPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Copy content from container file to host file
	_, err = io.Copy(outFile, reader)
	return err
}

// Stop container
func (pc *PlaywrightContainer) Terminate() error {
	log.Println("terminating playwright container...")
	return pc.container.Terminate(pc.ctx)
}
