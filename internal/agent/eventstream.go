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

			if !isEventForSession(event, sessionID) {
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

func isEventForSession(e opencode.EventListResponse, sID string) bool {
	var properties struct {
		SessionID string `json:"sessionID"`
	}
	if raw := e.JSON.Properties.Raw(); raw != "" {
		_ = json.Unmarshal([]byte(raw), &properties)
	}

	return properties.SessionID == sID
}
