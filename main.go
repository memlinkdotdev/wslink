package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const version = "0.3.0"

func main() {
	log.SetFlags(0)
	log.SetPrefix("")

	var (
		connect     string
		listenAddr  = "127.0.0.1"
		wslName     string
		windowsHost string
		showVersion bool
	)

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--version":
			showVersion = true
		case "--connect":
			connect = takeArg(args, &i, "--connect")
		case "--listen":
			listenAddr = takeArg(args, &i, "--listen")
		case "--wsl-name":
			wslName = takeArg(args, &i, "--wsl-name")
		case "--windows-host":
			windowsHost = takeArg(args, &i, "--windows-host")
		}
	}

	if showVersion {
		fmt.Printf("wslink %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return
	}

	var positional []string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			positional = append(positional, a)
		}
	}

	if len(positional) < 1 {
		fmt.Println(`wslink - WSL to Windows port bridge

Usage:
  wslink forward <port> [flags]
  wslink <port> [flags]              # "forward" is implicit

Flags:
  --connect <host:port>    Target directly (skip auto-detect)
  --listen <addr>          Listen address (default 127.0.0.1)
  --wsl-name <distro>      WSL distro name (Windows only)
  --windows-host <ip>      Windows host IP (WSL only)
  --version                Print version and exit

Examples:
  wslink forward 8000              # Auto-detect target
  wslink 8000 --connect 192.168.1.5:8000
  wslink forward 8000 --wsl-name Ubuntu
  wslink forward 8000 --windows-host 172.20.0.1`)
		return
	}

	portIdx := 0
	if positional[0] == "forward" {
		portIdx = 1
	}

	if len(positional) <= portIdx {
		log.Fatal("Missing port. Usage: wslink forward <port>")
	}

	portStr := positional[portIdx]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Invalid port: %s", portStr)
	}
	if port < 1 || port > 65535 {
		log.Fatalf("Port out of range (1-65535): %d", port)
	}

	var target string
	if connect != "" {
		target = connect
	} else {
		target = resolveTarget(port, wslName, windowsHost)
	}

	log.Printf("Forwarding %s:%d to %s", listenAddr, port, target)

	startProxy(listenAddr, port, target)
}

func takeArg(args []string, i *int, name string) string {
	if *i+1 < len(args) {
		*i++
		return args[*i]
	}
	log.Fatalf("Missing value for %s", name)
	return ""
}

func resolveTarget(port int, wslName, windowsHost string) string {
	if runtime.GOOS == "windows" {
		return resolveWslTarget(port, wslName)
	}
	return resolveWindowsTarget(port, windowsHost)
}

func resolveWslTarget(port int, wslName string) string {
	distros := listRunningWslDistros()
	if len(distros) == 0 {
		log.Fatal("No running WSL distros found. Start one with: wsl --distribution <name>")
	}

	var distro string
	if wslName != "" {
		found := false
		for _, d := range distros {
			if strings.EqualFold(d, wslName) {
				distro = d
				found = true
				break
			}
		}
		if !found {
			log.Fatalf("WSL distro '%s' not found or not running. Available: %s", wslName, strings.Join(distros, ", "))
		}
	} else {
		distro = distros[0]
	}

	ip := getWslIp(distro)
	if ip == "" {
		log.Fatalf("Could not resolve IP for WSL distro '%s'. Tried: hostname -I and ip addr show eth0", distro)
	}

	log.Printf("Detected WSL distro: %s (%s)", distro, ip)
	return fmt.Sprintf("%s:%d", ip, port)
}

func listRunningWslDistros() []string {
	out, err := runCmd("wsl.exe", "--list", "--running", "--quiet")
	if err != nil || out == "" {
		return nil
	}
	// wsl.exe may output UTF-16LE with embedded null bytes
	out = strings.ReplaceAll(out, "\x00", "")
	var distros []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			distros = append(distros, line)
		}
	}
	return distros
}

func getWslIp(distro string) string {
	out, err := runCmd("wsl.exe", "-d", distro, "--", "hostname", "-I")
	if err == nil {
		out = strings.ReplaceAll(out, "\x00", "")
		ip := strings.TrimSpace(strings.Split(out, " ")[0])
		if ip != "" {
			return ip
		}
	}

	out, err = runCmd("wsl.exe", "-d", distro, "--", "ip", "addr", "show", "eth0")
	if err == nil {
		out = strings.ReplaceAll(out, "\x00", "")
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "inet ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					return strings.Split(parts[1], "/")[0]
				}
			}
		}
	}

	return ""
}

func resolveWindowsTarget(port int, windowsHost string) string {
	if windowsHost == "" {
		windowsHost = detectWindowsHost()
	}
	if windowsHost == "" {
		log.Fatal("Could not detect Windows host IP. Use --windows-host <ip>")
	}
	log.Printf("Detected Windows host: %s", windowsHost)
	return fmt.Sprintf("%s:%d", windowsHost, port)
}

func detectWindowsHost() string {
	// WSL2: Windows host is the default gateway
	out, err := runCmd("sh", "-c", "ip route show default")
	if err == nil {
		for _, line := range strings.Split(out, "\n") {
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "via" && i+1 < len(parts) {
					ip := parts[i+1]
					if ip != "" {
						return ip
					}
				}
			}
		}
	}

	// Fallback: nameserver in resolv.conf (may be a DNS proxy, not the real host)
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "nameserver ") {
			ip := strings.TrimSpace(strings.TrimPrefix(line, "nameserver "))
			if ip != "" {
				return ip
			}
		}
	}
	return ""
}

func startProxy(listenAddr string, port int, target string) {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", listenAddr, port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		log.Print("\nShutting down...")
		ln.Close()
	}()

	var connID int64
	var active sync.WaitGroup

	for {
		conn, err := ln.Accept()
		if err != nil {
			break
		}
		connID++
		id := connID

		active.Add(1)
		go func() {
			defer active.Done()
			handleConn(id, conn, target)
		}()
	}

	active.Wait()
	log.Print("Stopped")
}

func handleConn(id int64, src net.Conn, target string) {
	defer src.Close()

	dst, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		log.Printf("[%d] connect failed: %v", id, err)
		return
	}
	defer dst.Close()

	log.Printf("[%d] open  %s - %s", id, src.RemoteAddr(), target)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		io.Copy(dst, src)
		dst.Close()
		wg.Done()
	}()
	go func() {
		io.Copy(src, dst)
		src.Close()
		wg.Done()
	}()

	wg.Wait()
	log.Printf("[%d] close %s - %s", id, src.RemoteAddr(), target)
}

func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
