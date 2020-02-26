package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Xiol/tvhtc2/internal/pkg/config"
	"github.com/Xiol/tvhtc2/internal/pkg/renamer"
)

func main() {
	path := flag.String("path", "", "Path to file to rename")
	dry := flag.Bool("dry-run", false, "Don't move anything, just print what would happen")
	flag.Parse()

	if err := config.InitConfig(); err != nil {
		fmt.Printf("error: failed to load config: %s", err)
		os.Exit(1)
	}

	if *path == "" {
		fmt.Printf("error: no path provided\n")
		os.Exit(1)
	}

	var err error
	if *path, err = filepath.Abs(*path); err != nil {
		fmt.Printf("error: could not find absolute path to file: %s\n", err)
		os.Exit(1)
	}

	r := renamer.NewRenamer()
	newPath := r.Rename(*path)

	if *path == newPath {
		os.Exit(0)
	}

	dir, _ := filepath.Split(newPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if *dry {
			fmt.Printf("create directory: %s\n", dir)
		} else {
			fmt.Printf("creating destination directory: %s\n", dir)
			if err := os.Mkdir(dir, 0755); err != nil {
				fmt.Printf("error: unable to create destination directory: %s\n", err)
				os.Exit(1)
			}
		}
	}

	if *dry {
		fmt.Printf("%s -> %s\n", *path, newPath)
		os.Exit(0)
	}

	if err := os.Rename(*path, newPath); err != nil {
		fmt.Printf("error: failed to move file: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("%s -> %s\n", *path, newPath)

	dir, _ = filepath.Split(*path)
	if dirEmpty(dir) {
		if err := os.Remove(dir); err != nil {
			fmt.Printf("error: failed to remove empty directory: %s", err)
		}
	}
}

func dirEmpty(dir string) bool {
	f, err := os.Open(dir)
	if err != nil {
		fmt.Printf("error: could not open directory %s for read: %s", dir, err)
		return false
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true
	}
	return false
}
