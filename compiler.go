package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const compilerBin = "sketchlang" // assumes in PATH

func Compile(code, outputName string, pos, size Vec2, log *Logger) (string, error) {
	tmpDir, err := os.MkdirTemp("", "sketch-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, outputName+".sketch")
	if err := os.WriteFile(inputPath, []byte(code), 0644); err != nil {
		return "", err
	}

	args := []string{
		outputName + ".sketch",
		"-o", outputName,
		"-pos", fmt.Sprintf("%g,%g", pos.X, pos.Y),
		"-size", fmt.Sprintf("%g,%g", size.X, size.Y),
		"--svg",
	}

	log.Debug("running: %s %v", compilerBin, args)

	cmd := exec.Command(compilerBin, args...)
	cmd.Dir = tmpDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("compile error: %s", stderr.String())
	}

	svgPath := filepath.Join(tmpDir, outputName+".svg")
	svg, err := os.ReadFile(svgPath)
	if err != nil {
		return "", fmt.Errorf("SVG not generated")
	}

	return string(svg), nil
}

func Validate(code string, log *Logger) (bool, []string) {
	tmpDir, err := os.MkdirTemp("", "sketch-validate-")
	if err != nil {
		return false, []string{err.Error()}
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "_validate.sketch")
	if err := os.WriteFile(inputPath, []byte(code), 0644); err != nil {
		return false, []string{err.Error()}
	}

	cmd := exec.Command(compilerBin, "_validate.sketch", "-o", "_validate", "--svg")
	cmd.Dir = tmpDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return false, []string{stderr.String()}
	}

	return true, nil
}