package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("repoRoot: %v", err)
	}
	return root
}

func buildCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "sft-test-bin")
	cmd := exec.Command("go", "build", "-o", bin, filepath.Join(repoRoot(t), "cmd", "sft"))
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("buildCLI: %v\n%s", err, out)
	}
	return bin
}

func runCLI(t *testing.T, binary, workdir string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestHelpFlags(t *testing.T) {
	t.Parallel()

	bin := buildCLI(t)
	for _, args := range [][]string{{"help"}, {"-h"}, {"--help"}} {
		args := args
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			t.Parallel()

			out, err := runCLI(t, bin, repoRoot(t), args...)
			if err != nil {
				t.Fatalf("runCLI(%v): %v\n%s", args, err, out)
			}
			if strings.Contains(out, `unknown command`) {
				t.Fatalf("help output should not contain unknown command:\n%s", out)
			}
			if !strings.Contains(out, "Workflow:") {
				t.Fatalf("help output missing usage text:\n%s", out)
			}
		})
	}
}

func TestExportCommand(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	bin := buildCLI(t)
	specPath := filepath.Join(tmp, "spec.yaml")
	exportPath := filepath.Join(tmp, "exported.yaml")
	spec := `app:
  name: demo
  description: Demo spec
  screens:
    - name: Home
      description: Welcome
`

	if err := os.WriteFile(specPath, []byte(spec), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}

	if out, err := runCLI(t, bin, tmp, "init", specPath); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}

	out, err := runCLI(t, bin, tmp, "export", exportPath)
	if err != nil {
		t.Fatalf("export failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "exported") {
		t.Fatalf("export output missing confirmation:\n%s", out)
	}

	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "name: demo") || !strings.Contains(text, "name: Home") {
		t.Fatalf("unexpected export contents:\n%s", text)
	}
}
