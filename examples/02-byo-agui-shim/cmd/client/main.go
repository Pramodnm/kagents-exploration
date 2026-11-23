// Host client: sends one message to each of the three kagent-hosted AG-UI
// agents over kagent's A2A JSON-RPC endpoint and checks that each reply
// completes and is non-empty.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type rpcResp struct {
	Result struct {
		Task struct {
			Status struct {
				State   string `json:"state"`
				Message struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"message"`
			} `json:"status"`
			Artifacts []struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"artifacts"`
		} `json:"task"`
	} `json:"result"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func send(base, agent, text string) (string, string, error) {
	body, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": "1", "method": "SendMessage",
		"params": map[string]any{"message": map[string]any{
			"messageId": fmt.Sprintf("client-%s-%d", agent, time.Now().UnixNano()),
			"role":      "ROLE_USER",
			"parts":     []map[string]string{{"text": text}},
		}},
	})
	req, err := http.NewRequest(http.MethodPost, base+"/agents/kagent/"+agent, bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("A2A-Version", "1.0")
	client := &http.Client{Timeout: 4 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var r rpcResp
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", "", fmt.Errorf("decode: %w; body=%s", err, raw)
	}
	if r.Error != nil {
		return "", "", fmt.Errorf("rpc error: %s", r.Error.Message)
	}
	state := r.Result.Task.Status.State
	var reply strings.Builder
	for _, p := range r.Result.Task.Status.Message.Parts {
		reply.WriteString(p.Text)
	}
	for _, a := range r.Result.Task.Artifacts {
		for _, p := range a.Parts {
			reply.WriteString(p.Text)
		}
	}
	return state, reply.String(), nil
}

func main() {
	base := os.Getenv("KAGENT_URL")
	if base == "" {
		base = "http://localhost:8083"
	}
	prompt := "Reply in one short sentence naming yourself."
	failed := 0
	for _, agent := range []string{"agui-assistant", "agui-coder", "agui-tutor"} {
		state, reply, err := send(base, agent, prompt)
		ok := err == nil && state == "TASK_STATE_COMPLETED" && strings.TrimSpace(reply) != ""
		if !ok {
			failed++
		}
		fmt.Printf("%-15s state=%-22s ok=%-5v reply=%q err=%v\n", agent, state, ok, reply, err)
	}
	if failed > 0 {
		fmt.Printf("FAILED: %d of 3 agents\n", failed)
		os.Exit(1)
	}
	fmt.Println("PASS: 3 of 3 agents completed with a reply")
}
