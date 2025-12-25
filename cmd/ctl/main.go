package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"
)

const (
	DefaultAPIAddr = "http://localhost:8475"
)

type ClientDetail struct {
	ClientID    string `json:"client_id"`
	RemoteAddr  string `json:"remote_addr"`
	ConnectedAt int64  `json:"connected_at"`
	Uptime      string `json:"uptime"`
}

type ListClientsResponse struct {
	Total   int            `json:"total"`
	Clients []ClientDetail `json:"clients"`
}

type SendRequest struct {
	ClientID string `json:"client_id"`
	Type     string `json:"type"`
	Payload  string `json:"payload"`
	WaitAck  bool   `json:"wait_ack"`
}

type SendResponse struct {
	Success bool   `json:"success"`
	MsgID   string `json:"msg_id"`
	Message string `json:"message"`
}

type BroadcastRequest struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type BroadcastResponse struct {
	Success      bool     `json:"success"`
	MsgID        string   `json:"msg_id"`
	Total        int      `json:"total"`
	SuccessCount int      `json:"success_count"`
	FailedCount  int      `json:"failed_count"`
	Errors       []string `json:"errors,omitempty"`
}

func main() {
	// 子命令
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	listAPI := listCmd.String("api", DefaultAPIAddr, "API server address")

	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	sendAPI := sendCmd.String("api", DefaultAPIAddr, "API server address")
	sendClient := sendCmd.String("client", "", "Target client ID (required)")
	sendType := sendCmd.String("type", "command", "Message type (command|event|query|response)")
	sendPayload := sendCmd.String("payload", "", "Message payload (JSON string, required)")
	sendWaitAck := sendCmd.Bool("wait-ack", false, "Wait for acknowledgment")

	broadcastCmd := flag.NewFlagSet("broadcast", flag.ExitOnError)
	broadcastAPI := broadcastCmd.String("api", DefaultAPIAddr, "API server address")
	broadcastType := broadcastCmd.String("type", "event", "Message type (command|event|query|response)")
	broadcastPayload := broadcastCmd.String("payload", "", "Message payload (JSON string, required)")

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "list":
		listCmd.Parse(os.Args[2:])
		if err := listClients(*listAPI); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "send":
		sendCmd.Parse(os.Args[2:])
		if *sendClient == "" {
			fmt.Fprintf(os.Stderr, "Error: -client is required\n")
			sendCmd.Usage()
			os.Exit(1)
		}
		if *sendPayload == "" {
			fmt.Fprintf(os.Stderr, "Error: -payload is required\n")
			sendCmd.Usage()
			os.Exit(1)
		}
		if err := sendMessage(*sendAPI, *sendClient, *sendType, *sendPayload, *sendWaitAck); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "broadcast":
		broadcastCmd.Parse(os.Args[2:])
		if *broadcastPayload == "" {
			fmt.Fprintf(os.Stderr, "Error: -payload is required\n")
			broadcastCmd.Usage()
			os.Exit(1)
		}
		if err := broadcastMessage(*broadcastAPI, *broadcastType, *broadcastPayload); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "help", "-h", "--help":
		printUsage()

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("QUIC Backbone CLI - Server Management Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  quic-ctl <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  list                List all connected clients")
	fmt.Println("  send                Send a message to a specific client")
	fmt.Println("  broadcast           Broadcast a message to all clients")
	fmt.Println("  help                Show this help message")
	fmt.Println()
	fmt.Println("List Options:")
	fmt.Println("  -api <addr>         API server address (default: http://localhost:8475)")
	fmt.Println()
	fmt.Println("Send Options:")
	fmt.Println("  -api <addr>         API server address (default: http://localhost:8475)")
	fmt.Println("  -client <id>        Target client ID (required)")
	fmt.Println("  -type <type>        Message type: command|event|query|response (default: command)")
	fmt.Println("  -payload <json>     Message payload as JSON string (required)")
	fmt.Println("  -wait-ack           Wait for acknowledgment from client")
	fmt.Println()
	fmt.Println("Broadcast Options:")
	fmt.Println("  -api <addr>         API server address (default: http://localhost:8475)")
	fmt.Println("  -type <type>        Message type: command|event|query|response (default: event)")
	fmt.Println("  -payload <json>     Message payload as JSON string (required)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # List all connected clients")
	fmt.Println("  quic-ctl list")
	fmt.Println()
	fmt.Println("  # Send a command to a specific client")
	fmt.Println(`  quic-ctl send -client client-001 -type command -payload '{"action":"restart"}'`)
	fmt.Println()
	fmt.Println("  # Broadcast an event to all clients")
	fmt.Println(`  quic-ctl broadcast -type event -payload '{"event":"update_available"}'`)
	fmt.Println()
}

func listClients(apiAddr string) error {
	url := fmt.Sprintf("%s/api/clients", apiAddr)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to API server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result ListClientsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Printf("Connected Clients: %d\n\n", result.Total)

	if result.Total == 0 {
		fmt.Println("No clients connected.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CLIENT ID\tREMOTE ADDRESS\tUPTIME\tCONNECTED AT")
	fmt.Fprintln(w, "---------\t--------------\t------\t------------")

	for _, client := range result.Clients {
		connectedTime := time.UnixMilli(client.ConnectedAt).Format("2006-01-02 15:04:05")
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			client.ClientID,
			client.RemoteAddr,
			client.Uptime,
			connectedTime,
		)
	}

	w.Flush()
	return nil
}

func sendMessage(apiAddr, clientID, msgType, payload string, waitAck bool) error {
	url := fmt.Sprintf("%s/api/send", apiAddr)

	req := SendRequest{
		ClientID: clientID,
		Type:     msgType,
		Payload:  payload,
		WaitAck:  waitAck,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to connect to API server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result SendResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Printf("✅ Message sent successfully\n")
	fmt.Printf("   Client ID: %s\n", clientID)
	fmt.Printf("   Message ID: %s\n", result.MsgID)
	fmt.Printf("   Type: %s\n", msgType)
	fmt.Printf("   Payload: %s\n", payload)
	if waitAck {
		fmt.Printf("   Wait Ack: true\n")
	}

	return nil
}

func broadcastMessage(apiAddr, msgType, payload string) error {
	url := fmt.Sprintf("%s/api/broadcast", apiAddr)

	req := BroadcastRequest{
		Type:    msgType,
		Payload: payload,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to connect to API server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result BroadcastResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Printf("✅ Message broadcast completed\n")
	fmt.Printf("   Message ID: %s\n", result.MsgID)
	fmt.Printf("   Type: %s\n", msgType)
	fmt.Printf("   Payload: %s\n", payload)
	fmt.Printf("   Total Clients: %d\n", result.Total)
	fmt.Printf("   Success: %d\n", result.SuccessCount)
	fmt.Printf("   Failed: %d\n", result.FailedCount)

	if len(result.Errors) > 0 {
		fmt.Printf("\nErrors:\n")
		for _, errMsg := range result.Errors {
			fmt.Printf("  - %s\n", errMsg)
		}
	}

	return nil
}
