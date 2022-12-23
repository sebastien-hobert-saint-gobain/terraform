package jsonformat

import (
	"fmt"
	"strings"

	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/terraform/internal/command/jsonplan"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/terminal"
)

type Plan struct {
	OutputChanges   map[string]jsonplan.Change        `json:"output_changes"`
	ResourceChanges []jsonplan.ResourceChange         `json:"resource_changes"`
	ResourceDrift   []jsonplan.ResourceChange         `json:"resource_drift"`
	ProviderSchemas map[string]*jsonprovider.Provider `json:"provider_schemas"`
}

type Renderer struct {
	Streams  *terminal.Streams
	Colorize *colorstring.Colorize
}

type JSONLogType string
type JSONLog map[string]interface{}

const (
	LogVersion         JSONLogType = "version"
	LogPlannedChange   JSONLogType = "planned_change"
	LogRefreshStart    JSONLogType = "refresh_start"
	LogRefreshComplete JSONLogType = "refresh_complete"
	LogApplyStart      JSONLogType = "apply_start"
	LogApplyComplete   JSONLogType = "apply_complete"
	LogChangeSummary   JSONLogType = "change_summary"
	LogOutputs         JSONLogType = "outputs"
)

func (r Renderer) RenderPlan(plan Plan) {
	// panic("not implemented")
	r.Streams.Printf("boop renderered plan!")
}

func (r Renderer) RenderLog(log JSONLog) {
	msg, ok := log["@message"].(string)
	if !ok {
		return
	}

	switch JSONLogType(log["type"].(string)) {
	case LogApplyStart, LogApplyComplete, LogRefreshStart, LogRefreshComplete:
		msg = fmt.Sprintf("[bold]%s[reset]", msg)
		r.Streams.Print(r.Colorize.Color(msg))
		r.Streams.Print("\n")
	case LogChangeSummary:
		// We will only render the text as green when it is an apply change
		// summary
		if strings.Contains(msg, "Plan") {
			s := strings.Split(msg, ":")
			msg = fmt.Sprintf("[bold]%s[reset]:%s", s[0], s[1])
		} else {
			msg = fmt.Sprintf("[bold][green]%s[reset][bold]", msg)
		}

		r.Streams.Print("\n\n")
		r.Streams.Print(r.Colorize.Color(msg))
		r.Streams.Print("\n\n")
	}
}
