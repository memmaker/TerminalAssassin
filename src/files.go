package main

import (
	"bufio"
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

type Files struct {
	fs embed.FS
}

func (f Files) GetFilesInPath(path string) []string {
	dir, err := f.readDir(path)
	//dir, err := os.readDir(path)
	if err != nil {
		return []string{}
	}
	filesInDir := make([]string, 0)
	for _, entry := range dir {
		if !entry.IsDir() {
			fullpath := filepath.Join(path, entry.Name())
			filesInDir = append(filesInDir, fullpath)
		}
	}
	return filesInDir
}

func (f Files) GetSubdirectories(path string) []string {
	dir, err := f.readDir(path)
	//dir, err := os.readDir(path)
	if err != nil {
		return []string{}
	}
	subdirectories := make([]string, 0)
	for _, entry := range dir {
		if entry.IsDir() {
			fullpath := filepath.Join(path, entry.Name())
			subdirectories = append(subdirectories, fullpath)
		}
	}
	return subdirectories
}
func (f Files) ReadDir(path string) ([]fs.DirEntry, error) {
	return f.readDir(path)
}
func (f Files) FileExists(filename string) bool {
	_, err := os.Open(filename)
	if os.IsNotExist(err) {
		_, secErr := f.fs.Open(filename)
		if secErr != nil {
			return false
		}
	}
	return true
}
func (f Files) Open(filename string) (fs.File, error) {
	file, err := os.Open(filename)
	if err != nil {
		secFile, secErr := f.fs.Open(filename)
		if secErr != nil {
			return nil, err
		}
		return secFile, nil
	}
	return file, nil
}

func (f Files) readDir(dir string) ([]fs.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	secEntries, secErr := f.fs.ReadDir(dir)
	if err != nil {
		return secEntries, secErr
	}
	if secErr != nil {
		return entries, err
	}
	return append(entries, secEntries...), nil
}

func (f Files) LoadTextFile(filename string) []string {
	file, _ := f.Open(filename)
	defer file.Close()
	scanner := bufio.NewScanner(file)
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}
