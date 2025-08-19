package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/lacymorrow/lash/internal/config"
	"github.com/lacymorrow/lash/internal/csync"
	"github.com/lacymorrow/lash/internal/llm/tools"
	"github.com/lacymorrow/lash/internal/permission"
	"github.com/lacymorrow/lash/internal/pubsub"
	"github.com/lacymorrow/lash/internal/version"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPState represents the current state of an MCP client
type MCPState int

const (
	MCPStateDisabled MCPState = iota
	MCPStateStarting
	MCPStateConnected
	MCPStateError
)

func (s MCPState) String() string {
	switch s {
	case MCPStateDisabled:
		return "disabled"
	case MCPStateStarting:
		return "starting"
	case MCPStateConnected:
		return "connected"
	case MCPStateError:
		return "error"
	default:
		return "unknown"
	}
}

// MCPEventType represents the type of MCP event
type MCPEventType string

const (
	MCPEventStateChanged MCPEventType = "state_changed"
)

// MCPEvent represents an event in the MCP system
type MCPEvent struct {
	Type      MCPEventType
	Name      string
	State     MCPState
	Error     error
	ToolCount int
}

// MCPClientInfo holds information about an MCP client's state
type MCPClientInfo struct {
	Name        string
	State       MCPState
	Error       error
	Client      *client.Client
	ToolCount   int
	ConnectedAt time.Time
}

var (
	mcpToolsOnce sync.Once
	mcpTools     []tools.BaseTool
	mcpClients   = csync.NewMap[string, *client.Client]()
	mcpStates    = csync.NewMap[string, MCPClientInfo]()
	mcpBroker    = pubsub.NewBroker[MCPEvent]()
	// Keep original MCP configs so we can restart clients on transport errors.
	mcpClientConfigs = csync.NewMap[string, config.MCPConfig]()
	// Per-client locks to prevent concurrent restarts.
	mcpClientLocks = csync.NewMap[string, *sync.Mutex]()
)

type McpTool struct {
	mcpName     string
	tool        mcp.Tool
	permissions permission.Service
	workingDir  string
}

func (b *McpTool) Name() string {
	return fmt.Sprintf("mcp_%s_%s", b.mcpName, b.tool.Name)
}

func (b *McpTool) Info() tools.ToolInfo {
	required := b.tool.InputSchema.Required
	if required == nil {
		required = make([]string, 0)
	}
	return tools.ToolInfo{
		Name:        fmt.Sprintf("mcp_%s_%s", b.mcpName, b.tool.Name),
		Description: b.tool.Description,
		Parameters:  b.tool.InputSchema.Properties,
		Required:    required,
	}
}

func isTransientTransportError(err error) bool {
	if err == nil {
		return false
	}
	e := strings.ToLower(err.Error())
	return strings.Contains(e, "broken pipe") ||
		strings.Contains(e, "connection reset") ||
		strings.Contains(e, "eof") ||
		strings.Contains(e, "use of closed network connection") ||
		strings.Contains(e, "transport error")
}

func getClientLock(name string) *sync.Mutex {
	if l, ok := mcpClientLocks.Get(name); ok && l != nil {
		return l
	}
	m := &sync.Mutex{}
	mcpClientLocks.Set(name, m)
	return m
}

func restartMCPClient(ctx context.Context, name string) (*client.Client, error) {
	cfg, ok := mcpClientConfigs.Get(name)
	if !ok {
		return nil, fmt.Errorf("no mcp config found for %s", name)
	}

	lock := getClientLock(name)
	lock.Lock()
	defer lock.Unlock()

	// Close any existing client
	if existing, ok := mcpClients.Take(name); ok && existing != nil {
		_ = existing.Close()
	}

	// Create and start a new client
	c, err := createMcpClient(cfg)
	if err != nil {
		updateMCPState(name, MCPStateError, err, nil, 0)
		return nil, err
	}

	// Use a long-lived context for starting the client process so it isn't tied
	// to a short-lived timeout. Initialization and subsequent requests will use
	// their own time-bounded contexts.
	if err := c.Start(context.Background()); err != nil {
		updateMCPState(name, MCPStateError, err, nil, 0)
		_ = c.Close()
		return nil, err
	}
	// Time-bound initialization
	rctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := c.Initialize(rctx, mcpInitRequest); err != nil {
		updateMCPState(name, MCPStateError, err, nil, 0)
		_ = c.Close()
		return nil, err
	}

	// Best-effort list tools for state/metrics; we do not rebuild wrappers here.
	toolCount := 0
	if res, err := c.ListTools(rctx, mcp.ListToolsRequest{}); err == nil {
		toolCount = len(res.Tools)
	}

	mcpClients.Set(name, c)
	updateMCPState(name, MCPStateConnected, nil, c, toolCount)
	return c, nil
}

func runTool(ctx context.Context, name, toolName string, input string) (tools.ToolResponse, error) {
	var args map[string]any
	if strings.TrimSpace(input) == "" {
		input = "{}"
	}
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return tools.NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}
	c, ok := mcpClients.Get(name)
	if !ok || c == nil {
		// Try to lazily (re)start the client once.
		var err error
		if c, err = restartMCPClient(ctx, name); err != nil {
			return tools.NewTextErrorResponse("mcp '" + name + "' not available: " + err.Error()), nil
		}
	}

	call := func(clientRef *client.Client) (*mcp.CallToolResult, error) {
		return clientRef.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      toolName,
				Arguments: args,
			},
		})
	}

	result, err := call(c)
	if err != nil && isTransientTransportError(err) {
		// Attempt a single restart + retry on transient transport errors
		slog.Warn("MCP transport error, attempting restart", "name", name, "error", err)
		if c2, rerr := restartMCPClient(ctx, name); rerr == nil {
			if result2, err2 := call(c2); err2 == nil {
				result = result2
				err = nil
			} else {
				err = err2
			}
		} else {
			// If restart fails keep original error context
			err = fmt.Errorf("%s; restart failed: %w", err.Error(), rerr)
		}
	}
	if err != nil {
		return tools.NewTextErrorResponse(err.Error()), nil
	}

	var output strings.Builder
	for _, v := range result.Content {
		if v, ok := v.(mcp.TextContent); ok {
			output.WriteString(v.Text)
		} else {
			_, _ = fmt.Fprintf(&output, "%v: ", v)
		}
	}

	return tools.NewTextResponse(output.String()), nil
}

func (b *McpTool) Run(ctx context.Context, params tools.ToolCall) (tools.ToolResponse, error) {
	sessionID, messageID := tools.GetContextValues(ctx)
	if sessionID == "" || messageID == "" {
		return tools.ToolResponse{}, fmt.Errorf("session ID and message ID are required for creating a new file")
	}
	permissionDescription := fmt.Sprintf("execute %s with the following parameters: %s", b.Info().Name, params.Input)
	p := b.permissions.Request(
		permission.CreatePermissionRequest{
			SessionID:   sessionID,
			ToolCallID:  params.ID,
			Path:        b.workingDir,
			ToolName:    b.Info().Name,
			Action:      "execute",
			Description: permissionDescription,
			Params:      params.Input,
		},
	)
	if !p {
		return tools.ToolResponse{}, permission.ErrorPermissionDenied
	}

	return runTool(ctx, b.mcpName, b.tool.Name, params.Input)
}

func getTools(ctx context.Context, name string, permissions permission.Service, c *client.Client, workingDir string) []tools.BaseTool {
	result, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		slog.Error("error listing tools", "error", err)
		updateMCPState(name, MCPStateError, err, nil, 0)
		c.Close()
		mcpClients.Del(name)
		return nil
	}
	mcpTools := make([]tools.BaseTool, 0, len(result.Tools))
	for _, tool := range result.Tools {
		mcpTools = append(mcpTools, &McpTool{
			mcpName:     name,
			tool:        tool,
			permissions: permissions,
			workingDir:  workingDir,
		})
	}
	return mcpTools
}

// SubscribeMCPEvents returns a channel for MCP events
func SubscribeMCPEvents(ctx context.Context) <-chan pubsub.Event[MCPEvent] {
	return mcpBroker.Subscribe(ctx)
}

// GetMCPStates returns the current state of all MCP clients
func GetMCPStates() map[string]MCPClientInfo {
	states := make(map[string]MCPClientInfo)
	for name, info := range mcpStates.Seq2() {
		states[name] = info
	}
	return states
}

// GetMCPState returns the state of a specific MCP client
func GetMCPState(name string) (MCPClientInfo, bool) {
	return mcpStates.Get(name)
}

// updateMCPState updates the state of an MCP client and publishes an event
func updateMCPState(name string, state MCPState, err error, client *client.Client, toolCount int) {
	info := MCPClientInfo{
		Name:      name,
		State:     state,
		Error:     err,
		Client:    client,
		ToolCount: toolCount,
	}
	if state == MCPStateConnected {
		info.ConnectedAt = time.Now()
	}
	mcpStates.Set(name, info)

	// Publish state change event
	mcpBroker.Publish(pubsub.UpdatedEvent, MCPEvent{
		Type:      MCPEventStateChanged,
		Name:      name,
		State:     state,
		Error:     err,
		ToolCount: toolCount,
	})
}

// CloseMCPClients closes all MCP clients. This should be called during application shutdown.
func CloseMCPClients() {
	for c := range mcpClients.Seq() {
		_ = c.Close()
	}
	mcpBroker.Shutdown()
}

var mcpInitRequest = mcp.InitializeRequest{
	Params: mcp.InitializeParams{
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
		ClientInfo: mcp.Implementation{
			Name:    "Crush",
			Version: version.Version,
		},
	},
}

func doGetMCPTools(ctx context.Context, permissions permission.Service, cfg *config.Config) []tools.BaseTool {
	var wg sync.WaitGroup
	result := csync.NewSlice[tools.BaseTool]()

	// Initialize states for all configured MCPs
	for name, m := range cfg.MCP {
		if m.Disabled {
			updateMCPState(name, MCPStateDisabled, nil, nil, 0)
			slog.Debug("skipping disabled mcp", "name", name)
			continue
		}

		// Set initial starting state
		updateMCPState(name, MCPStateStarting, nil, nil, 0)

		wg.Add(1)
		go func(name string, m config.MCPConfig) {
			defer func() {
				wg.Done()
				if r := recover(); r != nil {
					var err error
					switch v := r.(type) {
					case error:
						err = v
					case string:
						err = fmt.Errorf("panic: %s", v)
					default:
						err = fmt.Errorf("panic: %v", v)
					}
					updateMCPState(name, MCPStateError, err, nil, 0)
					slog.Error("panic in mcp client initialization", "error", err, "name", name)
				}
			}()

			// Save config for potential restarts on transient transport errors
			mcpClientConfigs.Set(name, m)

			c, err := createMcpClient(m)
			if err != nil {
				updateMCPState(name, MCPStateError, err, nil, 0)
				slog.Error("error creating mcp client", "error", err, "name", name)
				return
			}
			// Start with a non-cancelable context so the stdio process isn't torn down by a short timeout
			if err := c.Start(context.Background()); err != nil {
				updateMCPState(name, MCPStateError, err, nil, 0)
				slog.Error("error starting mcp client", "error", err, "name", name)
				_ = c.Close()
				return
			}
			// Bound initialization and tool discovery
			ictx, icancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer icancel()
			if _, err := c.Initialize(ictx, mcpInitRequest); err != nil {
				updateMCPState(name, MCPStateError, err, nil, 0)
				slog.Error("error initializing mcp client", "error", err, "name", name)
				_ = c.Close()
				return
			}

			slog.Info("Initialized mcp client", "name", name)
			mcpClients.Set(name, c)

			tools := getTools(ictx, name, permissions, c, cfg.WorkingDir())
			updateMCPState(name, MCPStateConnected, nil, c, len(tools))
			result.Append(tools...)
		}(name, m)
	}
	wg.Wait()
	return slices.Collect(result.Seq())
}

func createMcpClient(m config.MCPConfig) (*client.Client, error) {
	switch m.Type {
	case config.MCPStdio:
		return client.NewStdioMCPClientWithOptions(
			m.Command,
			m.ResolvedEnv(),
			m.Args,
			transport.WithCommandLogger(mcpLogger{}),
		)
	case config.MCPHttp:
		return client.NewStreamableHttpClient(
			m.URL,
			transport.WithHTTPHeaders(m.ResolvedHeaders()),
			transport.WithHTTPLogger(mcpLogger{}),
		)
	case config.MCPSse:
		return client.NewSSEMCPClient(
			m.URL,
			client.WithHeaders(m.ResolvedHeaders()),
			transport.WithSSELogger(mcpLogger{}),
		)
	default:
		return nil, fmt.Errorf("unsupported mcp type: %s", m.Type)
	}
}

// for MCP's clients.
type mcpLogger struct{}

func (l mcpLogger) Errorf(format string, v ...any) { slog.Error(fmt.Sprintf(format, v...)) }
func (l mcpLogger) Infof(format string, v ...any)  { slog.Info(fmt.Sprintf(format, v...)) }
