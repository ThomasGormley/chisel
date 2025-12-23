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

	var (
		prevToolHandled     string
		totalTokenInput     float64
		totalTokenOutput    float64
		totalTokenReasoning float64
		totalCost           float64
	)
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

			if event.Type != opencode.EventListResponseTypeMessagePartUpdated {
				if !sessionIDMatches(event, sessionID) {
					continue
				}
			} else {
				evt := event.AsUnion().(opencode.EventListResponseEventMessagePartUpdated)
				if evt.Properties.Part.SessionID != sessionID {
					continue
				}
			}

			switch event.Type {

			case opencode.EventListResponseTypePermissionUpdated:
				evt := event.AsUnion().(opencode.EventListResponseEventPermissionUpdated)
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

				switch part.Type {
				case opencode.PartTypeReasoning, opencode.PartTypeText:
					if evt.Properties.Delta != "" {
						print.Infof(os.Stdout, "%s", evt.Properties.Delta)
					}

				case opencode.PartTypeTool:
					handleToolPart(part, prevToolHandled)

				case opencode.PartTypeStepStart:
					handleStepStartPart(part)

				case opencode.PartTypeStepFinish:
					prevToolHandled = "" // reset the tool on step finish
					handleStepFinishPart(part, &totalTokenInput, &totalTokenOutput, &totalTokenReasoning, &totalCost)

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
				print.Errorf(os.Stdout, print.Wrap("❌ Session error: %s"), evt.Properties.Error.Name)

			case opencode.EventListResponseTypeLspClientDiagnostics:
				evt := event.AsUnion().(opencode.EventListResponseEventLspClientDiagnostics)

				print.Warningf(os.Stdout, print.Wrap("🚨 LSP Diagnostic at %s (Server: %s)"), evt.Properties.Path, evt.Properties.ServerID)

			case opencode.EventListResponseTypeSessionIdle:
				print.Success(os.Stdout, print.WrapTop("🏁 Done."))
				if totalTokenInput > 0 {
					print.Infof(os.Stdout, print.WrapBottom("  Input: %.0f tokens"), totalTokenInput)
				}
				if totalTokenOutput > 0 {
					print.Infof(os.Stdout, print.WrapBottom("  Output: %.0f tokens"), totalTokenOutput)
				}
				if totalTokenReasoning > 0 {
					print.Infof(os.Stdout, print.WrapBottom("  Reasoning: %.0f tokens"), totalTokenReasoning)
				}
				if totalCost > 0 {
					print.Infof(os.Stdout, print.WrapBottom("  Cost: $%.4f"), totalCost)
				}
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

func sessionIDMatches(event opencode.EventListResponse, sessionID string) bool {
	switch event.Type {
	case opencode.EventListResponseTypePermissionUpdated:
		evt := event.AsUnion().(opencode.EventListResponseEventPermissionUpdated)
		return evt.Properties.SessionID == sessionID
	case opencode.EventListResponseTypeSessionError:
		evt := event.AsUnion().(opencode.EventListResponseEventSessionError)
		return evt.Properties.SessionID == sessionID
	case opencode.EventListResponseTypeSessionIdle:
		evt := event.AsUnion().(opencode.EventListResponseEventSessionIdle)
		return evt.Properties.SessionID == sessionID
	default:
		return true
	}
}

func handleToolPart(part opencode.Part, prevHandledTool string) {
	if part.Tool != "" && part.Tool == prevHandledTool {
		return
	}
	state, ok := part.State.(opencode.ToolPartState)

	if part.Tool != "" {
		if ok && state.Title != "" {
			print.Notef(os.Stdout, print.Wrap("🔨 Tool: %s (%s)"), part.Tool, state.Title)
		}
		if ok && (state.Status == "completed" || state.Status == "error") {
			print.Infof(os.Stdout, "\n")
		}
	}
}

func handleStepStartPart(_ opencode.Part) {
	print.Notef(os.Stdout, print.Wrap("⚡ Step started"))
}

func handleStepFinishPart(
	part opencode.Part,
	totalTokenInput,
	totalTokenOutput,
	totalTokenReasoning,
	totalCost *float64,
) {
	print.Successf(os.Stdout, print.Wrap("✅ Step completed"))
	*totalCost += part.Cost
	if part.Tokens != nil {
		if tokens, ok := part.Tokens.(opencode.StepFinishPartTokens); ok {
			*totalTokenInput += tokens.Input
			*totalTokenOutput += tokens.Output
			*totalTokenReasoning += tokens.Reasoning
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
		if err, ok := part.Error.(opencode.PartRetryPartError); ok {
			print.Infof(os.Stdout, "  Error: %s", err.Name)
			print.Infof(os.Stdout, " - %s", err.Data.Message)
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
