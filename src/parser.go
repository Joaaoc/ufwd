package main

import (
	"fmt"
	"strings"
)

func parseRule(args []string) (Rule, error) {
	ruleStr := strings.TrimSpace(strings.Join(args, " "))

	if ruleStr == "" {
		return Rule{}, fmt.Errorf("empty rule")
	}

	tokens := strings.Fields(ruleStr)

	// Single format: "3000/tcp" or "3000"
	if len(tokens) == 1 {
		parts := strings.SplitN(ruleStr, "/", 2)
		port := parts[0]
		proto := "tcp"
		if len(parts) == 2 {
			proto = parts[1]
		}
		return Rule{IP: "any", Port: port, Proto: strings.ToLower(proto)}, nil
	}

	// Complex format: "proto tcp from IP port 3000"
	ip := "any"
	proto := "tcp"
	port := ""

	i := 0
	for i < len(tokens) {
		t := strings.ToLower(tokens[i])
		if t == "from" && i+1 < len(tokens) {
			ip = tokens[i+1]
			i += 2
		} else if t == "proto" && i+1 < len(tokens) {
			proto = strings.ToLower(tokens[i+1])
			i += 2
		} else if t == "port" && i+1 < len(tokens) {
			port = tokens[i+1]
			i += 2
		} else if t == "to" && i+1 < len(tokens) && strings.ToLower(tokens[i+1]) == "any" {
			i += 2
		} else {
			i++
		}
	}

	if port == "" {
		return Rule{}, fmt.Errorf("could not identify the port in the rule")
	}
	if ip == "anywhere" {
		ip = "any"
	}

	return Rule{IP: ip, Port: port, Proto: proto}, nil
}
