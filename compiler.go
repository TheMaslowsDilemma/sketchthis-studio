package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const compilerBin = "sketchlang"

func Compile(code, outputName string, pos, size Vec2, log *Logger) (svg, gcode string, err error) {
	tmpDir, err := os.MkdirTemp("", "sketch-")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, outputName+".sketch")
	if err := os.WriteFile(inputPath, []byte(code), 0644); err != nil {
		return "", "", err
	}

	args := []string{
		outputName + ".sketch",
		"-o", outputName,
		"-pos", fmt.Sprintf("%g,%g", pos.X, pos.Y),
		"-size", fmt.Sprintf("%g,%g", size.X, size.Y),
		"--svg",
		"--gcode",
	}

	log.Debug("running: %s %v", compilerBin, args)

	cmd := exec.Command(compilerBin, args...)
	cmd.Dir = tmpDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("compile error: %s", stderr.String())
	}

	svgData, err := os.ReadFile(filepath.Join(tmpDir, outputName+".svg"))
	if err != nil {
		return "", "", fmt.Errorf("SVG not generated")
	}

	gcodeData, err := os.ReadFile(filepath.Join(tmpDir, outputName+".txt"))
	if err != nil {
		return "", "", fmt.Errorf("gcode not generated")
	}

	return string(svgData), string(gcodeData), nil
}