package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/134130/gh-cherry-pick/internal/app"
)

type PlanOptions struct {
	Format  Format
	NoColor bool
	Verbose bool
}

type RunResultOptions struct {
	Format  Format
	Verbose bool
}

type ErrorOptions struct {
	Verbose bool
}

func RenderPlan(w io.Writer, plan app.Plan, opts PlanOptions) error {
	if opts.Format == FormatJSON {
		return renderPlanJSON(w, plan)
	}
	return renderPlanHuman(w, plan, opts)
}

func RenderRunResult(w io.Writer, result app.RunResult, opts RunResultOptions) error {
	if opts.Format == FormatJSON {
		return json.NewEncoder(w).Encode(runResultJSON(result))
	}
	if _, err := fmt.Fprintf(w, "Applied PR #%d using %s.\n", result.Plan.PR.Number, result.Plan.Method); err != nil {
		return err
	}
	if len(result.CommitSHAs) == 0 {
		_, err := fmt.Fprintln(w, "No new commits were created.")
		return err
	}
	if _, err := fmt.Fprintf(w, "\nCreated %d %s:\n", len(result.CommitSHAs), commitLabel(len(result.CommitSHAs))); err != nil {
		return err
	}
	for _, sha := range result.CommitSHAs {
		if _, err := fmt.Fprintf(w, "  %s\n", shortSHA(sha)); err != nil {
			return err
		}
	}
	return nil
}

func RenderExecutionError(w io.Writer, err error, opts ErrorOptions) (bool, error) {
	var failure *app.ApplyFailure
	if !errors.As(err, &failure) {
		return false, nil
	}
	if _, renderErr := fmt.Fprintln(w, failure.Summary); renderErr != nil {
		return true, renderErr
	}
	if len(failure.ConflictedFiles) > 0 {
		if _, renderErr := fmt.Fprintln(w, "\nConflicted files:"); renderErr != nil {
			return true, renderErr
		}
		for _, file := range failure.ConflictedFiles {
			if _, renderErr := fmt.Fprintf(w, "  %s\n", file); renderErr != nil {
				return true, renderErr
			}
		}
	}
	if failure.Guidance != "" {
		if _, renderErr := fmt.Fprintf(w, "\n%s\n", failure.Guidance); renderErr != nil {
			return true, renderErr
		}
	}
	if opts.Verbose && failure.Detail != "" {
		if _, renderErr := fmt.Fprintf(w, "\nDetails:\n  %s\n", failure.Detail); renderErr != nil {
			return true, renderErr
		}
	}
	return true, nil
}

func renderPlanHuman(w io.Writer, plan app.Plan, opts PlanOptions) error {
	if plan.Blocked() {
		return renderBlockedPlanHuman(w, plan)
	}
	if _, err := fmt.Fprintf(w, "Apply PR #%d: %s\n", plan.PR.Number, plan.PR.Title); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Destination: %s @ %s\n", plan.Destination.Branch, shortSHA(plan.Destination.Head)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Strategy: %s\n", strategyLine(plan)); err != nil {
		return err
	}
	if !opts.Verbose || len(plan.Commands) == 0 {
		return nil
	}
	if _, err := fmt.Fprintln(w, "\nCommands:"); err != nil {
		return err
	}
	for _, cmd := range plan.Commands {
		if _, err := fmt.Fprintf(w, "  %s\n", commandString(cmd)); err != nil {
			return err
		}
	}
	return nil
}

func renderBlockedPlanHuman(w io.Writer, plan app.Plan) error {
	reason := ensurePeriod(plan.BlockedReason)
	if _, err := fmt.Fprintf(w, "Cannot apply PR #%d: %s\n", plan.PR.Number, reason); err != nil {
		return err
	}
	guidance := blockedGuidance(plan.BlockedReason)
	if guidance == "" {
		return nil
	}
	_, err := fmt.Fprintf(w, "\n%s\n", guidance)
	return err
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

func strategyLine(plan app.Plan) string {
	method := plan.Method
	if method == "" {
		method = plan.RequestedMethod
	}
	line := string(method)
	if plan.RequestedMethod == app.MethodAuto && plan.Method != "" && plan.MergeShape != "" {
		return fmt.Sprintf("%s, auto-detected from GitHub %s", plan.Method, mergeShapeLabel(plan.MergeShape))
	}
	if plan.RequestedMethod != "" && plan.RequestedMethod != method {
		return fmt.Sprintf("%s, requested as %s", method, plan.RequestedMethod)
	}
	return line
}

func mergeShapeLabel(shape string) string {
	switch shape {
	case "rebase":
		return "rebase merge"
	case "squash":
		return "squash merge"
	case "merge_commit":
		return "merge commit"
	default:
		return shape
	}
}

func blockedGuidance(reason string) string {
	switch {
	case strings.Contains(reason, "dirty"):
		return "Commit, stash, or discard local changes, then retry."
	case strings.Contains(reason, "already in progress"):
		return "Finish or abort the current Git operation, then retry."
	case strings.Contains(reason, "not merged"):
		return "Merge the PR, then retry."
	default:
		return ""
	}
}

func ensurePeriod(text string) string {
	if text == "" {
		return text
	}
	switch text[len(text)-1:] {
	case ".", "!", "?":
		return text
	default:
		return text + "."
	}
}

func commitLabel(count int) string {
	if count == 1 {
		return "commit"
	}
	return "commits"
}

func shortSHA(sha string) string {
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}
