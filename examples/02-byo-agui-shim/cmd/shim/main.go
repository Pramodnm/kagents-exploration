package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	a2atype "github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	"github.com/kagent-dev/kagent/go/adk/pkg/app"
	"github.com/kagent-dev/kagent/go/adk/pkg/config"
	"github.com/kagent-dev/kagent/go/pkg/logging"
	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/provider/aguiprovider"
	"github.com/microsoft/agent-framework-go/provider/openaiprovider"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// personas are the three Microsoft Agent Framework agents served over AG-UI.
var personas = map[string]string{
	"assistant": "You are a concise general assistant.",
	"coder":     "You are a careful coding assistant. Answer with short code.",
	"tutor":     "You are a patient tutor. Explain step by step in simple words.",
}

// aguiExecutor is the kagent A2A side. It forwards each message to the
// in-process AG-UI endpoint for this Harness's persona and returns the reply.
type aguiExecutor struct {
	endpoint string
}

func (e aguiExecutor) Execute(ctx context.Context, req *a2asrv.ExecutorContext) iter.Seq2[a2atype.Event, error] {
	return func(yield func(a2atype.Event, error) bool) {
		if !yield(a2atype.NewSubmittedTask(req, req.Message), nil) {
			return
		}
		text, err := e.ask(ctx, req)
		state := a2atype.TaskStateCompleted
		msg := a2atype.NewMessage(a2atype.MessageRoleAgent, a2atype.NewTextPart(text))
		if err != nil {
			state = a2atype.TaskStateFailed
			msg = a2atype.NewMessage(a2atype.MessageRoleAgent, a2atype.NewTextPart("agent error: "+err.Error()))
		}
		msg.ContextID, msg.TaskID = req.ContextID, req.TaskID
		yield(a2atype.NewStatusUpdateEvent(req, state, msg), nil)
	}
}

func (e aguiExecutor) Cancel(context.Context, *a2asrv.ExecutorContext) iter.Seq2[a2atype.Event, error] {
	return func(func(a2atype.Event, error) bool) {}
}

// ask posts one AG-UI run and concatenates the TEXT_MESSAGE_CONTENT deltas.
func (e aguiExecutor) ask(ctx context.Context, req *a2asrv.ExecutorContext) (string, error) {
	var parts []string
	if req.Message != nil {
		for _, p := range req.Message.Parts {
			parts = append(parts, p.Text())
		}
	}
	body, _ := json.Marshal(map[string]any{
		"threadId": cmp.Or(req.ContextID, "ctx"),
		"runId":    cmp.Or(req.TaskID, "run"),
		"messages": []map[string]string{{"id": "m1", "role": "user", "content": strings.Join(parts, "")}},
		"tools":    []any{},
		"context":  []any{},
		"state":    map[string]any{},
		"forwardedProps": map[string]any{},
	})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, e.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out strings.Builder
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var ev struct {
			Type  string `json:"type"`
			Delta string `json:"delta"`
		}
		if json.Unmarshal([]byte(line[len("data: "):]), &ev) == nil && ev.Type == "TEXT_MESSAGE_CONTENT" {
			out.WriteString(ev.Delta)
		}
	}
	if out.Len() == 0 {
		return "", fmt.Errorf("no text from %s (status %d)", e.endpoint, resp.StatusCode)
	}
	return out.String(), nil
}

// startAGUI binds 127.0.0.1:8080 before returning, so the A2A side never sees
// connection refused, and serves one AG-UI endpoint per persona.
func startAGUI(baseURL, model, apiKey string) error {
	client := openai.NewClient(option.WithBaseURL(baseURL), option.WithAPIKey(apiKey))
	mux := http.NewServeMux()
	for name, instructions := range personas {
		a := openaiprovider.NewAgent(client, openaiprovider.AgentConfig{
			Config:             agent.Config{Name: name},
			Model:              model,
			Instructions:       instructions,
			DisableStoreOutput: true,
		})
		mux.Handle("/"+name, aguiprovider.NewJSONHTTPHandler(a, aguiprovider.HandlerConfig{}))
	}
	ln, err := net.Listen("tcp", "127.0.0.1:8080")
	if err != nil {
		return err
	}
	go func() { log.Fatal(http.Serve(ln, mux)) }()
	return nil
}

func main() {
	if err := config.MaterializeFromBootstrap(context.Background(), "/config"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	compiled, err := config.LoadAgentConfig("/config/config.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	runtime := config.ProcessConfig(compiled)
	logger, _ := logging.New(os.Stderr, cmp.Or(runtime.LogLevel, "info"))

	persona := cmp.Or(os.Getenv("AGUI_PATH"), "assistant")
	if _, ok := personas[persona]; !ok {
		log.Fatalf("AGUI_PATH %q is not a known persona", persona)
	}
	// The gateway replaces the placeholder Authorization header with the real
	// key on the way out, so the Actor never holds it.
	apiKey := cmp.Or(os.Getenv("OPENAI_API_KEY"), "placeholder")
	if err := startAGUI(os.Getenv("GATEWAY_BASE_URL"), os.Getenv("GATEWAY_MODEL"), apiKey); err != nil {
		log.Fatalf("start AG-UI: %v", err)
	}

	application, err := app.New(app.AppConfig{
		ControlPlaneURL:  runtime.ControlPlaneUrl,
		MaxContentLength: runtime.MaxContentLength,
		AgentCard: a2atype.AgentCard{Name: "agui_" + persona, Version: "v1",
			Capabilities: a2atype.AgentCapabilities{Streaming: true}},
		Port:    fmt.Sprint(cmp.Or(runtime.GetPort(), 80)),
		AppName: cmp.Or(runtime.Name, "agui-"+persona),
		Logger:  logger,
	}, aguiExecutor{endpoint: "http://127.0.0.1:8080/" + persona})
	if err != nil {
		logger.Error("failed to create A2A application", "error", err)
		return
	}
	if err := application.Run(); err != nil {
		logger.Error("A2A application stopped", "error", err)
	}
}
