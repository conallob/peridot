//go:build darwin

package darwin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const launchLabel = "com.peridot.peridotd"

type darwinDaemon struct{}

func plistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", launchLabel+".plist")
}

func (d *darwinDaemon) Install() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>%s</string>
  <key>ProgramArguments</key>
  <array><string>%s</string></array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardErrorPath</key><string>%s/.peridot/peridotd.err.log</string>
  <key>StandardOutPath</key><string>%s/.peridot/peridotd.out.log</string>
</dict>
</plist>
`, launchLabel, exe, home, home)
	p := plistPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(p, []byte(plist), 0o644); err != nil {
		return err
	}
	return d.load()
}

func (d *darwinDaemon) Uninstall() error {
	_ = d.unload()
	return os.Remove(plistPath())
}

func (d *darwinDaemon) load() error   { return run("launchctl", "load", "-w", plistPath()) }
func (d *darwinDaemon) unload() error { return run("launchctl", "unload", "-w", plistPath()) }

func (d *darwinDaemon) Start() error { return run("launchctl", "start", launchLabel) }
func (d *darwinDaemon) Stop() error  { return run("launchctl", "stop", launchLabel) }

func (d *darwinDaemon) Restart() error {
	_ = d.Stop()
	return d.Start()
}

func (d *darwinDaemon) Status() (string, error) {
	out, err := exec.Command("launchctl", "list").Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, launchLabel) {
			return "running: " + strings.TrimSpace(line), nil
		}
	}
	return "not running", nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %v: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}
