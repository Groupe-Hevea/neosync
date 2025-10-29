package datasync_workflow

import (
	"testing"

	benthosbuilder "github.com/Groupe-Hevea/neosync/internal/benthos/benthos-builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCompletionTracker(t *testing.T) {
	configs := []*benthosbuilder.BenthosConfigResponse{
		{Name: "schema.table1.insert", TableSchema: "schema", TableName: "table1"},
		{Name: "schema.table1.update.1", TableSchema: "schema", TableName: "table1"},
		{Name: "schema.table2.insert", TableSchema: "schema", TableName: "table2"},
	}

	tracker := NewCompletionTracker(configs)

	require.NotNil(t, tracker)
	assert.Equal(t, 2, len(tracker.tableCompletions))

	// Check table1 has 2 configs
	state1, exists := tracker.tableCompletions["schema.table1"]
	require.True(t, exists)
	assert.Equal(t, 2, state1.TotalRunConfigs)

	// Check table2 has 1 config
	state2, exists := tracker.tableCompletions["schema.table2"]
	require.True(t, exists)
	assert.Equal(t, 1, state2.TotalRunConfigs)
}

func TestMarkRunConfigComplete_SingleConfig(t *testing.T) {
	configs := []*benthosbuilder.BenthosConfigResponse{
		{Name: "schema.table1.insert", TableSchema: "schema", TableName: "table1", Columns: []string{"id", "name"}},
	}

	tracker := NewCompletionTracker(configs)

	// Mark the single config as complete
	err := tracker.MarkRunConfigComplete("schema.table1", "schema.table1.insert", []string{"id", "name"})
	require.NoError(t, err)

	// Table should be complete immediately
	assert.True(t, tracker.IsTableComplete("schema.table1"))

	// Check columns
	cols, ok := tracker.GetCompletedColumns("schema.table1")
	assert.True(t, ok)
	assert.ElementsMatch(t, []string{"id", "name"}, cols)
}

func TestMarkRunConfigComplete_MultipleConfigs(t *testing.T) {
	configs := []*benthosbuilder.BenthosConfigResponse{
		{Name: "schema.user.insert", TableSchema: "schema", TableName: "user", Columns: []string{"id", "username"}},
		{Name: "schema.user.update.1", TableSchema: "schema", TableName: "user", Columns: []string{"created_by"}},
		{Name: "schema.user.update.2", TableSchema: "schema", TableName: "user", Columns: []string{"updated_by"}},
	}

	tracker := NewCompletionTracker(configs)

	// Mark first config complete
	err := tracker.MarkRunConfigComplete("schema.user", "schema.user.insert", []string{"id", "username"})
	require.NoError(t, err)

	// Table should NOT be complete yet
	assert.False(t, tracker.IsTableComplete("schema.user"))

	// Mark second config complete
	err = tracker.MarkRunConfigComplete("schema.user", "schema.user.update.1", []string{"created_by"})
	require.NoError(t, err)

	// Table should still NOT be complete
	assert.False(t, tracker.IsTableComplete("schema.user"))

	// Mark third config complete
	err = tracker.MarkRunConfigComplete("schema.user", "schema.user.update.2", []string{"updated_by"})
	require.NoError(t, err)

	// NOW table should be complete
	assert.True(t, tracker.IsTableComplete("schema.user"))

	// Check all columns are present
	cols, ok := tracker.GetCompletedColumns("schema.user")
	assert.True(t, ok)
	assert.ElementsMatch(t, []string{"id", "username", "created_by", "updated_by"}, cols)
}

func TestMarkRunConfigComplete_UnknownTable(t *testing.T) {
	configs := []*benthosbuilder.BenthosConfigResponse{
		{Name: "schema.table1.insert", TableSchema: "schema", TableName: "table1"},
	}

	tracker := NewCompletionTracker(configs)

	// Try to mark a non-existent table
	err := tracker.MarkRunConfigComplete("schema.unknown", "schema.unknown.insert", []string{"id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown table")
}

func TestMarkRunConfigComplete_DuplicateColumns(t *testing.T) {
	configs := []*benthosbuilder.BenthosConfigResponse{
		{Name: "schema.table.insert", TableSchema: "schema", TableName: "table", Columns: []string{"id", "name"}},
		{Name: "schema.table.update.1", TableSchema: "schema", TableName: "table", Columns: []string{"name", "email"}},
	}

	tracker := NewCompletionTracker(configs)

	err := tracker.MarkRunConfigComplete("schema.table", "schema.table.insert", []string{"id", "name"})
	require.NoError(t, err)

	err = tracker.MarkRunConfigComplete("schema.table", "schema.table.update.1", []string{"name", "email"})
	require.NoError(t, err)

	// Should have unique columns only
	cols, ok := tracker.GetCompletedColumns("schema.table")
	assert.True(t, ok)
	assert.ElementsMatch(t, []string{"id", "name", "email"}, cols)
	assert.Equal(t, 3, len(cols)) // Not 4, because "name" is deduplicated
}

func TestToMap(t *testing.T) {
	configs := []*benthosbuilder.BenthosConfigResponse{
		{Name: "schema.table1.insert", TableSchema: "schema", TableName: "table1", Columns: []string{"id"}},
		{Name: "schema.table2.insert", TableSchema: "schema", TableName: "table2", Columns: []string{"id"}},
		{Name: "schema.table2.update.1", TableSchema: "schema", TableName: "table2", Columns: []string{"fk"}},
	}

	tracker := NewCompletionTracker(configs)

	// Complete table1
	err := tracker.MarkRunConfigComplete("schema.table1", "schema.table1.insert", []string{"id"})
	require.NoError(t, err)

	// Partially complete table2
	err = tracker.MarkRunConfigComplete("schema.table2", "schema.table2.insert", []string{"id"})
	require.NoError(t, err)

	// Get map - should only have table1
	resultMap, err := tracker.ToMap()
	require.NoError(t, err)
	assert.Equal(t, 1, len(resultMap))
	assert.Contains(t, resultMap, "schema.table1")
	assert.NotContains(t, resultMap, "schema.table2")

	// Complete table2
	err = tracker.MarkRunConfigComplete("schema.table2", "schema.table2.update.1", []string{"fk"})
	require.NoError(t, err)

	// Now map should have both
	resultMap, err = tracker.ToMap()
	require.NoError(t, err)
	assert.Equal(t, 2, len(resultMap))
	assert.Contains(t, resultMap, "schema.table1")
	assert.Contains(t, resultMap, "schema.table2")
}

func TestGetCompletionStatus(t *testing.T) {
	configs := []*benthosbuilder.BenthosConfigResponse{
		{Name: "schema.table1.insert", TableSchema: "schema", TableName: "table1"},
		{Name: "schema.table2.insert", TableSchema: "schema", TableName: "table2"},
		{Name: "schema.table2.update.1", TableSchema: "schema", TableName: "table2"},
		{Name: "schema.table2.update.2", TableSchema: "schema", TableName: "table2"},
	}

	tracker := NewCompletionTracker(configs)

	// Complete table1
	err := tracker.MarkRunConfigComplete("schema.table1", "schema.table1.insert", []string{"id"})
	require.NoError(t, err)

	// Partially complete table2
	err = tracker.MarkRunConfigComplete("schema.table2", "schema.table2.insert", []string{"id"})
	require.NoError(t, err)
	err = tracker.MarkRunConfigComplete("schema.table2", "schema.table2.update.1", []string{"fk1"})
	require.NoError(t, err)

	status := tracker.GetCompletionStatus()

	assert.Equal(t, "1/1 (COMPLETE)", status["schema.table1"])
	assert.Equal(t, "2/3", status["schema.table2"])
}

func TestConcurrentAccess(t *testing.T) {
	configs := []*benthosbuilder.BenthosConfigResponse{
		{Name: "schema.table1.insert", TableSchema: "schema", TableName: "table1"},
		{Name: "schema.table2.insert", TableSchema: "schema", TableName: "table2"},
		{Name: "schema.table3.insert", TableSchema: "schema", TableName: "table3"},
	}

	tracker := NewCompletionTracker(configs)

	// Test concurrent writes
	done := make(chan bool, 3)

	go func() {
		_ = tracker.MarkRunConfigComplete("schema.table1", "schema.table1.insert", []string{"id"})
		done <- true
	}()

	go func() {
		_ = tracker.MarkRunConfigComplete("schema.table2", "schema.table2.insert", []string{"id"})
		done <- true
	}()

	go func() {
		_ = tracker.MarkRunConfigComplete("schema.table3", "schema.table3.insert", []string{"id"})
		done <- true
	}()

	// Wait for all goroutines
	<-done
	<-done
	<-done

	// All tables should be complete
	assert.True(t, tracker.IsTableComplete("schema.table1"))
	assert.True(t, tracker.IsTableComplete("schema.table2"))
	assert.True(t, tracker.IsTableComplete("schema.table3"))
}

func TestUnionStringSlices(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected []string
	}{
		{
			name:     "empty slices",
			a:        []string{},
			b:        []string{},
			expected: []string{},
		},
		{
			name:     "a empty",
			a:        []string{},
			b:        []string{"x", "y"},
			expected: []string{"x", "y"},
		},
		{
			name:     "b empty",
			a:        []string{"x", "y"},
			b:        []string{},
			expected: []string{"x", "y"},
		},
		{
			name:     "no overlap",
			a:        []string{"a", "b"},
			b:        []string{"c", "d"},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name:     "complete overlap",
			a:        []string{"a", "b"},
			b:        []string{"a", "b"},
			expected: []string{"a", "b"},
		},
		{
			name:     "partial overlap",
			a:        []string{"a", "b", "c"},
			b:        []string{"b", "c", "d"},
			expected: []string{"a", "b", "c", "d"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unionStringSlices(tt.a, tt.b)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestBuildTableName(t *testing.T) {
	tests := []struct {
		name     string
		schema   string
		table    string
		expected string
	}{
		{
			name:     "with schema",
			schema:   "public",
			table:    "users",
			expected: "public.users",
		},
		{
			name:     "without schema",
			schema:   "",
			table:    "users",
			expected: "users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildTableName(tt.schema, tt.table)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsInsertCompleted(t *testing.T) {
	configs := []*benthosbuilder.BenthosConfigResponse{
		{Name: "schema.user__user.insert", TableSchema: "schema", TableName: "user__user"},
		{Name: "schema.user__user.update.1", TableSchema: "schema", TableName: "user__user"},
		{Name: "schema.user__user.update.2", TableSchema: "schema", TableName: "user__user"},
		{Name: "schema.products.insert", TableSchema: "schema", TableName: "products"},
	}

	tracker := NewCompletionTracker(configs)

	// Initially, no INSERT is completed
	assert.False(t, tracker.IsInsertCompleted("schema.user__user"), "INSERT should not be completed initially")
	assert.False(t, tracker.IsInsertCompleted("schema.products"), "INSERT should not be completed initially")

	// Complete the INSERT for user__user
	err := tracker.MarkRunConfigComplete("schema.user__user", "schema.user__user.insert", []string{"id", "name"})
	require.NoError(t, err)

	// Now INSERT should be completed for user__user
	assert.True(t, tracker.IsInsertCompleted("schema.user__user"), "INSERT should be completed after marking")
	assert.False(t, tracker.IsInsertCompleted("schema.products"), "products INSERT should still not be completed")

	// Complete an UPDATE (not INSERT) for user__user
	err = tracker.MarkRunConfigComplete("schema.user__user", "schema.user__user.update.1", []string{"manager_id"})
	require.NoError(t, err)

	// INSERT should still be completed
	assert.True(t, tracker.IsInsertCompleted("schema.user__user"), "INSERT should still be completed")

	// Test with unknown table
	assert.False(t, tracker.IsInsertCompleted("unknown.table"), "Unknown table should return false")
}

func TestGetInsertColumns(t *testing.T) {
	configs := []*benthosbuilder.BenthosConfigResponse{
		{Name: "schema.user__user.insert", TableSchema: "schema", TableName: "user__user"},
		{Name: "schema.user__user.update.1", TableSchema: "schema", TableName: "user__user"},
		{Name: "schema.products.insert", TableSchema: "schema", TableName: "products"},
	}

	tracker := NewCompletionTracker(configs)

	// Initially, no columns available
	cols, hasInsert := tracker.GetInsertColumns("schema.user__user")
	assert.False(t, hasInsert, "Should not have INSERT columns initially")
	assert.Nil(t, cols, "Columns should be nil initially")

	// Complete the INSERT for user__user
	err := tracker.MarkRunConfigComplete("schema.user__user", "schema.user__user.insert", []string{"id", "name", "email"})
	require.NoError(t, err)

	// Now columns should be available
	cols, hasInsert = tracker.GetInsertColumns("schema.user__user")
	assert.True(t, hasInsert, "Should have INSERT columns after marking")
	assert.ElementsMatch(t, []string{"id", "name", "email"}, cols, "Should return INSERT columns")

	// Complete an UPDATE - columns should accumulate
	err = tracker.MarkRunConfigComplete("schema.user__user", "schema.user__user.update.1", []string{"manager_id"})
	require.NoError(t, err)

	// Columns should now include both INSERT and UPDATE columns
	cols, hasInsert = tracker.GetInsertColumns("schema.user__user")
	assert.True(t, hasInsert, "Should still have INSERT columns")
	assert.ElementsMatch(t, []string{"id", "name", "email", "manager_id"}, cols, "Should return all accumulated columns")

	// Test with unknown table
	cols, hasInsert = tracker.GetInsertColumns("unknown.table")
	assert.False(t, hasInsert, "Unknown table should return false")
	assert.Nil(t, cols, "Unknown table should return nil columns")

	// Test with table that has no INSERT completed yet
	cols, hasInsert = tracker.GetInsertColumns("schema.products")
	assert.False(t, hasInsert, "Table with no INSERT completed should return false")
	assert.Nil(t, cols, "Table with no INSERT completed should return nil columns")
}
