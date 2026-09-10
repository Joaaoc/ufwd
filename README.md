# UFWD - Uncomplicated Firewall for Docker

**UFWD** is a compiled CLI tool written in Go that securely manages Docker's external port exposures while respecting your system's UFW (Uncomplicated Firewall) rules.

## 🚨 The Problem

If you run Docker on a Linux server protected by UFW, you've likely encountered a critical security flaw: **Docker completely bypasses UFW.** 
When you map a port using `docker run -p 8080:80`, Docker injects raw `iptables` rules into the `PREROUTING` chain. This means the port is exposed directly to the internet, entirely ignoring any UFW block rules you have set up.

## 🛡️ The Solution (UFWD)

UFWD fixes this gracefully. It utilizes the official `DOCKER-USER` iptables chain (which Docker specifically provides for custom user rules) and manages it automatically. 
Instead of relying on fragile bash scripts, UFWD is a static Go binary that applies rules using **atomic transactions** (`iptables-restore`). This ensures that your firewall is never caught in an inconsistent state.

### Key Features
* **UFW-like Syntax:** Built to mimic some of UFW's natural language (`ufwd allow 3000/tcp`, `ufwd status numbered`).
* **Atomic Reloads:** Uses `iptables-restore` to swap firewall states in milliseconds with zero packet loss.
* **Smart Conntrack:** Leverages `ctorigdstport` to filter packets *before* Docker's NAT kicks in, avoiding the headache of chasing internal container IP addresses.
* **Reboot Resilient:** Installs native hooks into `/etc/ufw/after.init` to ensure rules survive reboots and UFW restarts seamlessly.

---

## 🚀 Installation

### Prerequisites
Before installing UFWD, ensure your Linux distribution has the following standard tools installed:
* `ufw` (Uncomplicated Firewall)
* `iptables`
* `curl`
* `tar`

UFWD provides a safe, fully automated installation script. It detects your architecture (AMD64 or ARM64), downloads the latest release, and configures the hooks.

Run the following command on your Linux server:
```bash
curl -sL https://raw.githubusercontent.com/Joaaoc/ufwd/main/install.sh | sudo bash
```

*(Note: UFWD's installer will safely check for these and abort if any are missing, so it won't break your system).*

### Manual Installation (Build from source)
If you prefer to compile it yourself, you can clone this repository and build it:
```bash
go build -o ./bin/ufwd ./src
sudo mv ./bin/ufwd /usr/local/bin/ufwd
sudo ufwd install
```

---

## 📖 Usage

Using UFWD feels exactly like using UFW. All commands require `sudo`.

### Allowing Ports
```bash
# Allow a port from anywhere
sudo ufwd allow 3000/tcp

# Allow multiple ports or ranges
sudo ufwd allow 3000,3005/tcp
sudo ufwd allow 2999:3005/udp

# Allow a port from a specific IP address
sudo ufwd allow proto udp from 192.168.1.100 port 3306
```

### Viewing Rules
List your active firewall rules with beautifully formatted tables.
```bash
sudo ufwd status
sudo ufwd status numbered  # or 'status -n'
```

### Deleting Rules
You can delete rules by matching their exact definition, or by their row number.
```bash
# Delete by exact rule string
sudo ufwd delete allow 3000/tcp

# Delete by rule number (from 'status numbered')
sudo ufwd delete 2
```

### Inserting Rules (Ordering)
You can inject a rule at a specific position.
```bash
sudo ufwd insert 1 allow proto tcp from 10.0.0.5 port 5432
```

### Lifecycle Commands
* `sudo ufwd disable` - Disables UFWD protection (opens up all mapped Docker ports to the world).
* `sudo ufwd enable` - Re-enables protection and locks down Docker.
* `sudo ufwd reset` - Wipes all customized rules and starts fresh.
* `sudo ufwd uninstall` - Safely removes all UFWD hooks from your system and restores the default Docker networking behavior.

*(Tip: Destructive commands prompt for `y/N`. Use the `-f` or `--force` flag to skip confirmations in automated scripts).*

---

## 🧠 Architecture details
* **State File:** Rules are persisted cleanly in `/etc/ufwd/rules.json`.
* **Hooking:** UFWD integrates gracefully via `/etc/ufw/after.init`, tying its lifecycle natively to the UFW daemon.
* **Fail-Safe:** The final rule injected by UFWD is a rigid `DROP`. Any external traffic hitting a Docker container that hasn't been explicitly allowed by UFWD is silently discarded.

## 🤝 License
MIT
