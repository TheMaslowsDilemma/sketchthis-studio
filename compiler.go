package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const compilerBin = "sketchlang"

func randomB64String(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:n], nil
}

func parseSketchError(errstr string) (*SketchError, error) {
	var skerr SketchError
	if err := json.Unmarshal([]byte(errstr), &skerr); err != nil {
		return &SketchError{ Message: errstr }, nil
	}
	return &skerr, nil
}

func ValidateSketch(code string, log *Logger) (*SketchError, error) {
	tmpDir, err := os.MkdirTemp("", "sketchstudio-validation-")
	if err != nil {
		return nil, fmt.Errorf("MkdirTemp: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	filename, err := randomB64String(7)
	if err != nil {
		return nil, fmt.Errorf("random filename: %w", err)
	}
	filename += ".sketch"

	if err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("write code: %w", err)
	}

	args := []string{filename, "-pos", "0,0", "-size", "100,100"}
	log.Debug("validate: %s %v\n", compilerBin, args)

	cmd := exec.Command(compilerBin, args...)
	cmd.Dir = tmpDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return parseSketchError(stderr.String())
	}

	log.Debug("validate: ok\n")
	return nil, nil
}

func Compile(code, name string, pos, size Vec2, log *Logger) (svg, gcode string, err error) {
	tmpDir, err := os.MkdirTemp("", "sketch-")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, name+".sketch"), []byte(code), 0644); err != nil {
		return "", "", err
	}

	args := []string{
		name + ".sketch",
		"-o", name,
		"-pos", fmt.Sprintf("%g,%g", pos.X, pos.Y),
		"-size", fmt.Sprintf("%g,%g", size.X, size.Y),
		"--svg", "--gcode",
	}
	log.Debug("compile: %s %v\n", compilerBin, args)

	cmd := exec.Command(compilerBin, args...)
	cmd.Dir = tmpDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("%s", stderr.String())
	}

	svgData, err := os.ReadFile(filepath.Join(tmpDir, name+".svg"))
	if err != nil {
		return "", "", fmt.Errorf("SVG not generated")
	}

	gcodeData, err := os.ReadFile(filepath.Join(tmpDir, name+".txt"))
	if err != nil {
		return "", "", fmt.Errorf("gcode not generated")
	}

	return string(svgData), string(gcodeData), nil
}