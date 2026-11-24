package eval

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// A2AClient talks to a kagent agent through the controller's A2A endpoint
// (JSON-RPC 2.0 over HTTP, method "message/send").
type A2AClient struct {
	// BaseURL is the controller address, e.g. "http://127.0.0.1:18083".
	BaseURL string
	HTTP    *http.Client
}

// Step is one entry of the agent's transcript.
type Step struct {
	Role   string          `json:"role"`             // "user" or "agent"
	Kind   string          `json:"kind"`             // "text", "tool_call" or "tool_result"
	Text   string          `json:"text,omitempty"`   // for text steps
	Tool   string          `json:"tool,omitempty"`   // for tool steps
	Args   json.RawMessage `json:"args,omitempty"`   // tool_call arguments
	Result string          `json:"result,omitempty"` // tool_result output
}

// Reply is the agent's answer plus everything it did to get there.
type Reply struct {
	Answer    string          `json:"answer"`
	State     string          `json:"state"`
	ContextID string          `json:"contextId"`
	Steps     []Step          `json:"steps"`
	ToolCalls int             `json:"toolCalls"`
	Tokens    int             `json:"tokens"` // sum of totalTokenCount over all model calls
	Raw       json.RawMessage `json:"-"`      // the complete JSON-RPC response
}

type rpcRequest struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      string    `json:"id"`
	Method  string    `json:"method"`
	Params  rpcParams `json:"params"`
}

type rpcParams struct {
	Message a2aMessage `json:"message"`
}

type a2aMessage struct {
	Kind      string          `json:"kind,omitempty"`
	MessageID string          `json:"messageId,omitempty"`
	Role      string          `json:"role"`
	Parts     []a2aPart       `json:"parts"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

type a2aPart struct {
	Kind     string          `json:"kind"`
	Text     string          `json:"text,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
	Metadata struct {
		ADKType string `json:"adk_type"`
	} `json:"metadata"`
}

type rpcResponse struct {
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Result *struct {
		Kind      string `json:"kind"`
		ContextID string `json:"contextId"`
		Status    struct {
			State   string      `json:"state"`
			Message *a2aMessage `json:"message"`
		} `json:"status"`
		Artifacts []struct {
			Parts []a2aPart `json:"parts"`
		} `json:"artifacts"`
		History []a2aMessage `json:"history"`
		// Present when the result is a plain message instead of a task.
		Parts []a2aPart `json:"parts"`
	} `json:"result"`
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Send asks the agent one question in a fresh context and waits for the
// final answer.
func (c A2AClient) Send(ctx context.Context, namespace, agent, text string) (Reply, error) {
	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0", ID: newID(), Method: "message/send",
		Params: rpcParams{Message: a2aMessage{
			Kind: "message", MessageID: newID(), Role: "user",
			Parts: []a2aPart{{Kind: "text", Text: text}},
		}},
	})
	if err != nil {
		return Reply{}, err
	}
	url := fmt.Sprintf("%s/api/a2a/%s/%s/", strings.TrimRight(c.BaseURL, "/"), namespace, agent)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return Reply{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return Reply{}, fmt.Errorf("call agent: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return Reply{}, fmt.Errorf("read agent response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return Reply{Raw: raw}, fmt.Errorf("agent endpoint answered HTTP %s: %s", resp.Status, truncate(string(raw), 300))
	}
	return ParseReply(raw)
}

// ParseReply turns a raw JSON-RPC response into a Reply.
func ParseReply(raw []byte) (Reply, error) {
	reply := Reply{Raw: raw}
	var r rpcResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return reply, fmt.Errorf("agent response is not JSON: %w", err)
	}
	if r.Error != nil {
		return reply, fmt.Errorf("agent returned an error (%d): %s", r.Error.Code, r.Error.Message)
	}
	if r.Result == nil {
		return reply, fmt.Errorf("agent response has no result")
	}
	res := r.Result
	reply.State = res.Status.State
	reply.ContextID = res.ContextID

	var prevUser string
	for _, m := range res.History {
		reply.Tokens += usageTokens(m.Metadata)
		for _, p := range m.Parts {
			step, ok := toStep(m.Role, p)
			if !ok {
				continue
			}
			// The runtime echoes the user message; keep one copy.
			if step.Role == "user" && step.Kind == "text" {
				if step.Text == prevUser {
					continue
				}
				prevUser = step.Text
			}
			if step.Kind == "tool_call" {
				reply.ToolCalls++
			}
			reply.Steps = append(reply.Steps, step)
		}
	}

	// The answer: artifact text, else the task's status message, else a
	// plain message result, else the last thing the agent said.
	var texts []string
	for _, a := range res.Artifacts {
		texts = append(texts, textOf(a.Parts)...)
	}
	if len(texts) == 0 && res.Status.Message != nil {
		texts = textOf(res.Status.Message.Parts)
	}
	if len(texts) == 0 {
		texts = textOf(res.Parts)
	}
	if len(texts) == 0 {
		for i := len(reply.Steps) - 1; i >= 0; i-- {
			if s := reply.Steps[i]; s.Role == "agent" && s.Kind == "text" {
				texts = []string{s.Text}
				break
			}
		}
	}
	reply.Answer = strings.TrimSpace(strings.Join(texts, "\n"))
	if reply.State != "" && reply.State != "completed" {
		return reply, fmt.Errorf("agent task ended in state %q", reply.State)
	}
	return reply, nil
}

func textOf(parts []a2aPart) []string {
	var out []string
	for _, p := range parts {
		if p.Kind == "text" && strings.TrimSpace(p.Text) != "" {
			out = append(out, p.Text)
		}
	}
	return out
}

func toStep(role string, p a2aPart) (Step, bool) {
	switch {
	case p.Kind == "text":
		if strings.TrimSpace(p.Text) == "" {
			return Step{}, false
		}
		return Step{Role: role, Kind: "text", Text: p.Text}, true
	case p.Kind == "data" && p.Metadata.ADKType == "function_call":
		var d struct {
			Name string          `json:"name"`
			Args json.RawMessage `json:"args"`
		}
		_ = json.Unmarshal(p.Data, &d)
		return Step{Role: role, Kind: "tool_call", Tool: d.Name, Args: d.Args}, true
	case p.Kind == "data" && p.Metadata.ADKType == "function_response":
		var d struct {
			Name     string          `json:"name"`
			Response json.RawMessage `json:"response"`
		}
		_ = json.Unmarshal(p.Data, &d)
		return Step{Role: role, Kind: "tool_result", Tool: d.Name, Result: flattenToolOutput(d.Response)}, true
	}
	return Step{}, false
}

// flattenToolOutput unwraps the {"output": {"output": "..."}} envelope the
// runtime puts around tool results, so transcripts show readable text.
func flattenToolOutput(raw json.RawMessage) string {
	cur := raw
	for range 4 {
		var s string
		if json.Unmarshal(cur, &s) == nil {
			return s
		}
		var obj map[string]json.RawMessage
		if json.Unmarshal(cur, &obj) != nil || len(obj) != 1 {
			break
		}
		for _, v := range obj {
			cur = v
		}
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, cur, "", "  ") == nil {
		return pretty.String()
	}
	return string(cur)
}

func usageTokens(metadata json.RawMessage) int {
	if len(metadata) == 0 {
		return 0
	}
	var m struct {
		Usage struct {
			Total int `json:"totalTokenCount"`
		} `json:"adk_usage_metadata"`
	}
	_ = json.Unmarshal(metadata, &m)
	return m.Usage.Total
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
