package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type LibraryMap struct {
	library string
	path    string
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <directory>", os.Args[0])
	}

	directory := os.Args[1]
	err := filepath.Walk(directory, processFile)
	if err != nil {
		log.Fatalf("Failed to read directory: %s", err)
	}
}

func processFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		log.Printf("Failed to access file: %s", err)
		return err
	}

	if !info.IsDir() {
		if isBinaryOrLibrary(path) {
			linkedLibraries, err := getLinkedLibraries(path)
			if err != nil {
				return nil
			}
			if len(linkedLibraries) > 0 {
				fmt.Printf("\n\nFile: %s\n", path)
				for _, lib := range linkedLibraries {
					fmt.Printf("  Linked Library: %s => %s\n", lib.library, lib.path)
				}
			}
		}
	}

	return nil
}

func isBinaryOrLibrary(filePath string) bool {
	cmd := exec.Command("file", filePath)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to determine file type for %s: %s", filePath, err)
		return false
	}

	fileType := string(output)
	return strings.Contains(fileType, "ELF") || strings.Contains(fileType, "executable")
}

func getLinkedLibraries(filePath string) ([]LibraryMap, error) {
	out, err := exec.Command("ldd", filePath).Output()
	if err != nil {
		return nil, err
	}

	var libraries []LibraryMap
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 3 && parts[1] == "=>" && parts[2] == "not" {
			packages, err := lookupLibrary(parts[0])
			if err != nil {
				return nil, err
			}
			fmt.Printf("Library %s is missing from your project.\nYou can add it by installing one of the following packages: %s", parts[0], packages)
			libraries = append(libraries, LibraryMap{parts[0], parts[2]})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return libraries, nil
}

func lookupLibrary(filename string) (string, error) {
	cmd := exec.Command("nix-locate", "--top-level", "--minimal", filename)
	results, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error when locating package: %w", err)
	}
	if results == nil {
		return "", fmt.Errorf("no results found")
	}
	return string(results), nil
}

func patchBinary(filePath string, libraryPath string) error {
	cmd := exec.Command("patchelf", "--set-rpath", libraryPath, filePath)
	_, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to set rpath: %w", err)
	}
	fmt.Printf("Patched binary %s to use library path %s\n", filePath, libraryPath)
	return nil
}
