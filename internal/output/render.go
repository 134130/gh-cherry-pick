package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/134130/gh-cherry-pick/internal/app"
)

type PlanOptions struct {
	Format  Format
	NoColor bool
}

func RenderPlan(w io.Writer, plan app.Plan, opts PlanOptions) error {
	if opts.Format == FormatJSON {
		return renderPlanJSON(w, plan)
	}
	return renderPlanHuman(w, plan)
}

func RenderRunResult(w io.Writer, result app.RunResult, format Format) error {
	if format == FormatJSON {
		return json.NewEncoder(w).Encode(runResultJSON(result))
	}
	if _, err := fmt.Fprintf(w, "Applied PR #%d using %s\n", result.Plan.PR.Number, result.Plan.Method); err != nil {
		return err
	}
	if len(result.CommitSHAs) == 0 {
		_, err := fmt.Fprintln(w, "No new commits were created.")
		return err
	}
	if _, err := fmt.Fprintln(w, "Commits:"); err != nil {
		return err
	}
	for _, sha := range result.CommitSHAs {
		if _, err := fmt.Fprintf(w, "  %s\n", sha); err != nil {
			return err
		}
	}
	return nil
}

func renderPlanHuman(w io.Writer, plan app.Plan) error {
	if _, err := fmt.Fprintf(w, "PR #%d: %s\n", plan.PR.Number, plan.PR.Title); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Destination: %s @ %s\n", plan.Destination.Branch, shortSHA(plan.Destination.Head)); err != nil {
		return err
	}
	method := plan.Method
	if method == "" {
		method = plan.RequestedMethod
	}
	methodLine := string(method)
	if plan.RequestedMethod != "" && plan.RequestedMethod != method {
		methodLine = fmt.Sprintf("%s (requested %s)", plan.Method, plan.RequestedMethod)
	}
	if _, err := fmt.Fprintf(w, "Method: %s\n", methodLine); err != nil {
		return err
	}
	if plan.MergeShape != "" {
		if _, err := fmt.Fprintf(w, "Merge shape: %s\n", plan.MergeShape); err != nil {
			return err
		}
	}
	if plan.Blocked() {
		if _, err := fmt.Fprintf(w, "Blocked: %s\n", plan.BlockedReason); err != nil {
			return err
		}
	}
	if len(plan.Commands) == 0 {
		_, err := fmt.Fprintln(w, "Commands: none")
		return err
	}
	if _, err := fmt.Fprintln(w, "Commands:"); err != nil {
		return err
	}
	for _, cmd := range plan.Commands {
		if _, err := fmt.Fprintf(w, "  %s\n", commandString(cmd)); err != nil {
			return err
		}
	}
	return nil
}

func renderPlanJSON(w io.Writer, plan app.Plan) error {
	return json.NewEncoder(w).Encode(planJSON(plan))
}

type commandJSON struct {
	Program string   `json:"program"`
	Args    []string `json:"args"`
}

type planJSONShape struct {
	PR struct {
		Number int    `json:"number"`
		State  string `json:"state"`
		Title  string `json:"title"`
		URL    string `json:"url,omitempty"`
	} `json:"pr"`
	Destination struct {
		Branch string `json:"branch"`
		Head   string `json:"head"`
	} `json:"destination"`
	RequestedMethod string        `json:"requestedMethod"`
	Method          string        `json:"method"`
	MergeShape      string        `json:"mergeShape,omitempty"`
	BlockedReason   string        `json:"blockedReason,omitempty"`
	Commands        []commandJSON `json:"commands"`
}

type runResultJSONShape struct {
	Plan       planJSONShape `json:"plan"`
	CommitSHAs []string      `json:"commitShas"`
}

func planJSON(plan app.Plan) planJSONShape {
	var out planJSONShape
	out.PR.Number = plan.PR.Number
	out.PR.State = string(plan.PR.State)
	out.PR.Title = plan.PR.Title
	out.PR.URL = plan.PR.Url
	out.Destination.Branch = plan.Destination.Branch
	out.Destination.Head = plan.Destination.Head
	out.RequestedMethod = string(plan.RequestedMethod)
	if plan.Method == "" {
		out.Method = string(plan.RequestedMethod)
	} else {
		out.Method = string(plan.Method)
	}
	out.MergeShape = plan.MergeShape
	out.BlockedReason = plan.BlockedReason
	out.Commands = make([]commandJSON, 0, len(plan.Commands))
	for _, cmd := range plan.Commands {
		out.Commands = append(out.Commands, commandJSON{Program: cmd.Program, Args: cmd.Args})
	}
	return out
}

func runResultJSON(result app.RunResult) runResultJSONShape {
	return runResultJSONShape{
		Plan:       planJSON(result.Plan),
		CommitSHAs: result.CommitSHAs,
	}
}

func commandString(cmd app.Command) string {
	if len(cmd.Args) == 0 {
		return cmd.Program
	}
	return cmd.Program + " " + strings.Join(cmd.Args, " ")
}

func shortSHA(sha string) string {
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}
