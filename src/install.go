package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const canonicalAfterInit = `#!/bin/sh
#
# after.init: if executable, called by ufw-init. See 'man ufw-framework' for
#             details. Note that output from these scripts is not seen via the
#             the ufw command, but instead via ufw-init.
#
# Copyright 2013 Canonical Ltd.
#
#    This program is free software: you can redistribute it and/or modify
#    it under the terms of the GNU General Public License version 3,
#    as published by the Free Software Foundation.
#
#    This program is distributed in the hope that it will be useful,
#    but WITHOUT ANY WARRANTY; without even the implied warranty of
#    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
#    GNU General Public License for more details.
#
#    You should have received a copy of the GNU General Public License
#    along with this program.  If not, see <http://www.gnu.org/licenses/>.
#
set -e

case "$1" in
start)
    # typically required
    ;;
stop)
    # typically required
    ;;
status)
    # optional
    ;;
flush-all)
    # optional
    ;;
*)
    echo "'$1' not supported"
    echo "Usage: after.init {start|stop|flush-all|status}"
    ;;
esac
`

func injectHookIntoAfterInit() {
	afterInitPath := "/etc/ufw/after.init"

	var content string
	if _, err := os.Stat(afterInitPath); os.IsNotExist(err) {
		content = canonicalAfterInit
	} else {
		data, err := os.ReadFile(afterInitPath)
		if err != nil {
			fmt.Printf("Warning: Could not read %s: %v\n", afterInitPath, err)
			return
		}
		content = string(data)
	}

	if strings.Contains(content, "# BEGIN UFWD START") {
		fmt.Println("✓ UFWD hooks are already present in after.init.")
	} else {
		lines := strings.Split(content, "\n")
		var newLines []string
		for _, line := range lines {
			newLines = append(newLines, line)
			trimmed := strings.TrimSpace(line)
			if trimmed == "start)" {
				newLines = append(newLines, "    # BEGIN UFWD START")
				newLines = append(newLines, "    if [ -x /usr/local/bin/ufwd ]; then")
				newLines = append(newLines, "        /usr/local/bin/ufwd reload > /dev/null 2>&1")
				newLines = append(newLines, "    fi")
				newLines = append(newLines, "    # END UFWD START")
			} else if trimmed == "stop)" || trimmed == "flush-all)" {
				newLines = append(newLines, "    # BEGIN UFWD STOP")
				newLines = append(newLines, "    iptables -F DOCKER-USER 2>/dev/null || true")
				newLines = append(newLines, "    iptables -A DOCKER-USER -j RETURN 2>/dev/null || true")
				newLines = append(newLines, "    # END UFWD STOP")
			}
		}

		err := os.WriteFile(afterInitPath, []byte(strings.Join(newLines, "\n")), 0755)
		if err != nil {
			if os.IsPermission(err) {
				fmt.Println("Permission error. Did you run with sudo?")
				return
			}
			fmt.Printf("Warning: Failed to update %s: %v\n", afterInitPath, err)
		} else {
			fmt.Println("✓ Hook injected perfectly into /etc/ufw/after.init.")
			os.Chmod(afterInitPath, 0755)
		}
	}
}

func installHook() {
	fmt.Println("Starting UFWD installation and configuration...")

	os.MkdirAll(filepath.Dir(configFile), 0755)
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		saveConfig(Config{Enabled: true, ExternalPorts: []Rule{}})
		fmt.Println("✓ UFWD state file created.")
	}

	injectHookIntoAfterInit()

	afterRulesPath := "/etc/ufw/after.rules"
	if _, err := os.Stat(afterRulesPath); err == nil {
		content, err := os.ReadFile(afterRulesPath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			var newLines []string
			inDockerBlock := false
			cleaned := false

			for _, line := range lines {
				if strings.Contains(line, "# BEGIN UFW AND DOCKER") {
					inDockerBlock = true
					cleaned = true
					continue
				}
				if strings.Contains(line, "# END UFW AND DOCKER") {
					inDockerBlock = false
					continue
				}
				if !inDockerBlock {
					newLines = append(newLines, line)
				}
			}

			if cleaned {
				os.WriteFile(afterRulesPath+".ufwd.bak", content, 0644)
				os.WriteFile(afterRulesPath, []byte(strings.Join(newLines, "\n")), 0644)
				fmt.Println("✓ Old 'ufw-docker' clutter cleaned from firewall (Backup saved as .bak).")
				fmt.Println("✓ UFW service will be restarted shortly.")
				runCmd("systemctl restart ufw || ufw reload")
			} else {
				fmt.Println("✓ No legacy 'ufw-docker' configuration found.")
			}
		} else {
			fmt.Printf("Error while cleaning legacy rules: %v\n", err)
		}
	}

	fmt.Println("\nInstallation complete! You can now use the tool.")
}

func uninstallHook() {
	fmt.Println("Starting full UFWD uninstallation...")

	afterInitPath := "/etc/ufw/after.init"
	if _, err := os.Stat(afterInitPath); err == nil {
		data, err := os.ReadFile(afterInitPath)
		if err == nil {
			lines := strings.Split(string(data), "\n")
			var newLines []string
			skip := false

			for _, line := range lines {
				if strings.Contains(line, "# BEGIN UFWD START") || strings.Contains(line, "# BEGIN UFWD STOP") {
					skip = true
					continue
				}
				if skip && (strings.Contains(line, "# END UFWD START") || strings.Contains(line, "# END UFWD STOP")) {
					skip = false
					continue
				}
				if !skip {
					newLines = append(newLines, line)
				}
			}

			os.WriteFile(afterInitPath, []byte(strings.Join(newLines, "\n")), 0755)
			fmt.Printf("✓ Hooks surgically removed from %s\n", afterInitPath)
		}
	}

	if err := os.Remove(configFile); err == nil {
		fmt.Printf("✓ State file deleted (%s)\n", configFile)
	}

	runCmd("iptables -F DOCKER-USER 2>/dev/null")
	runCmd("iptables -A DOCKER-USER -j RETURN 2>/dev/null")
	fmt.Println("✓ iptables flushed and restored to default Docker behavior.")
	fmt.Println("\nUninstallation complete.")
	fmt.Println("Tip: To remove the executable, run: sudo rm /usr/local/bin/ufwd")
}
