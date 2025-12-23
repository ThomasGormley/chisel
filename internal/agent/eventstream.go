package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sst/opencode-sdk-go"
	"github.com/thomasgormley/chisel/internal/print"
)

type DialogResponse struct {
	Button  string
	Success bool
}

func permissionDialog(title, message string) DialogResponse {
	script := fmt.Sprintf(`display dialog "%s" with title "%s" buttons {"Reject", "Allow Once", "Always"} default button "Always" cancel button "Reject" with icon caution`, message, title)
	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return DialogResponse{Success: false}
	}

	outStr := string(output)
	if strings.Contains(outStr, "Always") {
		return DialogResponse{Button: "Always", Success: true}
	} else if strings.Contains(outStr, "Allow Once") {
		return DialogResponse{Button: "Allow Once", Success: true}
	}

	return DialogResponse{Button: "Reject", Success: true}
}

func ListenForEvents(ctx context.Context, client *opencode.Client, sessionID string) error {
	stream := client.Event.ListStreaming(ctx, opencode.EventListParams{})
	defer stream.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if !stream.Next() {
				return stream.Err()
			}

			event := stream.Current()

			if !shouldListen(event, sessionID) {
				continue
			}

			switch event.Type {

			case opencode.EventListResponseTypePermissionUpdated:
				evt := event.AsUnion().(opencode.EventListResponseEventPermissionUpdated)
				if evt.Properties.SessionID != sessionID {
					continue
				}
				dialogResult := permissionDialog("Chisel Permission", "Agent is requesting permission to perform an action.")

				response := opencode.SessionPermissionRespondParamsResponseReject
				if dialogResult.Success {
					switch dialogResult.Button {
					case "Always":
						response = opencode.SessionPermissionRespondParamsResponseAlways
					case "Allow Once":
						response = opencode.SessionPermissionRespondParamsResponseOnce
					}
				}

				client.Session.Permissions.Respond(ctx, evt.Properties.SessionID, evt.Properties.ID, opencode.SessionPermissionRespondParams{
					Response: opencode.F(response),
				})

			case opencode.EventListResponseTypeMessagePartUpdated:
				evt := event.AsUnion().(opencode.EventListResponseEventMessagePartUpdated)
				part := evt.Properties.Part
				if part.SessionID != sessionID {
					continue
				}

				switch part.Type {
				case opencode.PartTypeReasoning, opencode.PartTypeText:
					if evt.Properties.Delta != "" {
						print.Infof(os.Stdout, "%s", evt.Properties.Delta)
					}

				case opencode.PartTypeTool:
					handleToolPart(part)

				case opencode.PartTypeStepStart:
					handleStepStartPart(part)

				case opencode.PartTypeStepFinish:
					handleStepFinishPart(part)

				case opencode.PartTypeSnapshot:
					handleSnapshotPart(part)

				case opencode.PartTypePatch:
					handlePatchPart(part)

				case opencode.PartTypeAgent:
					handleAgentPart(part)

				case opencode.PartTypeRetry:
					handleRetryPart(part)

				case opencode.PartTypeFile:
					handleFilePart(part)
				}

				if part.URL != "" {
					print.Infof(os.Stdout, print.Wrap("🌐 Fetching: %s"), part.URL)
				}

			case opencode.EventListResponseTypeFileEdited:
				evt := event.AsUnion().(opencode.EventListResponseEventFileEdited)
				print.Successf(os.Stdout, print.Wrap("💾 Edited: %s"), evt.Properties.File)

			case opencode.EventListResponseTypeSessionError:
				evt := event.AsUnion().(opencode.EventListResponseEventSessionError)
				if evt.Properties.SessionID != sessionID {
					continue
				}
				print.Errorf(os.Stdout, print.Wrap("❌ Session error: %s"), evt.Properties.Error.Name)

			case opencode.EventListResponseTypeLspClientDiagnostics:
				evt := event.AsUnion().(opencode.EventListResponseEventLspClientDiagnostics)

				print.Warningf(os.Stdout, print.Wrap("🚨 LSP Diagnostic at %s (Server: %s)"), evt.Properties.Path, evt.Properties.ServerID)

			case opencode.EventListResponseTypeSessionIdle:
				evt := event.AsUnion().(opencode.EventListResponseEventSessionIdle)
				if evt.Properties.SessionID != sessionID {
					continue
				}
				print.Success(os.Stdout, print.Wrap("🏁 Done."))
			}
		}
	}
}

func shouldListen(e opencode.EventListResponse, sID string) bool {
	var properties struct {
		SessionID string `json:"sessionID"`
	}
	if raw := e.JSON.Properties.Raw(); raw != "" {
		_ = json.Unmarshal([]byte(raw), &properties)
	}

	if properties.SessionID == "" {
		return true
	}

	return properties.SessionID == sID
}

func handleToolPart(part opencode.Part) {
	state, ok := part.State.(opencode.ToolPartState)

	if part.Tool != "" {
		if ok && state.Title != "" {
			print.Notef(os.Stdout, print.Wrap("🔨 Tool: %s (%s)"), part.Tool, state.Title)
		} else {
			print.Notef(os.Stdout, print.Wrap("🔨 Tool: %s"), part.Tool)
		}
		if ok && (state.Status == "completed" || state.Status == "error") {
			print.Infof(os.Stdout, "\n")
		}
	}
}

func handleStepStartPart(part opencode.Part) {
	print.Notef(os.Stdout, print.Wrap("⚡ Step started"))
	if part.Snapshot != "" {
		print.Infof(os.Stdout, print.Wrap("  Snapshot: %s"), part.Snapshot)
	}
}

func handleStepFinishPart(part opencode.Part) {
	print.Successf(os.Stdout, print.Wrap("✓ Step completed"))
	if part.Reason != "" {
		print.Infof(os.Stdout, "  Reason: %s\n", part.Reason)
	}
	if part.Tokens != nil {
		if tokens, ok := part.Tokens.(map[string]interface{}); ok {
			if input, ok := tokens["input"].(float64); ok && input > 0 {
				print.Infof(os.Stdout, "  Tokens: input=%.0f", input)
			}
			if output, ok := tokens["output"].(float64); ok && output > 0 {
				print.Infof(os.Stdout, " output=%.0f", output)
			}
			if reasoning, ok := tokens["reasoning"].(float64); ok && reasoning > 0 {
				print.Infof(os.Stdout, " reasoning=%.0f", reasoning)
			}
			print.Infof(os.Stdout, "\n")
		}
	}
	if part.Cost > 0 {
		print.Infof(os.Stdout, "  Cost: $%.4f\n", part.Cost)
	}
}

func handleSnapshotPart(part opencode.Part) {
	print.Notef(os.Stdout, print.Wrap("📸 Snapshot: %s"), part.Snapshot)
}

func handlePatchPart(part opencode.Part) {
	if part.Files != nil {
		if files, ok := part.Files.([]interface{}); ok && len(files) > 0 {
			print.Notef(os.Stdout, print.Wrap("📦 Patching %d files"), len(files))
			hash := part.Hash
			if hash != "" {
				print.Infof(os.Stdout, "  Hash: %s\n", hash)
			}
		}
	}
}

func handleAgentPart(part opencode.Part) {
	print.Notef(os.Stdout, print.Wrap("🤖 Agent: %s"), part.Name)
	if part.Source != nil {
		if source, ok := part.Source.(opencode.AgentPartSource); ok {
			print.Infof(os.Stdout, "  Source: %s (chars %d-%d)\n", source.Value, source.Start, source.End)
		}
	}
}

func handleRetryPart(part opencode.Part) {
	print.Warningf(os.Stdout, print.Wrap("🔄 Retry attempt %.0f"), part.Attempt)
	if part.Error != nil {
		if err, ok := part.Error.(map[string]interface{}); ok {
			if name, ok := err["name"].(string); ok {
				print.Infof(os.Stdout, "  Error: %s", name)
			}
			if data, ok := err["data"].(map[string]interface{}); ok {
				if msg, ok := data["message"].(string); ok {
					print.Infof(os.Stdout, " - %s", msg)
				}
			}
			print.Infof(os.Stdout, "\n")
		}
	}
}

func handleFilePart(part opencode.Part) {
	if part.Filename != "" {
		print.Notef(os.Stdout, print.Wrap("📄 File: %s"), part.Filename)
	} else if part.URL != "" {
		print.Notef(os.Stdout, print.Wrap("📄 Downloading: %s"), part.URL)
	}
	if part.Mime != "" {
		print.Infof(os.Stdout, "  Type: %s\n", part.Mime)
	}
}
