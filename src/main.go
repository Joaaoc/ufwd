package main

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"
)

func confirmAction(actionName string, force bool) {
	if force {
		return
	}
	fmt.Printf("Warning! You are about to execute '%s'. Proceed (y/N)? ", actionName)
	reader := bufio.NewReader(os.Stdin)
	resp, _ := reader.ReadString('\n')
	resp = strings.ToLower(strings.TrimSpace(resp))
	if resp != "y" {
		fmt.Println("Operation aborted.")
		os.Exit(0)
	}
}

func printHelp() {
	fmt.Println("UFWD - Uncomplicated Firewall for Docker")
	fmt.Println("Usage: ufwd [command] [options]")
	fmt.Println("\nAvailable commands:")
	fmt.Println("  install          Install configurations and clean up residues")
	fmt.Println("  uninstall        Remove hooks and delete ufwd configurations")
	fmt.Println("  reset            Erase all saved ufwd rules")
	fmt.Println("  enable           Enable ufwd protection")
	fmt.Println("  disable          Disable ufwd and fallback to default Docker behavior")
	fmt.Println("  allow            Allow port in firewall (Ex: 3000/tcp, 3001:3005/tcp, 80,443/tcp)")
	fmt.Println("  insert NUM       Insert rule at a specific position (Ex: ufwd insert 1 allow 3000/tcp)")
	fmt.Println("  delete           Remove a firewall rule (by rule definition or NUM)")
	fmt.Println("  status           List active rules (use 'status numbered' for numbered list)")
	fmt.Println("  reload           Reload rules silently")
	fmt.Println("\nOptions:")
	fmt.Println("  -f, --force      Skip confirmation prompts (y/N) for destructive actions")
}

func checkRoot() {
	if runtime.GOOS == "linux" {
		u, err := user.Current()
		if err == nil && u.Uid != "0" {
			fmt.Println("Error: This tool must be run as root (use sudo).")
			os.Exit(1)
		}
	}
}

var Version = "dev"

func main() {
	checkRoot()

	args := os.Args[1:]
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		printHelp()
		os.Exit(0)
	}

	if args[0] == "-v" || args[0] == "--version" || args[0] == "version" {
		fmt.Printf("ufwd version %s\n", Version)
		os.Exit(0)
	}

	if len(args) >= 2 && args[0] == "delete" && args[1] == "allow" {
		args = append(args[:1], args[2:]...)
	}

	action := args[0]
	force := false
	var cmdArgs []string

	for _, arg := range args[1:] {
		if arg == "--force" || arg == "-f" {
			force = true
		} else {
			cmdArgs = append(cmdArgs, arg)
		}
	}

	config := loadConfig()

	switch action {
	case "install":
		confirmAction("install", force)
		installHook()
		os.Exit(0)

	case "uninstall":
		confirmAction("uninstall", force)
		uninstallHook()
		os.Exit(0)

	case "reset":
		confirmAction("reset", force)
		config.ExternalPorts = []Rule{}
		saveConfig(config)
		fmt.Println("Success: All ufwd rules have been erased.")
		reloadRules(config)

	case "disable":
		confirmAction("disable", force)
		config.Enabled = false
		saveConfig(config)
		fmt.Println("Warning: ufwd disabled. Docker is now operating openly.")
		reloadRules(config)

	case "enable":
		confirmAction("enable", force)
		config.Enabled = true
		saveConfig(config)
		fmt.Println("Success: ufwd enabled. Protection is active.")
		reloadRules(config)

	case "allow", "delete", "insert":
		var rule Rule
		var err error
		insertIndex := -1
		deleteIndex := -1

		if action == "delete" && len(cmdArgs) == 1 {
			if num, parseErr := strconv.Atoi(cmdArgs[0]); parseErr == nil {
				deleteIndex = num - 1
			}
		}

		if action == "insert" {
			if len(cmdArgs) < 2 {
				fmt.Println("Syntax error: insert requires a line number and a rule")
				os.Exit(1)
			}
			num, parseErr := strconv.Atoi(cmdArgs[0])
			if parseErr != nil {
				fmt.Println("Syntax error: insert requires a valid line number")
				os.Exit(1)
			}
			insertIndex = num - 1

			ruleArgs := cmdArgs[1:]
			if len(ruleArgs) > 0 && strings.ToLower(ruleArgs[0]) == "allow" {
				ruleArgs = ruleArgs[1:]
			}
			rule, err = parseRule(ruleArgs)
		} else if deleteIndex == -1 {
			rule, err = parseRule(cmdArgs)
		}

		if err != nil && deleteIndex == -1 {
			fmt.Printf("Syntax error: %v\n", err)
			os.Exit(1)
		}

		if action == "allow" || action == "insert" {
			foundIndex := -1
			for i, r := range config.ExternalPorts {
				if r.IP == rule.IP && r.Port == rule.Port && r.Proto == rule.Proto {
					foundIndex = i
					break
				}
			}

			if foundIndex == -1 {
				if action == "insert" {
					if insertIndex < 0 {
						insertIndex = 0
					}
					if insertIndex > len(config.ExternalPorts) {
						insertIndex = len(config.ExternalPorts)
					}
					newPorts := make([]Rule, 0, len(config.ExternalPorts)+1)
					newPorts = append(newPorts, config.ExternalPorts[:insertIndex]...)
					newPorts = append(newPorts, rule)
					newPorts = append(newPorts, config.ExternalPorts[insertIndex:]...)
					config.ExternalPorts = newPorts
				} else {
					config.ExternalPorts = append(config.ExternalPorts, rule)
				}

				saveConfig(config)
				fmt.Printf("Rule added (IP: %s, Port(s): %s, Proto: %s).\n", rule.IP, rule.Port, rule.Proto)
				reloadRules(config)
				if !config.Enabled {
					fmt.Println("Note: ufwd is currently DISABLED. Enable it with 'ufwd enable' for the rule to take effect.")
				}
			} else {
				fmt.Println("Skipping adding existing rule.")
			}

		} else if action == "delete" {
			if deleteIndex != -1 {
				if deleteIndex >= 0 && deleteIndex < len(config.ExternalPorts) {
					delRule := config.ExternalPorts[deleteIndex]
					config.ExternalPorts = append(config.ExternalPorts[:deleteIndex], config.ExternalPorts[deleteIndex+1:]...)
					saveConfig(config)
					fmt.Printf("Rule deleted (IP: %s, Port(s): %s, Proto: %s).\n", delRule.IP, delRule.Port, delRule.Proto)
					reloadRules(config)
				} else {
					fmt.Println("Error: Invalid rule number.")
					os.Exit(1)
				}
			} else {
				foundIndex := -1
				for i, r := range config.ExternalPorts {
					if r.IP == rule.IP && r.Port == rule.Port && r.Proto == rule.Proto {
						foundIndex = i
						break
					}
				}
				if foundIndex != -1 {
					config.ExternalPorts = append(config.ExternalPorts[:foundIndex], config.ExternalPorts[foundIndex+1:]...)
					saveConfig(config)
					fmt.Printf("Rule deleted (IP: %s, Port(s): %s, Proto: %s).\n", rule.IP, rule.Port, rule.Proto)
					reloadRules(config)
				} else {
					fmt.Println("Could not find a rule that matches.")
				}
			}
		}

	case "status":
		numbered := false
		for _, arg := range cmdArgs {
			if arg == "numbered" || arg == "--numbered" || arg == "-n" {
				numbered = true
				break
			}
		}

		if !config.Enabled {
			fmt.Println("Status: inactive (UFWD disabled)\n")
			os.Exit(0)
		}
		fmt.Println("Status: active (DOCKER-USER)\n")
		if len(config.ExternalPorts) == 0 {
			fmt.Println("No external ports allowed yet.")
		} else {
			if numbered {
				fmt.Printf("%-5s %-27s %-11s %s\n", "[Num]", "To", "Action", "From")
				fmt.Printf("%-5s %-27s %-11s %s\n", "-----", "--", "------", "----")
			} else {
				fmt.Printf("%-27s %-11s %s\n", "To", "Action", "From")
				fmt.Printf("%-27s %-11s %s\n", "--", "------", "----")
			}

			for i, r := range config.ExternalPorts {
				portProto := fmt.Sprintf("%s/%s", r.Port, r.Proto)
				fromStr := "Anywhere"
				if r.IP != "any" {
					fromStr = r.IP
				}
				if numbered {
					fmt.Printf("[%3d] %-27s %-11s %s\n", i+1, portProto, "ALLOW", fromStr)
				} else {
					fmt.Printf("%-27s %-11s %s\n", portProto, "ALLOW", fromStr)
				}
			}
		}
		fmt.Println("\n(To view raw kernel rules: sudo iptables -L DOCKER-USER -n -v)")

	case "reload":
		reloadRules(config)

	default:
		fmt.Printf("Unknown command: %s\n", action)
		printHelp()
		os.Exit(1)
	}
}
