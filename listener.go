package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/sst/opencode-sdk-go"
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

func listen(ctx context.Context, client *opencode.Client) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Event stream goroutine panicked: %v\n", r)
		}
	}()

	stream := client.Event.ListStreaming(ctx, opencode.EventListParams{})
	defer stream.Close()

	lastToolCallID := ""
	lastToolTitle := ""
	lastTodoStatus := make(map[string]string)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			if !stream.Next() {
				return
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
				if part.Type == opencode.PartTypeTool && part.Tool != "" {
					if part.ID != lastToolCallID {
						lastToolCallID = part.ID
						lastToolTitle = ""
						fmt.Printf("🔨 Tool: %s\n", part.Tool)
					}

					// Safely handle state as a map for more flexible property checking
					state, ok := part.State.(opencode.ToolPartState)
					if ok {
						if state.Title != "" && state.Title != lastToolTitle {
							lastToolTitle = state.Title
							fmt.Printf(" (%s)\n", state.Title)
						}
						if state.Status == "completed" || state.Status == "error" {
							fmt.Println()
						}
					}
				}

			case opencode.EventListResponseTypeFileEdited:
				evt := event.AsUnion().(opencode.EventListResponseEventFileEdited)
				fmt.Printf("💾 Edited: %s\n", evt.Properties.File)

			case opencode.EventListResponseTypeTodoUpdated:
				evt := event.AsUnion().(opencode.EventListResponseEventTodoUpdated)
				for _, todo := range evt.Properties.Todos {
					prevStatus := lastTodoStatus[todo.ID]
					if todo.Status != prevStatus {
						lastTodoStatus[todo.ID] = todo.Status
						if todo.Status == "completed" {
							fmt.Printf("✅ %s\n", todo.Content)
						} else if todo.Status == "in_progress" {
							fmt.Printf("⏳ %s\n", todo.Content)
						}
					}
				}

			case opencode.EventListResponseTypeSessionError:
				fmt.Printf("❌ Session error occurred\n")

			case opencode.EventListResponseTypeSessionIdle:
				fmt.Println("🏁 Done.")
			}
		}
	}
}
