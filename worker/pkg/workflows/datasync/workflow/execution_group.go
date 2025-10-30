package datasync_workflow

import (
	benthosbuilder "github.com/Groupe-Hevea/neosync/internal/benthos/benthos-builder"
	runconfigs "github.com/Groupe-Hevea/neosync/internal/runconfigs"
)

// ExecutionGroup represents a logical unit of execution for table synchronization.
// It groups INSERT and UPDATE configs that should be orchestrated together.
//
// For tables in a circular dependency cycle:
//   - All INSERT configs execute together (Phase 1)
//   - All UPDATE configs execute after INSERTs complete (Phase 2)
//
// For independent tables:
//   - The group contains only INSERT configs (single phase)
type ExecutionGroup struct {
	ID             string                                      // Unique identifier (e.g., "cycle:table1_table2" or "table:table1")
	Tables         []string                                    // Tables included in this group
	InsertConfigs  []*benthosbuilder.BenthosConfigResponse     // INSERT configs to execute in phase 1
	UpdateConfigs  []*benthosbuilder.BenthosConfigResponse     // UPDATE configs to execute in phase 2
	DependsOnGroups []string                                   // IDs of groups that must complete before this one
	IsInCycle      bool                                        // Whether this group represents a circular dependency cycle
}

// buildExecutionGroups creates execution groups from Benthos configs.
// It detects circular dependencies and groups configs accordingly.
func buildExecutionGroups(configs []*benthosbuilder.BenthosConfigResponse) ([]*ExecutionGroup, error) {
	// Build dependency graph to detect cycles
	graph := buildConfigDependencyGraph(configs)
	cycles := runconfigs.FindCircularDependencies(graph)

	// Map tables to their cycle (if any)
	tableToCycle := make(map[string]int) // table -> cycle index (-1 if not in cycle)
	for i, cycle := range cycles {
		for _, table := range cycle {
			tableToCycle[table] = i
		}
	}

	// Group configs by cycle or individual table
	cycleGroups := make(map[int]*ExecutionGroup)  // cycle index -> group
	tableGroups := make(map[string]*ExecutionGroup) // table -> group (for non-cycle tables)

	for _, cfg := range configs {
		tableName := buildTableName(cfg.TableSchema, cfg.TableName)

		if cycleIdx, inCycle := tableToCycle[tableName]; inCycle {
			// Part of a cycle - add to cycle group
			if cycleGroups[cycleIdx] == nil {
				cycleGroups[cycleIdx] = &ExecutionGroup{
					ID:        buildCycleGroupID(cycles[cycleIdx]),
					Tables:    cycles[cycleIdx],
					IsInCycle: true,
				}
			}
			group := cycleGroups[cycleIdx]
			if cfg.RunType == runconfigs.RunTypeInsert {
				group.InsertConfigs = append(group.InsertConfigs, cfg)
			} else {
				group.UpdateConfigs = append(group.UpdateConfigs, cfg)
			}
		} else {
			// Not in a cycle - create individual table group
			if tableGroups[tableName] == nil {
				tableGroups[tableName] = &ExecutionGroup{
					ID:        "table:" + tableName,
					Tables:    []string{tableName},
					IsInCycle: false,
				}
			}
			group := tableGroups[tableName]
			if cfg.RunType == runconfigs.RunTypeInsert {
				group.InsertConfigs = append(group.InsertConfigs, cfg)
			} else {
				group.UpdateConfigs = append(group.UpdateConfigs, cfg)
			}
		}
	}

	// Combine all groups
	var groups []*ExecutionGroup
	for _, g := range cycleGroups {
		groups = append(groups, g)
	}
	for _, g := range tableGroups {
		groups = append(groups, g)
	}

	// Calculate inter-group dependencies
	for _, group := range groups {
		group.DependsOnGroups = calculateGroupDependencies(group, groups, tableToCycle, cycles)
	}

	return groups, nil
}

// buildConfigDependencyGraph builds a table-level dependency graph from configs
func buildConfigDependencyGraph(configs []*benthosbuilder.BenthosConfigResponse) map[string][]string {
	graph := make(map[string][]string)

	for _, cfg := range configs {
		tableName := buildTableName(cfg.TableSchema, cfg.TableName)

		for _, dep := range cfg.DependsOn {
			// Only add if it's a dependency to a different table
			if dep.Table != tableName {
				// Check if already added to avoid duplicates
				found := false
				for _, existing := range graph[tableName] {
					if existing == dep.Table {
						found = true
						break
					}
				}
				if !found {
					graph[tableName] = append(graph[tableName], dep.Table)
				}
			}
		}
	}

	return graph
}

// calculateGroupDependencies determines which groups this group depends on
func calculateGroupDependencies(
	group *ExecutionGroup,
	allGroups []*ExecutionGroup,
	tableToCycle map[string]int,
	cycles [][]string,
) []string {
	dependentGroupIDs := make(map[string]bool)

	// Check all configs in this group
	allConfigs := append(group.InsertConfigs, group.UpdateConfigs...)
	for _, cfg := range allConfigs {
		currentTable := buildTableName(cfg.TableSchema, cfg.TableName)

		for _, dep := range cfg.DependsOn {
			// Skip self-references
			if dep.Table == currentTable {
				continue
			}

			// Skip dependencies within the same cycle
			if group.IsInCycle {
				currentCycleIdx := tableToCycle[currentTable]
				if depCycleIdx, ok := tableToCycle[dep.Table]; ok && depCycleIdx == currentCycleIdx {
					continue
				}
			}

			// Find which group contains this dependency
			for _, otherGroup := range allGroups {
				if otherGroup.ID == group.ID {
					continue
				}
				for _, table := range otherGroup.Tables {
					if table == dep.Table {
						dependentGroupIDs[otherGroup.ID] = true
						break
					}
				}
			}
		}
	}

	// Convert map to slice
	var deps []string
	for id := range dependentGroupIDs {
		deps = append(deps, id)
	}

	return deps
}

// buildCycleGroupID creates a unique ID for a cycle group
func buildCycleGroupID(tables []string) string {
	// Sort tables for consistent ID
	sorted := make([]string, len(tables))
	copy(sorted, tables)
	// Simple bubble sort for small slices
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	id := "cycle:"
	for i, table := range sorted {
		if i > 0 {
			id += "_"
		}
		id += table
	}
	return id
}
