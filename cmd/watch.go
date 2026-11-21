package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

func runWithWatch(rootDir string, childArgs []string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "tmp" || name == "node_modules" {
				return filepath.SkipDir
			}
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	binPath := filepath.Join(os.TempDir(), fmt.Sprintf("backend-go-watch-%d", os.Getpid()))

	var (
		mu       sync.Mutex
		proc     *exec.Cmd
		debTimer *time.Timer
	)

	startChild := func() {
		mu.Lock()
		defer mu.Unlock()

		if proc != nil && proc.Process != nil {
			_ = proc.Process.Kill()
			_ = proc.Wait()
		}

		build := exec.Command("go", "build", "-o", binPath, ".")
		build.Dir = rootDir
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			fmt.Printf("[watch] build failed: %v\n", err)
			return
		}

		cmd := exec.Command(binPath, childArgs...)
		cmd.Dir = rootDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Start(); err != nil {
			fmt.Printf("[watch] failed to start process: %v\n", err)
			return
		}
		proc = cmd
		fmt.Printf("[watch] started: %s %s\n", filepath.Base(binPath), strings.Join(childArgs, " "))
	}

	startChild()

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&fsnotify.Create != 0 {
				if info, statErr := os.Stat(event.Name); statErr == nil && info.IsDir() {
					_ = watcher.Add(event.Name)
				}
			}
			if !(strings.HasSuffix(event.Name, ".go") ||
				strings.HasSuffix(event.Name, ".sql") ||
				strings.HasSuffix(event.Name, ".env")) {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}

			name := event.Name
			mu.Lock()
			if debTimer != nil {
				debTimer.Stop()
			}
			debTimer = time.AfterFunc(200*time.Millisecond, func() {
				fmt.Printf("[watch] change detected (%s), restarting...\n", name)
				startChild()

				defer func() {
					mu.Lock()
					if proc != nil && proc.Process != nil {
						_ = proc.Process.Kill()
						_ = proc.Wait()
					}
					mu.Unlock()
					_ = os.Remove(binPath)
				}()
			})
			mu.Unlock()

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Printf("[watch] error: %v\n", err)
		}
	}
}
