package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// expandPaths expands every argument (file, directory or glob) into a sorted,
// deduplicated list of concrete files. Directories are walked recursively and
// only *.log / *.jsonl files are collected, filtered by the exclude pattern.
func expandPaths(args []string, exclude string) ([]string, error) {
	var out []string
	for _, arg := range args {
		expanded, err := expandPath(arg, exclude)
		if err != nil {
			return nil, err
		}
		out = append(out, expanded...)
	}
	return uniqueSorted(out), nil
}

// expandPath resolves a single argument into concrete files.
func expandPath(arg, exclude string) ([]string, error) {
	// Glob pattern.
	if strings.ContainsAny(arg, "*?[") {
		matches, err := filepath.Glob(arg)
		if err != nil {
			return nil, fmt.Errorf("invalid glob %q: %w", arg, err)
		}
		var out []string
		for _, m := range matches {
			info, err := os.Stat(m)
			if err != nil {
				continue
			}
			if info.IsDir() {
				sub, err := walkDir(m, exclude)
				if err != nil {
					return nil, err
				}
				out = append(out, sub...)
			} else if !excluded(m, exclude) {
				out = append(out, m)
			}
		}
		return out, nil
	}

	// Directory: recursive walk.
	if info, err := os.Stat(arg); err == nil && info.IsDir() {
		return walkDir(arg, exclude)
	}

	// Plain file.
	if info, err := os.Stat(arg); err == nil && !info.IsDir() {
		if excluded(arg, exclude) {
			return nil, nil
		}
		return []string{arg}, nil
	}
	return nil, fmt.Errorf("no such file or directory: %s", arg)
}

// walkDir collects *.log and *.jsonl files under dir, honoring exclude.
func walkDir(dir, exclude string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isLogFile(path) {
			return nil
		}
		if excluded(path, exclude) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

// isLogFile reports whether the file has a log-like extension.
func isLogFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".log" || ext == ".jsonl"
}

// excluded reports whether the file matches the exclude glob (by base name).
func excluded(path, pattern string) bool {
	if pattern == "" {
		return false
	}
	ok, err := filepath.Match(pattern, filepath.Base(path))
	return err == nil && ok
}

// uniqueSorted deduplicates and sorts a list of paths for stable output.
func uniqueSorted(paths []string) []string {
	set := make(map[string]bool, len(paths))
	for _, p := range paths {
		set[p] = true
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
