package datasync_workflow

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"

	benthosbuilder "github.com/Groupe-Hevea/neosync/internal/benthos/benthos-builder"
)

// CompletionTracker tracks the completion status of table synchronization.
// It ensures that a table is only marked as complete once ALL of its RunConfigs
// (INSERT + all UPDATEs) have finished executing.
//
// This prevents race conditions where dependent tables start executing before
// their parent tables are fully synchronized.
type CompletionTracker struct {
	mu               sync.RWMutex
	tableCompletions map[string]*TableCompletionState
	fullyCompleted   map[string][]string // tableName -> all columns
}

// TableCompletionState tracks the completion state of a single table.
type TableCompletionState struct {
	TotalRunConfigs     int             // Total number of RunConfigs for this table (e.g., 3 for insert + 2 updates)
	CompletedRunConfigs map[string]bool // runConfigId -> completed status
	AllColumns          []string        // Union of all columns from all completed RunConfigs
}

// NewCompletionTracker creates a new CompletionTracker initialized with the expected
// number of RunConfigs per table based on the provided Benthos configs.
func NewCompletionTracker(configs []*benthosbuilder.BenthosConfigResponse) *CompletionTracker {
	tracker := &CompletionTracker{
		tableCompletions: make(map[string]*TableCompletionState),
		fullyCompleted:   make(map[string][]string),
	}

	// Pre-calculate how many RunConfigs exist per table
	configCountByTable := make(map[string]int)
	for _, cfg := range configs {
		tableName := buildTableName(cfg.TableSchema, cfg.TableName)
		configCountByTable[tableName]++
	}

	// Initialize TableCompletionState for each table
	for tableName, count := range configCountByTable {
		tracker.tableCompletions[tableName] = &TableCompletionState{
			TotalRunConfigs:     count,
			CompletedRunConfigs: make(map[string]bool),
			AllColumns:          []string{},
		}
	}

	return tracker
}

// MarkRunConfigComplete marks a specific RunConfig as completed.
// When all RunConfigs for a table are complete, the table is marked as fully completed.
//
// Parameters:
//   - tableName: The fully qualified table name (e.g., "schema.table")
//   - runConfigId: The unique identifier for the RunConfig (e.g., "schema.table.insert")
//   - columns: The columns handled by this RunConfig
func (ct *CompletionTracker) MarkRunConfigComplete(tableName, runConfigId string, columns []string) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	state, exists := ct.tableCompletions[tableName]
	if !exists {
		slog.Error(
			"CompletionTracker: unknown table",
			"tableName", tableName,
			"runConfigId", runConfigId,
			"knownTables", ct.getTableNames(),
		)
		return fmt.Errorf("unknown table: %s", tableName)
	}

	// Mark this specific RunConfig as complete
	state.CompletedRunConfigs[runConfigId] = true
	slog.Debug(
		"CompletionTracker: marked RunConfig complete",
		"tableName", tableName,
		"runConfigId", runConfigId,
		"completedCount", len(state.CompletedRunConfigs),
		"totalCount", state.TotalRunConfigs,
	)

	// Add columns to the union of all columns
	state.AllColumns = unionStringSlices(state.AllColumns, columns)

	// Check if ALL RunConfigs for this table are now complete
	if len(state.CompletedRunConfigs) == state.TotalRunConfigs {
		// All RunConfigs complete - mark table as fully completed
		ct.fullyCompleted[tableName] = state.AllColumns
		slog.Info("CompletionTracker: table fully completed", "tableName", tableName, "runConfigCount", state.TotalRunConfigs)
	}

	return nil
}

// getTableNames returns a list of all known table names (must be called with lock held)
func (ct *CompletionTracker) getTableNames() []string {
	names := make([]string, 0, len(ct.tableCompletions))
	for name := range ct.tableCompletions {
		names = append(names, name)
	}
	return names
}

// IsTableComplete checks if a table is fully completed (all RunConfigs finished).
func (ct *CompletionTracker) IsTableComplete(tableName string) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	_, complete := ct.fullyCompleted[tableName]
	return complete
}

// IsInsertCompleted checks if the INSERT RunConfig for a table has completed.
// This is used for self-referencing tables where UPDATEs can start as soon as
// the INSERT finishes, without waiting for all UPDATEs to complete.
func (ct *CompletionTracker) IsInsertCompleted(tableName string) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	state, exists := ct.tableCompletions[tableName]
	if !exists {
		return false
	}

	// Check if any RunConfig ending with ".insert" is completed
	for runConfigId, completed := range state.CompletedRunConfigs {
		if completed && strings.HasSuffix(runConfigId, ".insert") {
			return true
		}
	}

	return false
}

// GetInsertColumns returns the columns available after INSERT completion.
// Used for self-referencing tables where UPDATEs need columns from INSERT.
func (ct *CompletionTracker) GetInsertColumns(tableName string) ([]string, bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	state, exists := ct.tableCompletions[tableName]
	if !exists {
		return nil, false
	}

	// Check if INSERT is completed (inline to avoid mutex deadlock)
	insertCompleted := false
	for runConfigId, completed := range state.CompletedRunConfigs {
		if completed && strings.HasSuffix(runConfigId, ".insert") {
			insertCompleted = true
			break
		}
	}

	// If INSERT is completed, return all columns accumulated so far
	if insertCompleted {
		return state.AllColumns, true
	}

	return nil, false
}

// GetCompletedColumns returns the columns that have been completed for a table.
// Returns the columns and true if the table is fully completed, or nil and false otherwise.
func (ct *CompletionTracker) GetCompletedColumns(tableName string) ([]string, bool) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	cols, ok := ct.fullyCompleted[tableName]
	return cols, ok
}

// ToMap converts the completion state to a map compatible with the existing
// AreConfigDependenciesSatisfied function.
//
// Returns: map[tableName][]columns for all FULLY completed tables only.
// This ensures that only tables with ALL RunConfigs completed are included.
func (ct *CompletionTracker) ToMap() (map[string][]string, error) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	// Return only fully completed tables
	result := make(map[string][]string, len(ct.fullyCompleted))
	for tableName, columns := range ct.fullyCompleted {
		result[tableName] = columns
	}

	return result, nil
}

// GetCompletionStatus returns the current completion status for debugging purposes.
// Returns a map of table name to "X/Y completed" string.
func (ct *CompletionTracker) GetCompletionStatus() map[string]string {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	status := make(map[string]string)
	for tableName, state := range ct.tableCompletions {
		completed := len(state.CompletedRunConfigs)
		total := state.TotalRunConfigs
		if ct.IsTableComplete(tableName) {
			status[tableName] = fmt.Sprintf("%d/%d (COMPLETE)", completed, total)
		} else {
			status[tableName] = fmt.Sprintf("%d/%d", completed, total)
		}
	}

	return status
}

// buildTableName constructs a fully qualified table name from schema and table.
func buildTableName(schema, table string) string {
	if schema == "" {
		return table
	}
	return fmt.Sprintf("%s.%s", schema, table)
}

// unionStringSlices returns the union of two string slices.
func unionStringSlices(a, b []string) []string {
	result := make([]string, len(a))
	copy(result, a)

	for _, item := range b {
		if !slices.Contains(result, item) {
			result = append(result, item)
		}
	}

	return result
}
