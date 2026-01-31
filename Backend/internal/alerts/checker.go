package alerts

import (
	"fmt"
	"time"

	"github.com/dennisdiepolder/monti/backend/internal/types"
)

// CheckAgentAlerts evaluates alert rules for a slice of agents,
// mutating each agent's Alerts field in place.
func CheckAgentAlerts(agents []types.AgentInfo) {
	now := time.Now()
	for i := range agents {
		agents[i].Alerts = nil

		switch agents[i].State {
		case types.StateAfterCallWork:
			if agents[i].ACWStartTime != nil {
				dur := now.Sub(*agents[i].ACWStartTime)
				if dur > 5*time.Minute {
					agents[i].Alerts = append(agents[i].Alerts, types.AgentAlert{
						Rule:     "acw_long",
						Severity: types.SeverityWarning,
						Message:  fmt.Sprintf("ACW for %s", formatDuration(dur)),
					})
				}
			} else {
				dur := now.Sub(agents[i].StateStart)
				if dur > 5*time.Minute {
					agents[i].Alerts = append(agents[i].Alerts, types.AgentAlert{
						Rule:     "acw_long",
						Severity: types.SeverityWarning,
						Message:  fmt.Sprintf("ACW for %s", formatDuration(dur)),
					})
				}
			}

		case types.StateBreak:
			if agents[i].BreakStartTime != nil {
				dur := now.Sub(*agents[i].BreakStartTime)
				if dur > 10*time.Minute {
					agents[i].Alerts = append(agents[i].Alerts, types.AgentAlert{
						Rule:     "break_long",
						Severity: types.SeverityCritical,
						Message:  fmt.Sprintf("Break for %s", formatDuration(dur)),
					})
				}
			} else {
				dur := now.Sub(agents[i].StateStart)
				if dur > 10*time.Minute {
					agents[i].Alerts = append(agents[i].Alerts, types.AgentAlert{
						Rule:     "break_long",
						Severity: types.SeverityCritical,
						Message:  fmt.Sprintf("Break for %s", formatDuration(dur)),
					})
				}
			}
		}
	}
}

func formatDuration(d time.Duration) string {
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	if mins >= 60 {
		hours := mins / 60
		mins = mins % 60
		return fmt.Sprintf("%dh%dm", hours, mins)
	}
	return fmt.Sprintf("%dm%ds", mins, secs)
}
