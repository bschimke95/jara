package llm

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bschimke95/jara/internal/model"
)

// FormatStatusContext converts the cluster status into a concise text
// representation suitable for injection into an LLM system prompt.
func FormatStatusContext(status *model.FullStatus) string {
	if status == nil {
		return "No cluster status available."
	}

	var b strings.Builder

	// Model info.
	fmt.Fprintf(&b, "Model: %s (cloud: %s, region: %s, version: %s)\n\n",
		status.Model.Name, status.Model.Cloud, status.Model.Region, status.Model.Version)

	// Applications.
	appNames := sortedKeys(status.Applications)
	if len(appNames) > 0 {
		b.WriteString("Applications:\n")
		for _, name := range appNames {
			app := status.Applications[name]
			fmt.Fprintf(&b, "  %s: status=%s charm=%s scale=%d",
				name, app.Status, app.Charm, app.Scale)
			if app.StatusMessage != "" {
				fmt.Fprintf(&b, " message=%q", app.StatusMessage)
			}
			if app.Exposed {
				b.WriteString(" exposed=true")
			}
			b.WriteString("\n")

			// Units with non-active/idle statuses (highlight problems).
			for _, u := range app.Units {
				if isNominal(u.WorkloadStatus, u.AgentStatus) {
					continue
				}
				fmt.Fprintf(&b, "    %s: workload=%s agent=%s",
					u.Name, u.WorkloadStatus, u.AgentStatus)
				if u.WorkloadMessage != "" {
					fmt.Fprintf(&b, " message=%q", u.WorkloadMessage)
				}
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}

	// Machines.
	machIDs := sortedKeys(status.Machines)
	if len(machIDs) > 0 {
		b.WriteString("Machines:\n")
		for _, id := range machIDs {
			mach := status.Machines[id]
			fmt.Fprintf(&b, "  %s: status=%s dns=%s base=%s",
				id, mach.Status, mach.DNSName, mach.Base)
			if mach.StatusMessage != "" {
				fmt.Fprintf(&b, " message=%q", mach.StatusMessage)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Relations.
	if len(status.Relations) > 0 {
		b.WriteString("Relations:\n")
		for _, rel := range status.Relations {
			endpoints := make([]string, len(rel.Endpoints))
			for i, ep := range rel.Endpoints {
				endpoints[i] = fmt.Sprintf("%s:%s", ep.ApplicationName, ep.Name)
			}
			fmt.Fprintf(&b, "  %s interface=%s status=%s\n",
				strings.Join(endpoints, " <-> "), rel.Interface, rel.Status)
		}
		b.WriteString("\n")
	}

	// Summary counts.
	fmt.Fprintf(&b, "Summary: %d applications, %d machines, %d relations",
		len(status.Applications), len(status.Machines), len(status.Relations))

	return b.String()
}

func isNominal(workload, agent string) bool {
	return (workload == "active" || workload == "") && (agent == "idle" || agent == "")
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
