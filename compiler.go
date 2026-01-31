package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"encoding/json"
)

const compilerBin = "sketchlang"

func parseCompilerError(errstr string) (*SketchError, error) {
	var skerr SketchError
	if err := json.Unmarshal([]byte(errstr), &skerr); err != nil {
		return nil, fmt.Errorf("failed to parse compiler err: %v")
	}
	return &skerr
}

func ValidateSketch(text string, log *Logger) (*SketchError, error) {
	tmpDir, err := os.MkdirTemp("", "sketchstudio-validation-")
	if err != nil {
		return nil, fmt.Errorf("failed to MkdirTemp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filename, err := randomB64String(7)
	if err != nil {
		return nil, fmt.Errorf("failed to get random filename: %v", err)
	}

	filename = filename + ".sketch"
	tmpFilePath := filepath.Join(tmpDir, filename)

	if err := os.WriteFile(tmpFilePath, []byte(text), 0644); err != nil {
		return nil, fmt.Errorf("failed to write code: %v", err)
	}

	// pos and size dont matter, so vals are arbitrary
	args := []string{
		filename,
		"-pos 0,0", 
		"-size 100,100",
	}

	log.Debug("validate begin. %s %v", compilerBin, args)
	
	cmd := exec.Command(compilerBin, args...)
	cmd.Dir = tmpDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		return parseSketchError(stderr.String())
	}

	// success
	log.Debug("validate success")
	return nil, nil
}

func Compile(code, outputName string, pos, size Vec2, log *Logger) (svg, gcode string, err error) {
	tmpDir, err := os.MkdirTemp("", "sketch-")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, outputName + ".sketch")
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

	log.Debug("%s %v", compilerBin, args)

	cmd := exec.Command(compilerBin, args...)
	cmd.Dir = tmpDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("%s", stderr.String())
	}

	svgData, err := os.ReadFile(filepath.Join(tmpDir, outputName + ".svg"))
	if err != nil {
		return "", "", fmt.Errorf("SVG not generated")
	}

	gcodeData, err := os.ReadFile(filepath.Join(tmpDir, outputName + ".txt"))
	if err != nil {
		return "", "", fmt.Errorf("gcode not generated")
	}

	return string(svgData), string(gcodeData), nil
}