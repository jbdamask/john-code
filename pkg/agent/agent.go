package agent

import (
	"context"
	"fmt"
    "os"
    "strings"

	"github.com/jbdamask/john-code/pkg/config"
	"github.com/jbdamask/john-code/pkg/history"
	"github.com/jbdamask/john-code/pkg/llm"
	"github.com/jbdamask/john-code/pkg/tools"
	"github.com/jbdamask/john-code/pkg/ui"
)

type Agent struct {
	cfg     *config.Config
	ui      *ui.UI
	tools   *tools.Registry
	client  llm.Client
	history []llm.Message
    session *history.SessionManager
}

func New(cfg *config.Config, ui *ui.UI) *Agent {
    registry := tools.NewRegistry()
    registry.Register(tools.NewBashTool())
    registry.Register(&tools.ReadTool{})
    registry.Register(&tools.WriteTool{})
    registry.Register(&tools.EditTool{})
    registry.Register(&tools.GlobTool{})
    registry.Register(tools.NewTodoWriteTool())
    registry.Register(&tools.GrepTool{})
    
    registry.Register(tools.NewWebSearchTool())
    registry.Register(tools.NewWebFetchTool())
    registry.Register(tools.NewAskUserQuestionTool(ui))
    registry.Register(&tools.NotebookEditTool{})
    registry.Register(&tools.BashOutputTool{})
    registry.Register(&tools.KillShellTool{})

    // Task Tool - Recursive Agent
    // We need to define the runner closure
    // Note: This creates a circular dependency concept if we try to use 'New' directly? 
    // No, we are inside 'New', so we can't use 'New' easily without infinite recursion if we aren't careful about compilation,
    // but runtime is fine.
    // Actually, we need to extract NewAgent logic or use a method on Agent.
    
    // For now, let's delay the runner creation or use a method.
    // But we need to register the tool NOW.
    
    // We can pass a placeholder and set it later? No, registry needs initialized tool.
    // We can make a closure that calls a package level function? No.
    
    // Let's solve this by passing the factory function to New? 
    // Or just creating the tool with a closure that refers to a function we define here.
    
    taskRunner := func(ctx context.Context, task string) (string, error) {
        // Create a new agent instance for the subtask
        // We need to use the same config and UI (maybe indented UI?)
        // For MVP, share UI.
        
        // We can't call New() here easily if it's in the same package but we are in New...
        // Go allows recursive calls.
        
        subAgent := New(cfg, ui)
        
        // Override history to start with the task
        subAgent.history = []llm.Message{
            {
                Role: llm.RoleSystem,
                Content: "You are a sub-agent working on a specific task: " + task,
            },
            {
                Role: llm.RoleUser,
                Content: "Please perform the task: " + task,
            },
        }
        
        // Run the agent loop until it finishes? 
        // Our current Agent.Run() is an interactive loop reading from Stdin.
        // We need a non-interactive Run mode (RunTask).
        
        return subAgent.RunTask(ctx)
    }
    
    registry.Register(tools.NewTaskTool(taskRunner))

    // Use real client if configured
    var client llm.Client
    if cfg.APIKey != "dummy" && cfg.APIKey != "" {
        client = llm.NewAnthropicClient(cfg.APIKey)
    } else {
        client = llm.NewMockClient()
    }

    // Initialize Session Manager
    // We need CWD
    // Since we use NewBashTool which gets CWD, we should match.
    // But NewBashTool is internal.
    // Let's just use "." and let SessionManager expand it.
    // Actually SessionManager does string replacement, so we should get absolute path.
    
    // We'll initialize it in New, logging error if fails but not crashing?
    
    // We can't get error from New easily without changing signature.
    // Let's assume we can get CWD.
    
	return &Agent{
		cfg:    cfg,
		ui:     ui,
		tools:  registry,
		client: client,
        session: nil, // Will init in Run
		history: []llm.Message{
            {
                Role: llm.RoleSystem,
                Content: SystemPrompt,
            },
        },
	}
}

func (a *Agent) Run() error {
	a.ui.Print("John Code initialized. Type 'exit' or 'quit' to stop.")

    cwd, err := os.Getwd()
    if err == nil {
        sm, err := history.NewSessionManager(cwd)
        if err != nil {
            a.ui.Print(fmt.Sprintf("Warning: Failed to initialize session manager: %v", err))
        } else {
            a.session = sm
            a.ui.Print(fmt.Sprintf("Session ID: %s", sm.SessionID))
        }
    }

	for {
		input := a.ui.Prompt("> ")
		if input == "exit" || input == "quit" {
			break
		}
        if input == "" {
            continue
        }

		// Parse for images in input
        var images []string
        cleanInput := input
        
        // Very basic regex-like parsing for [Image: path]
        for {
            start := strings.Index(cleanInput, "[Image: ")
            if start == -1 { break }
            end := strings.Index(cleanInput[start:], "]")
            if end == -1 { break }
            
            fullTag := cleanInput[start : start+end+1]
            path := strings.TrimPrefix(fullTag, "[Image: ")
            path = strings.TrimSuffix(path, "]")
            
            images = append(images, strings.TrimSpace(path))
            
            // Remove tag from text
            cleanInput = strings.Replace(cleanInput, fullTag, "", 1)
        }
        cleanInput = strings.TrimSpace(cleanInput)

		// Add user message to history
        userMsg := llm.Message{
			Role:    llm.RoleUser,
			Content: cleanInput,
            Images:  images,
		}
		a.history = append(a.history, userMsg)
        
        if a.session != nil {
            if err := a.session.Append(llm.RoleUser, userMsg); err != nil {
                a.ui.Print(fmt.Sprintf("Warning: Failed to log user message: %v", err))
            }
        }

		// Run the LLM loop (handling tool calls)
		if err := a.processTurn(); err != nil {
			a.ui.Print(fmt.Sprintf("Error: %v", err))
		}
	}
	return nil
}

func (a *Agent) RunTask(ctx context.Context) (string, error) {
    // Run the agent loop non-interactively until it produces a final answer or finishes.
    // For the agent to "finish", it needs to decide it is done. 
    // Standard tool-use agents usually stop when they output text without tool calls?
    // Or we can give it a "TaskDone" tool?
    // For now, let's say if it outputs text without tool calls, that's the result.
    
    // We'll run up to N turns.
    
    // But wait, processTurn runs up to 10 tool interactions in a loop.
    // If processTurn returns nil (no tool calls), it means it has produced a final response text.
    
    err := a.processTurn()
    if err != nil {
        return "", err
    }
    
    // The last message in history (from Assistant) is the result
    if len(a.history) > 0 {
        last := a.history[len(a.history)-1]
        if last.Role == llm.RoleAssistant {
            return last.Content, nil
        }
    }
    return "Task completed with no output", nil
}

func (a *Agent) processTurn() error {
    ctx := context.Background()
    
    // Max turns to prevent infinite loops
    for i := 0; i < 10; i++ {
        // Prepare tools for the API
        var apiTools []interface{}
        for _, t := range a.tools.List() {
             apiTools = append(apiTools, t)
        }

        ch := make(chan string)
        var resp *llm.Message
        var genErr error
        
        go func() {
            defer close(ch)
            resp, genErr = a.client.GenerateStream(ctx, a.history, apiTools, ch)
        }()

        a.ui.DisplayStream(ch)
        
        if genErr != nil {
            return genErr
        }
        if resp == nil {
            return fmt.Errorf("generation produced no response")
        }

        a.history = append(a.history, *resp)
        if a.session != nil {
            if err := a.session.Append(llm.RoleAssistant, *resp); err != nil {
                a.ui.Print(fmt.Sprintf("Warning: Failed to log assistant message: %v", err))
            }
        }

        // If no tool calls, we're done with this turn (waiting for user input)
        if len(resp.ToolCalls) == 0 {
            return nil
        }

        // Handle tool calls
        for _, tc := range resp.ToolCalls {
            a.ui.Print(fmt.Sprintf("Running tool: %s", tc.Name))
            
            tool, found := a.tools.Get(tc.Name)
            var result string
            var err error
            
            if !found {
                result = fmt.Sprintf("Error: Tool %s not found", tc.Name)
            } else {
                result, err = tool.Execute(ctx, tc.Args)
                if err != nil {
                    result = fmt.Sprintf("Error executing tool: %v", err)
                }
            }
            
            // Append tool result to history
            toolMsg := llm.Message{
                Role: llm.RoleTool,
                ToolResult: &llm.ToolResult{
                    ToolCallID: tc.ID,
                    Content: result,
                },
            }
            a.history = append(a.history, toolMsg)
            
            if a.session != nil {
                if err := a.session.Append(llm.RoleTool, toolMsg); err != nil {
                    a.ui.Print(fmt.Sprintf("Warning: Failed to log tool result: %v", err))
                }
            }
        }
        // Loop continues to send tool results back to LLM
    }
    
    return fmt.Errorf("max turns reached")
}
