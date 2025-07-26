// SPDX-License-Identifier: GPL-2.0-or-later
// Copyright 2025 Joe Lesko

package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"flag"
	"fmt"
	"net"
	"os"
	"time"
	"runtime"
	"os/exec"

	"github.com/beevik/ntp"
)

func main() {
	// Parse command line flags
	var loop int
	var espCommand, secretKey string
	cls := flag.Bool("cls", false, "Clear screen between update")
	flag.IntVar(&loop, "loop", 0, "Loop every loop seconds")
	flag.StringVar(&espCommand, "c", "", "esp32NTP command (reboot, reset wifi, stats, display)")
	flag.StringVar(&secretKey, "p", "MySecretKey123", "Password for HMAC")
	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, "Welcome to ntpclient!\n\n")
		fmt.Fprintf(os.Stdout, "This program displays ntp server values as well as sending command to esp32NTP.\n")
		fmt.Fprintf(os.Stdout, "Usage: %s [options] [server] [server]\n\n", os.Args[0])
		fmt.Fprintf(os.Stdout, "Options:\n")
		flag.PrintDefaults() // Prints flag details
		fmt.Fprintf(os.Stdout, "\nExamples:\n")
		fmt.Fprintf(os.Stdout, "  %s -loop 1 -cls pool.ntp.org\n", os.Args[0])
	}
	flag.Parse()

	servers := flag.Args()
	if len(servers) == 0 {
		servers = []string{"pool.ntp.org"}
	}

	// Validate ESP32 command
	validCommands := map[string]bool{"reboot": true, "reset wifi": true, "stats": true, "display": true}

	if espCommand != "" && !validCommands[espCommand] {
		fmt.Fprintf(os.Stderr, "Invalid ESP32 command: %s. Must be 'reboot', 'reset', 'stats', or 'display'\n", espCommand)
		os.Exit(1)
	}

	// Send UDP command to ESP32 if specified
	if espCommand != "" {
		for _, server := range servers {
			err := sendESPCommand(server, espCommand, secretKey)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error sending ESP32 command: %v\n", err)
				os.Exit(1)
			}
		}
		return
	}

	// Get NTP values from server and output to screen
	var docls bool
	for {
		docls = true
		for _, server := range servers {
			// Get the current time from the NTP server
			currentTime, err := ntp.Query(server)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error querying NTP server %s: %v\n", server, err)
				continue
			}

			localTime := time.Now()

			if *cls && docls {
					// Clear the screen
				switch runtime.GOOS {
				case "windows":
					exec.Command("cls");
				default:
					fmt.Print("\033[H\033[2J")
				}
			}
			docls = false
			// Print the current time
			fmt.Printf("NTP server: %s\n", server)
			fmt.Printf("NTP Stratum: %d\n", currentTime.Stratum)
			fmt.Printf("NTP Time: %s\n", currentTime.Time.Format(time.RFC3339))
			fmt.Printf("Local Time: %s\n", localTime.Format(time.RFC3339))

			// Print the offset from local time
			fmt.Printf("Offset from local time: %s\n", currentTime.ClockOffset)
			// Print the round trip time
			fmt.Printf("Round trip time: %s\n", currentTime.RTT)
			fmt.Println("")
		}
		// Print a separator
		if !*cls && loop > 0 {
			fmt.Printf("--------------------------------------------------\n\n")
		}
		if loop == 0 {
			break
		}
		time.Sleep(time.Duration(loop) * time.Second)
	}
}

// sendESPCommand sends a UDP packet to the ESP32 with the command and HMAC
func sendESPCommand(server, command, secretKey string) error {
	// Compute HMAC-SHA256
	hmacObj := hmac.New(sha256.New, []byte(secretKey))
	hmacObj.Write([]byte(command))
	hmacHex := fmt.Sprintf("%x", hmacObj.Sum(nil))
	packet := fmt.Sprintf("%s:%s", command, hmacHex)

	// Resolve UDP address
	addr, err := net.ResolveUDPAddr("udp", server+":123")
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %v", err)
	}

	// Create UDP connection
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("failed to dial UDP: %v", err)
	}
	defer conn.Close()

	// Send packet
	_, err = conn.Write([]byte(packet))
	if err != nil {
		return fmt.Errorf("failed to send UDP packet: %v", err)
	}
	fmt.Printf("Sent packet to %s:123: %s...\n", server, packet[:7])

	// Set timeout for response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	buffer := make([]byte, 1024)

	// Receive response
	n, _, err := conn.ReadFromUDP(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			fmt.Println("No response received from ESP32")
			return nil
		}
		return fmt.Errorf("failed to read response: %v", err)
	}

	fmt.Printf("Response from ESP32: %s\n", string(buffer[:n]))
	return nil
}
