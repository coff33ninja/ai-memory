package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

type Request struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      interface{}      `json:"id,omitempty"`
	Method  string           `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Server struct {
	name    string
	version string
	mu      sync.Mutex
	writer  *bufio.Writer
	reader  *bufio.Reader

	tools      []ToolDef
	resources  []ResourceDef
	templates  []TemplateDef
	prompts    []PromptDef

	handleTool      func(name string, args map[string]interface{}) (interface{}, error)
	handleResource  func(uri string) (interface{}, error)
	handlePrompt    func(name string, args map[string]interface{}) (interface{}, error)

	database interface{}
	searcher interface{}
	syncer   interface{}
	indexer  interface{}
	store    interface{}
	emb      interface{}
	pm       interface{}
	eng      interface{}
}

func New(name, version string) *Server {
	return &Server{
		name:    name,
		version: version,
		writer:  bufio.NewWriter(os.Stdout),
		reader:  bufio.NewReader(os.Stdin),
	}
}

func NewServer(version string) *Server {
	return New("ai-memory", version)
}

func (s *Server) SetHandlers(
	database interface{},
	searcher interface{},
	syncer interface{},
	indexer interface{},
	store interface{},
	emb interface{},
	pm interface{},
	eng interface{},
) {
	s.database = database
	s.searcher = searcher
	s.syncer = syncer
	s.indexer = indexer
	s.store = store
	s.emb = emb
	s.pm = pm
	s.eng = eng
}

func (s *Server) Serve() error {
	return s.Run()
}

func (s *Server) SetToolHandler(fn func(string, map[string]interface{}) (interface{}, error)) {
	s.handleTool = fn
}

func (s *Server) SetResourceHandler(fn func(string) (interface{}, error)) {
	s.handleResource = fn
}

func (s *Server) SetPromptHandler(fn func(string, map[string]interface{}) (interface{}, error)) {
	s.handlePrompt = fn
}

func (s *Server) RegisterTools(tools ...ToolDef) {
	s.tools = append(s.tools, tools...)
}

func (s *Server) RegisterResources(resources ...ResourceDef) {
	s.resources = append(s.resources, resources...)
}

func (s *Server) RegisterTemplates(templates ...TemplateDef) {
	s.templates = append(s.templates, templates...)
}

func (s *Server) RegisterPrompts(prompts ...PromptDef) {
	s.prompts = append(s.prompts, prompts...)
}

func (s *Server) Run() error {
	for {
		line, err := s.reader.ReadBytes('\n')
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		s.handleRequest(req)
	}
}

func (s *Server) handleRequest(req Request) {
	var resp Response
	resp.JSONRPC = "2.0"
	resp.ID = req.ID

	switch req.Method {
	case "initialize":
		resp.Result = map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"resources": map[string]interface{}{},
				"tools":     map[string]interface{}{},
				"prompts":   map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    s.name,
				"version": s.version,
			},
		}
	case "notifications/initialized":
		return
	case "tools/list":
		resp.Result = map[string]interface{}{"tools": s.tools}
	case "resources/list":
		resp.Result = map[string]interface{}{"resources": s.resources}
	case "resources/templates/list":
		resp.Result = map[string]interface{}{"resourceTemplates": s.templates}
	case "prompts/list":
		resp.Result = map[string]interface{}{"prompts": s.prompts}
	case "tools/call":
		resp.Result = s.handleToolCall(req.Params)
	case "resources/read":
		resp.Result = s.handleResourceRead(req.Params)
	case "prompts/get":
		resp.Result = s.handlePromptGet(req.Params)
	default:
		resp.Error = &Error{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}
	}

	s.send(resp)
}

func (s *Server) handleToolCall(params json.RawMessage) map[string]interface{} {
	var p struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	json.Unmarshal(params, &p)

	if s.handleTool == nil {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": "no tool handler registered"},
			},
			"isError": true,
		}
	}

	result, err := s.handleTool(p.Name, p.Arguments)
	if err != nil {
		return map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": fmt.Sprintf("error: %v", err)},
			},
			"isError": true,
		}
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{"type": "text", "text": formatResult(result)},
		},
	}
}

func (s *Server) handleResourceRead(params json.RawMessage) map[string]interface{} {
	var p struct {
		URI string `json:"uri"`
	}
	json.Unmarshal(params, &p)

	if s.handleResource == nil {
		return map[string]interface{}{
			"contents": []map[string]interface{}{},
		}
	}

	result, err := s.handleResource(p.URI)
	if err != nil {
		return map[string]interface{}{
			"contents": []map[string]interface{}{
				{"uri": p.URI, "text": fmt.Sprintf("error: %v", err)},
			},
		}
	}

	return map[string]interface{}{
		"contents": []map[string]interface{}{
			{"uri": p.URI, "mimeType": "application/json", "text": formatResult(result)},
		},
	}
}

func (s *Server) handlePromptGet(params json.RawMessage) map[string]interface{} {
	var p struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	json.Unmarshal(params, &p)

	if s.handlePrompt == nil {
		return map[string]interface{}{
			"messages": []map[string]interface{}{},
		}
	}

	result, err := s.handlePrompt(p.Name, p.Arguments)
	if err != nil {
		return map[string]interface{}{
			"messages": []map[string]interface{}{
				{
					"role":    "assistant",
					"content": map[string]interface{}{"type": "text", "text": fmt.Sprintf("error: %v", err)},
				},
			},
		}
	}

	return result.(map[string]interface{})
}

func (s *Server) send(resp Response) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, _ := json.Marshal(resp)
	s.writer.Write(data)
	s.writer.WriteByte('\n')
	s.writer.Flush()
}

func formatResult(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	default:
		data, _ := json.MarshalIndent(val, "", "  ")
		return string(data)
	}
}
