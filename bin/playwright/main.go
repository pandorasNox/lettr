package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"

	pw "github.com/pandorasNox/lettr/pkg/playwright"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %s", err)
	}
}

func run() error {
	log.Println("start...")

	testDir := ""
	reportDir := ""
	flag.StringVar(&testDir, "testDir", "./playwright", "pass absolue or relative (to working dir) path to the playwright test directory")
	flag.StringVar(&reportDir, "reportDir", "./playwright/reports", "pass absolue or relative (to working dir) path to the playwright report directory")

	flag.Parse()

	log.Printf("using testDir='%s'\n", testDir)
	log.Printf("using reportDir='%s'\n", reportDir)

	// testDirPath := "./tests/playwright"
	absTestDirPath, err := resolvePath(testDir)
	if err != nil {
		return fmt.Errorf("couldn't resolve absolute path for testDir with path='%s' error='%s'", absTestDirPath, err)
	}
	if !directoryExists(absTestDirPath) {
		return fmt.Errorf("testDir directory with path='%s' does not exist", absTestDirPath)
	}

	// reportDirPath := "./tests/playwright/playwright-report"
	absReportDirPath, err := resolvePath(reportDir)
	if err != nil {
		return fmt.Errorf("couldn't resolve absolute path for reportDir with path='%s' error='%s'", absReportDirPath, err)
	}
	if !directoryExists(absReportDirPath) {
		return fmt.Errorf("reportDir directory with path='%s' does not exist", absReportDirPath)
	}

	// ------------------------------------------------------------------------

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	net, err := network.New(ctx)
	if err != nil {
		return fmt.Errorf("couldn't create network: %s", err)
	}
	defer func() {
		if err := net.Remove(ctx); err != nil {
			log.Printf("failed to remove network: %s", err)
		}
	}()

	lettrAppContainer, err := NewLettrAppContainer(ctx, *net)
	if err != nil {
		return fmt.Errorf("couldn't create lettr app container: %s", err)
	}
	defer lettrAppContainer.Terminate(ctx)

	pwContainer, err := pw.NewPlaywrightContainer(ctx, absTestDirPath, absReportDirPath, net.Name)
	if err != nil {
		return fmt.Errorf("couldn't create playwright container: %s", err)
	}
	defer pwContainer.Terminate()

	logs, err := pwContainer.RunTests()
	if err != nil {
		return fmt.Errorf("running tests failed with error='%s' and logs='%s'", err, logs)
	}
	log.Printf("playwright test run logs: \n%s\n", logs)

	return nil
}

// resolvePath returns the absolute path for a given input.
// If the input is already absolute, it normalizes it.
// If it's relative, it joins it with the current working directory.
func resolvePath(p string) (string, error) {
	absPath := p
	if !filepath.IsAbs(p) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		absPath = filepath.Join(cwd, p)
	}

	return filepath.Abs(absPath)
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil && info.IsDir()
}

func NewLettrAppContainer(ctx context.Context, net tc.DockerNetwork) (container tc.Container, err error) {
	log.Print("create lettr app container")

	// Create a child context with a 30-second timeout
	childCtx, cancelChild := context.WithTimeout(ctx, 30*time.Second)
	defer cancelChild()

	maybePort := "80"
	port, err := nat.NewPort("tcp", maybePort)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse port with port input='%s': %s", maybePort, err)
	}

	cr := tc.ContainerRequest{
		FromDockerfile: tc.FromDockerfile{
			// relative to repo root dir
			Context:    ".",
			Dockerfile: "container-images/app/Dockerfile",
			// KeepImage: true,
		},
		ExposedPorts: []string{"80"},
		Networks:     []string{net.Name},
		NetworkAliases: map[string][]string{
			net.Name: []string{
				"lettrapp.aliases.containernetwork",
			},
		},
		Env: map[string]string{
			"PORT":         "80",
			"GITHUB_TOKEN": "my-secret-token-for-local-dev",
			"IMPRINT_URL":  "http://www.example.com/imprint",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(port).WithStartupTimeout(60 * time.Second),
		),
	}

	container, err = tc.GenericContainer(childCtx, tc.GenericContainerRequest{
		ContainerRequest: cr,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't start lettr app container: %s", err)
	}

	return container, nil
}

func execContaincerCmd(ctx context.Context, c tc.Container, cmd []string) (exitCode int, cmdLog string, err error) {
	exitCode, cmdExecLogReader, err := c.Exec(ctx, cmd)
	if err != nil {
		return -1, "", fmt.Errorf("couldn't exec cmd '%s' in container: %s", strings.Join(cmd, " "), err)
	}

	execCmdLog, err := io.ReadAll(cmdExecLogReader)
	if err != nil {
		return -1, "", fmt.Errorf("couldn't read exec cmd log: %s", err)
	}
	cmdLog = string(execCmdLog)

	if exitCode != 0 {
		return -1, "", fmt.Errorf("exec cmd in container failed with non zero exit code: exitCode=%d \n  logs:\n\n%s", exitCode, cmdLog)
	}

	return exitCode, cmdLog, nil
}
