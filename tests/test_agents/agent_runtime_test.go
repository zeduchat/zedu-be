package test_agents

import (
	"testing"

	agentruntime "github.com/hngprojects/telex_be/services/agents"
)

func TestToolDefinitionsExposeRuntimeActions(t *testing.T) {
	definitions := agentruntime.ToolDefinitions()
	if len(definitions) != 7 {
		t.Fatalf("expected 7 tool definitions, got %d", len(definitions))
	}

	names := make(map[string]bool, len(definitions))
	for _, definition := range definitions {
		names[definition.Function.Name] = true
	}

	expected := []string{
		"capitalize_text",
		"search_organisation_members",
		"list_channel_members",
		"add_user_to_channel",
		"remove_user_from_channel",
		"move_user_between_channels",
		"export_channel_members",
	}

	for _, name := range expected {
		if !names[name] {
			t.Fatalf("missing tool definition: %s", name)
		}
	}
}

func TestToolRegistryValidatesUnknownAndMissingContext(t *testing.T) {
	registry := agentruntime.NewToolRegistry()

	if _, err := registry.Execute(agentruntime.ToolExecutionContext{}, "unknown_tool", nil); err == nil {
		t.Fatal("expected unknown tool error")
	}

	if _, err := registry.Execute(agentruntime.ToolExecutionContext{}, "list_channel_members", map[string]any{
		"channel_name": "general",
	}); err == nil {
		t.Fatal("expected database/context validation error")
	}
}

func TestCapitalizeToolDoesNotNeedDatabase(t *testing.T) {
	registry := agentruntime.NewToolRegistry()

	result, err := registry.Execute(agentruntime.ToolExecutionContext{}, "capitalize_text", map[string]any{
		"text": "Zedu",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", result)
	}

	if payload["capitalized"] != "ZEDU" {
		t.Fatalf("unexpected capitalized value: %#v", payload["capitalized"])
	}
}
