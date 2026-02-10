// Package tests contains end-to-end integration tests.
//
// TestE2E_Docker runs a full flow inside Docker: server, app, and client containers
// orchestrated by docker-compose and scripts in tests/docker/scripts/.
package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestE2E_Docker runs the Docker-based e2e: spawns server, app, and client via
// docker-compose and run-e2e.sh, then asserts the public URL returns the app body.
// Requires Docker and docker compose. Skip on non-Linux or when FWDX_SKIP_DOCKER_E2E=1.
func TestE2E_Docker(t *testing.T) {
	if os.Getenv("FWDX_SKIP_DOCKER_E2E") == "1" {
		t.Skip("FWDX_SKIP_DOCKER_E2E=1")
	}
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Docker e2e is supported on linux and darwin")
	}
	// Find tests/docker/scripts/run-e2e.sh relative to this test file
	_, file, _, _ := runtime.Caller(0)
	testDir := filepath.Dir(file)
	scriptPath := filepath.Join(testDir, "docker", "scripts", "run-e2e.sh")
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skip("tests/docker/scripts/run-e2e.sh not found")
	}
	// Run from tests/docker so compose and context paths are correct
	composeDir := filepath.Join(testDir, "docker")
	cmd := exec.Command("sh", "./scripts/run-e2e.sh")
	cmd.Dir = composeDir
	cmd.Env = append(os.Environ(), "PATH="+os.Getenv("PATH"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("run-e2e.sh output:\n%s", out)
		t.Fatalf("run-e2e.sh: %v", err)
	}
	t.Logf("Docker e2e output:\n%s", out)
}
