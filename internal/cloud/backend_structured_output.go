package cloud

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/logging"
)

func (b *Cloud) renderPlan(ctx context.Context, run *tfe.Run) error {
	logs, err := b.client.Plans.Logs(ctx, run.Plan.ID)
	if err != nil {
		return err
	}

	reader := bufio.NewReaderSize(logs, 64*1024)
	deferredLogs := []jsonformat.JSONLog{}
	planStarted := false

	if b.CLI != nil {
		for next := true; next; {
			var l, line []byte
			var err error

			for isPrefix := true; isPrefix; {
				l, isPrefix, err = reader.ReadLine()
				if err != nil {
					if err != io.EOF {
						return generalError("Failed to read logs", err)
					}
					next = false
				}

				line = append(line, l...)
			}

			if next || len(line) > 0 {
				log := make(jsonformat.JSONLog)
				if err := json.Unmarshal(line, &log); err != nil || log == nil {
					// If we can not parse the line as JSON, we will simply
					// print the line.
					b.CLI.Output(b.Colorize().Color(string(line)))
					continue
				}

				logType := jsonformat.JSONLogType(log["type"].(string))
				// We'll defer any log during a plan operation that is not
				// plan output.
				if planStarted && logType != jsonformat.LogPlannedChange {
					deferredLogs = append(deferredLogs, log)
					continue
				}

				// If the log is plan output, we will indicate the plan has
				// started and continue the loop.
				if logType == jsonformat.LogPlannedChange {
					planStarted = true
					continue
				}

				if b.renderer != nil {
					// Otherwise, we will print the log
					b.renderer.RenderLog(log)
				}
			}
		}

	}

	// Get a refreshed view of the workspace and
	// check if structured run output is enabled
	ws, err := b.client.Workspaces.ReadByID(ctx, run.Workspace.ID)
	if err != nil {
		return err
	}

	if ws.StructuredRunOutputEnabled && b.renderer != nil {
		// Fetch the redacted plan
		redacted, err := b.readRedactedPlan(ctx, run.Plan.ID)
		if err != nil {
			return err
		}

		// Render plan output
		b.renderer.RenderPlan(*redacted)

		for _, log := range deferredLogs {
			b.renderer.RenderLog(log)
		}
	}

	return nil
}

func (b *Cloud) renderApply(ctx context.Context, run *tfe.Run) error {
	logs, err := b.client.Applies.Logs(ctx, run.Apply.ID)
	if err != nil {
		return err
	}

	reader := bufio.NewReaderSize(logs, 64*1024)
	if b.CLI != nil {
		skip := 0
		for next := true; next; {
			var l, line []byte
			var err error

			for isPrefix := true; isPrefix; {
				l, isPrefix, err = reader.ReadLine()
				if err != nil {
					if err != io.EOF {
						return generalError("Failed to read logs", err)
					}
					next = false
				}

				line = append(line, l...)
			}

			// Apply logs show the same Terraform info logs as shown in the plan logs
			// (which are the first three lines), we therefore skip to prevent duplicate output
			if skip < 3 {
				skip++
				continue
			}

			if next || len(line) > 0 {
				log := make(jsonformat.JSONLog)
				if err := json.Unmarshal(line, &log); err != nil || log == nil {
					// If we can not parse the line as JSON, we will simply
					// print the line.
					b.CLI.Output(b.Colorize().Color(string(line)))
					continue
				}

				if b.renderer != nil {
					// Otherwise, we will print the log
					b.renderer.RenderLog(log)
				}
			}
		}
	}

	return nil
}

func (b *Cloud) readRedactedPlan(ctx context.Context, planID string) (*jsonformat.Plan, error) {
	client := retryablehttp.NewClient()
	client.RetryMax = 10
	client.Logger = logging.HCLogger()

	u := fmt.Sprintf("https://%s/api/v2/plans/%s/json-output-redacted",
		url.QueryEscape(b.hostname),
		url.QueryEscape(planID),
	)

	req, err := retryablehttp.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	token, err := b.token()
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	p := &jsonformat.Plan{}
	resp, err := client.Do(req)
	if err != nil {
		return p, err
	}

	if err := json.NewDecoder(resp.Body).Decode(p); err != nil {
		return nil, err
	}

	return p, nil
}
