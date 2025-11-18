package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// AttachClientCommands wires `todoctl client ...` subcommands for interacting with
// the running API. This keeps main.go slim and keeps HTTP wiring here.
func AttachClientCommands(root *cobra.Command) {
	clientCmd := &cobra.Command{
		Use:   "client",
		Short: "Call the running HTTP API",
	}

	var (
		apiAddr   = defaultAPIAddr()
		apiKey    string
		todoItems []string
		todoID    int
	)

	clientCmd.PersistentFlags().StringVar(&apiAddr, "addr", apiAddr, "API base address")
	clientCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key (falls back to APP_API_KEY env)")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create todos via the API",
		RunE: func(cmd *cobra.Command, args []string) error {
			todos, err := buildTodosFromFlags(todoItems)
			if err != nil {
				return err
			}
			return doAPIRequest(apiAddr, apiKey, http.MethodPost, "/todos", map[string]any{"todos": todos})
		},
	}
	createCmd.Flags().StringSliceVar(&todoItems, "todo", nil, "todo item(s). Format: title|description|due (due as RFC3339). Repeatable.")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get a todo by id",
		RunE: func(cmd *cobra.Command, args []string) error {
			if todoID == 0 {
				return fmt.Errorf("--id required")
			}
			return doAPIRequest(apiAddr, apiKey, http.MethodGet, fmt.Sprintf("/todos/%d", todoID), nil)
		},
	}
	getCmd.Flags().IntVar(&todoID, "id", 0, "todo id")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List todos",
		RunE: func(cmd *cobra.Command, args []string) error {
			return doAPIRequest(apiAddr, apiKey, http.MethodGet, "/todos", nil)
		},
	}

	clientCmd.AddCommand(createCmd, getCmd, listCmd)
	root.AddCommand(clientCmd)
}

func doAPIRequest(apiAddr, apiKey, method, path string, payload any) error {
	url := strings.TrimSuffix(apiAddr, "/") + path
	var body io.Reader
	if payload != nil {
		buf, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewBuffer(buf)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	if key := resolveAPIKey(apiKey); key != "" {
		req.Header.Set("X-API-Key", key)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := newHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	fmt.Fprintf(stdOut, "%s %s -> %d\n", method, url, resp.StatusCode)
	respBody, _ := io.ReadAll(resp.Body)
	if len(respBody) > 0 {
		fmt.Fprintln(stdOut, string(respBody))
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("request failed: %s", resp.Status)
	}
	return nil
}

// buildTodosFromFlags parses --todo flags. Format per item:
// "title|description|due" where description/due are optional. Due must be RFC3339.
func buildTodosFromFlags(todoItems []string) ([]map[string]any, error) {
	if len(todoItems) == 0 {
		return nil, fmt.Errorf("at least one --todo \"title|description|due\" required")
	}
	var todos []map[string]any
	for _, raw := range todoItems {
		parts := strings.Split(raw, "|")
		if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("todo title is required in %q", raw)
		}
		item := map[string]any{
			"title": strings.TrimSpace(parts[0]),
		}
		if len(parts) > 1 && strings.TrimSpace(parts[1]) != "" {
			item["description"] = strings.TrimSpace(parts[1])
		}
		if len(parts) > 2 && strings.TrimSpace(parts[2]) != "" {
			due, err := time.Parse(time.RFC3339, strings.TrimSpace(parts[2]))
			if err != nil {
				return nil, fmt.Errorf("invalid due date %q: %w", parts[2], err)
			}
			item["due_date"] = due.Format(time.RFC3339)
		}
		todos = append(todos, item)
	}
	return todos, nil
}

func resolveAPIKey(flagVal string) string {
	if flagVal != "" {
		return flagVal
	}
	return os.Getenv("APP_API_KEY")
}

func defaultAPIAddr() string {
	if v := os.Getenv("APP_API_ADDR"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

var newHTTPClient = func() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

// stdOut is overridden in tests to capture output.
var stdOut io.Writer = os.Stdout
