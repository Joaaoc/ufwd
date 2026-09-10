package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func runCmd(command string) {
	cmd := exec.Command("sh", "-c", command)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Firewall error while executing: %s\n", command)
		fmt.Println(strings.TrimSpace(stderr.String()))
	}
}

func reloadRules(config Config) {
	var rulesBatch []string
	rulesBatch = append(rulesBatch, "*filter", ":DOCKER-USER - [0:0]", "-F DOCKER-USER")

	if !config.Enabled {
		rulesBatch = append(rulesBatch, "-A DOCKER-USER -j RETURN")
		rulesBatch = append(rulesBatch, "COMMIT\n")
	} else {
		rulesBatch = append(rulesBatch, "-A DOCKER-USER -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT")

		for _, rule := range config.ExternalPorts {
			ipFlag := ""
			if rule.IP != "any" {
				ipFlag = fmt.Sprintf("-s %s ", rule.IP)
			}

			parts := strings.Split(rule.Port, ",")
			for _, pPart := range parts {
				pPart = strings.TrimSpace(pPart)
				if pPart == "" {
					continue
				}

				if strings.Contains(pPart, ":") {
					rangeParts := strings.SplitN(pPart, ":", 2)
					startP, err1 := strconv.Atoi(rangeParts[0])
					endP, err2 := strconv.Atoi(rangeParts[1])

					if err1 == nil && err2 == nil && endP-startP <= 2000 {
						for p := startP; p <= endP; p++ {
							rulesBatch = append(rulesBatch, fmt.Sprintf("-A DOCKER-USER %s-p %s -m conntrack --ctorigdstport %d -j ACCEPT", ipFlag, rule.Proto, p))
						}
					} else {
						rulesBatch = append(rulesBatch, fmt.Sprintf("-A DOCKER-USER %s-p %s -m conntrack --ctorigdstport %s -j ACCEPT", ipFlag, rule.Proto, pPart))
					}
				} else {
					rulesBatch = append(rulesBatch, fmt.Sprintf("-A DOCKER-USER %s-p %s -m conntrack --ctorigdstport %s -j ACCEPT", ipFlag, rule.Proto, pPart))
				}
			}
		}

		rulesBatch = append(rulesBatch, "-A DOCKER-USER -i docker0 -j RETURN")
		rulesBatch = append(rulesBatch, "-A DOCKER-USER -i br-+ -j RETURN")
		rulesBatch = append(rulesBatch, "-A DOCKER-USER -j DROP")
		rulesBatch = append(rulesBatch, "COMMIT\n")
	}

	batchData := strings.Join(rulesBatch, "\n")

	cmd := exec.Command("iptables-restore", "--noflush")
	cmd.Stdin = strings.NewReader(batchData)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("Critical error while applying atomic rules (iptables-restore):")
		fmt.Println(stderr.String())
		os.Exit(1)
	}
}
