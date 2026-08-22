// Package comparison owns immutable deterministic Project comparisons.
package comparison

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/leohteixeira/opensource-project-intelligence/internal/metric"
)

var ErrInvalid = errors.New("invalid comparison")

type Project struct {
	ID       int64
	Name     string
	Resolved bool
	Deleted  bool
	Archived bool
	Metrics  []metric.Snapshot
}

type Cell struct {
	ProjectID int64           `json:"project_id,string"`
	Status    metric.Status   `json:"status"`
	Value     *float64        `json:"value,omitempty"`
	Version   string          `json:"version"`
	Evidence  []metric.Factor `json:"evidence"`
}

type Row struct {
	Metric string `json:"metric"`
	Unit   string `json:"unit"`
	Cells  []Cell `json:"cells"`
}

type Comparison struct {
	ID         int64         `json:"id,string"`
	ProjectIDs []int64       `json:"project_ids"`
	Window     metric.Window `json:"window"`
	Rows       []Row         `json:"rows"`
	CreatedAt  time.Time     `json:"created_at"`
}

func Materialize(id int64, projects []Project, window metric.Window) (Comparison, error) {
	if id <= 0 || len(projects) < 2 || len(projects) > 5 || window.Validate() != nil {
		return Comparison{}, ErrInvalid
	}
	seen := make(map[int64]struct{}, len(projects))
	projectIDs := make([]int64, 0, len(projects))
	for _, project := range projects {
		if project.ID <= 0 || project.Deleted || !project.Resolved {
			return Comparison{}, fmt.Errorf("%w: every Project must be resolved and available", ErrInvalid)
		}
		if _, duplicate := seen[project.ID]; duplicate {
			return Comparison{}, fmt.Errorf("%w: duplicate Project", ErrInvalid)
		}
		seen[project.ID] = struct{}{}
		projectIDs = append(projectIDs, project.ID)
	}
	definitionNames := make(map[string]struct{})
	for _, project := range projects {
		for _, snapshot := range project.Metrics {
			definitionNames[snapshot.Definition.Name] = struct{}{}
		}
	}
	names := make([]string, 0, len(definitionNames))
	for name := range definitionNames {
		names = append(names, name)
	}
	slices.Sort(names)
	rows := make([]Row, 0, len(names))
	for _, name := range names {
		row := Row{Metric: name, Cells: make([]Cell, 0, len(projects))}
		versions := make(map[string]struct{})
		for _, project := range projects {
			cell := Cell{ProjectID: project.ID, Status: metric.StatusInsufficientData}
			for _, snapshot := range project.Metrics {
				if snapshot.Definition.Name != name {
					continue
				}
				cell.Status, cell.Value, cell.Version, cell.Evidence = snapshot.Status, snapshot.Value, snapshot.Definition.Version, slices.Clone(snapshot.Factors)
				row.Unit = snapshot.Definition.Unit
				if !snapshot.Window.From.Equal(window.From) || !snapshot.Window.To.Equal(window.To) || !snapshot.Window.Cutoff.Equal(window.Cutoff) {
					cell.Status, cell.Value = metric.StatusIncomparable, nil
				}
				break
			}
			if cell.Version != "" {
				versions[strings.ToLower(cell.Version)] = struct{}{}
			}
			row.Cells = append(row.Cells, cell)
		}
		if len(versions) > 1 {
			for index := range row.Cells {
				if row.Cells[index].Version != "" {
					row.Cells[index].Status, row.Cells[index].Value = metric.StatusIncomparable, nil
				}
			}
		}
		rows = append(rows, row)
	}
	return Comparison{ID: id, ProjectIDs: projectIDs, Window: window, Rows: rows, CreatedAt: window.Cutoff}, nil
}
