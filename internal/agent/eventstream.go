package agent

import (
	"context"
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

func ListenForEvents(ctx context.Context, client *opencode.Client) error {
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
					if part.Tool != "" {

						print.Notef(os.Stdout, print.Wrap("🔨 Tool: %s"), part.Tool)

						// Safely handle state as a map for more flexible property checking
						state, ok := part.State.(opencode.ToolPartState)
						if ok && state.Title != "" {
							print.Infof(os.Stdout, " (%s)", state.Title)
							if state.Status == "completed" || state.Status == "error" {
								print.Infof(os.Stdout, "\n")
							}
						}
					}
				}

				if part.URL != "" {
					print.Infof(os.Stdout, print.Wrap("🌐 Fetching: %s"), part.URL)
				}

			case opencode.EventListResponseTypeFileEdited:
				evt := event.AsUnion().(opencode.EventListResponseEventFileEdited)
				print.Successf(os.Stdout, print.Wrap("💾 Edited: %s"), evt.Properties.File)

			case opencode.EventListResponseTypeTodoUpdated:
				evt := event.AsUnion().(opencode.EventListResponseEventTodoUpdated)
				for _, todo := range evt.Properties.Todos {
					switch todo.Status {
					case "completed":
						print.Successf(os.Stdout, print.Wrap("✅ %s"), todo.Content)
					case "in_progress":
						print.Warningf(os.Stdout, print.Wrap("⏳ %s"), todo.Content)
					}
				}

			case opencode.EventListResponseTypeSessionError:
				evt := event.AsUnion().(opencode.EventListResponseEventSessionError)
				print.Errorf(os.Stdout, print.Wrap("❌ Session error: %s"), evt.Properties.Error.Name)

			case opencode.EventListResponseTypeLspClientDiagnostics:
				evt := event.AsUnion().(opencode.EventListResponseEventLspClientDiagnostics)
				print.Warningf(os.Stdout, print.Wrap("🚨 LSP Diagnostic at %s (Server: %s)"), evt.Properties.Path, evt.Properties.ServerID)

			case opencode.EventListResponseTypeSessionIdle:
				print.Success(os.Stdout, print.Wrap("🏁 Done."))
			}
		}
	}
}
