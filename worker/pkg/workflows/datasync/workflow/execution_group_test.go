package datasync_workflow

import (
	"testing"

	benthosbuilder "github.com/Groupe-Hevea/neosync/internal/benthos/benthos-builder"
	runconfigs "github.com/Groupe-Hevea/neosync/internal/runconfigs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildExecutionGroups_NoCycles(t *testing.T) {
	configs := []*benthosbuilder.BenthosConfigResponse{
		{
			Name:        "public.users.insert",
			TableSchema: "public",
			TableName:   "users",
			RunType:     runconfigs.RunTypeInsert,
			DependsOn:   []*runconfigs.DependsOn{},
		},
		{
			Name:        "public.orders.insert",
			TableSchema: "public",
			TableName:   "orders",
			RunType:     runconfigs.RunTypeInsert,
			DependsOn: []*runconfigs.DependsOn{
				{Table: "public.users", Columns: []string{"id"}},
			},
		},
	}

	groups, err := buildExecutionGroups(configs)
	require.NoError(t, err)

	// Should create 2 independent table groups
	assert.Equal(t, 2, len(groups))

	// Find users group
	var usersGroup *ExecutionGroup
	for _, g := range groups {
		if g.ID == "table:public.users" {
			usersGroup = g
			break
		}
	}
	require.NotNil(t, usersGroup)
	assert.False(t, usersGroup.IsInCycle)
	assert.Equal(t, 1, len(usersGroup.InsertConfigs))
	assert.Equal(t, 0, len(usersGroup.UpdateConfigs))
	assert.Equal(t, 0, len(usersGroup.DependsOnGroups))

	// Find orders group
	var ordersGroup *ExecutionGroup
	for _, g := range groups {
		if g.ID == "table:public.orders" {
			ordersGroup = g
			break
		}
	}
	require.NotNil(t, ordersGroup)
	assert.False(t, ordersGroup.IsInCycle)
	assert.Equal(t, 1, len(ordersGroup.InsertConfigs))
	assert.Equal(t, 0, len(ordersGroup.UpdateConfigs))
	assert.Equal(t, 1, len(ordersGroup.DependsOnGroups))
	assert.Contains(t, ordersGroup.DependsOnGroups, "table:public.users")
}

func TestBuildExecutionGroups_SimpleCycle(t *testing.T) {
	configs := []*benthosbuilder.BenthosConfigResponse{
		{
			Name:        "public.store_customers.insert",
			TableSchema: "public",
			TableName:   "store_customers",
			RunType:     runconfigs.RunTypeInsert,
			DependsOn:   []*runconfigs.DependsOn{},
		},
		{
			Name:        "public.store_customers.update.1",
			TableSchema: "public",
			TableName:   "store_customers",
			RunType:     runconfigs.RunTypeUpdate,
			DependsOn: []*runconfigs.DependsOn{
				{Table: "public.referral_codes", Columns: []string{"id"}},
			},
		},
		{
			Name:        "public.referral_codes.insert",
			TableSchema: "public",
			TableName:   "referral_codes",
			RunType:     runconfigs.RunTypeInsert,
			DependsOn: []*runconfigs.DependsOn{
				{Table: "public.store_customers", Columns: []string{"id"}},
			},
		},
	}

	groups, err := buildExecutionGroups(configs)
	require.NoError(t, err)

	// Should create 1 cycle group
	assert.Equal(t, 1, len(groups))

	cycleGroup := groups[0]
	assert.True(t, cycleGroup.IsInCycle)
	assert.Equal(t, 2, len(cycleGroup.Tables))
	assert.Contains(t, cycleGroup.Tables, "public.store_customers")
	assert.Contains(t, cycleGroup.Tables, "public.referral_codes")

	// Should have 2 INSERTs and 1 UPDATE
	assert.Equal(t, 2, len(cycleGroup.InsertConfigs))
	assert.Equal(t, 1, len(cycleGroup.UpdateConfigs))

	// Cycle should have no external dependencies
	assert.Equal(t, 0, len(cycleGroup.DependsOnGroups))
}

func TestBuildExecutionGroups_ThreeTableCycle(t *testing.T) {
	configs := []*benthosbuilder.BenthosConfigResponse{
		{
			Name:        "public.addresses.insert",
			TableSchema: "public",
			TableName:   "addresses",
			RunType:     runconfigs.RunTypeInsert,
			DependsOn:   []*runconfigs.DependsOn{},
		},
		{
			Name:        "public.addresses.update.1",
			TableSchema: "public",
			TableName:   "addresses",
			RunType:     runconfigs.RunTypeUpdate,
			DependsOn: []*runconfigs.DependsOn{
				{Table: "public.orders", Columns: []string{"id"}},
			},
		},
		{
			Name:        "public.customers.insert",
			TableSchema: "public",
			TableName:   "customers",
			RunType:     runconfigs.RunTypeInsert,
			DependsOn: []*runconfigs.DependsOn{
				{Table: "public.addresses", Columns: []string{"id"}},
			},
		},
		{
			Name:        "public.customers.update.1",
			TableSchema: "public",
			TableName:   "customers",
			RunType:     runconfigs.RunTypeUpdate,
			DependsOn: []*runconfigs.DependsOn{
				{Table: "public.addresses", Columns: []string{"id"}},
			},
		},
		{
			Name:        "public.orders.insert",
			TableSchema: "public",
			TableName:   "orders",
			RunType:     runconfigs.RunTypeInsert,
			DependsOn: []*runconfigs.DependsOn{
				{Table: "public.customers", Columns: []string{"id"}},
			},
		},
		{
			Name:        "public.orders.update.1",
			TableSchema: "public",
			TableName:   "orders",
			RunType:     runconfigs.RunTypeUpdate,
			DependsOn: []*runconfigs.DependsOn{
				{Table: "public.customers", Columns: []string{"id"}},
			},
		},
	}

	groups, err := buildExecutionGroups(configs)
	require.NoError(t, err)

	// Should create 1 cycle group
	assert.Equal(t, 1, len(groups))

	cycleGroup := groups[0]
	assert.True(t, cycleGroup.IsInCycle)
	assert.Equal(t, 3, len(cycleGroup.Tables))
	assert.Contains(t, cycleGroup.Tables, "public.addresses")
	assert.Contains(t, cycleGroup.Tables, "public.customers")
	assert.Contains(t, cycleGroup.Tables, "public.orders")

	// Should have 3 INSERTs and 3 UPDATEs
	assert.Equal(t, 3, len(cycleGroup.InsertConfigs))
	assert.Equal(t, 3, len(cycleGroup.UpdateConfigs))
}

func TestBuildExecutionGroups_CycleWithExternalDependency(t *testing.T) {
	configs := []*benthosbuilder.BenthosConfigResponse{
		// Independent table
		{
			Name:        "public.regions.insert",
			TableSchema: "public",
			TableName:   "regions",
			RunType:     runconfigs.RunTypeInsert,
			DependsOn:   []*runconfigs.DependsOn{},
		},
		// Cycle
		{
			Name:        "public.store_customers.insert",
			TableSchema: "public",
			TableName:   "store_customers",
			RunType:     runconfigs.RunTypeInsert,
			DependsOn: []*runconfigs.DependsOn{
				{Table: "public.regions", Columns: []string{"id"}},
			},
		},
		{
			Name:        "public.store_customers.update.1",
			TableSchema: "public",
			TableName:   "store_customers",
			RunType:     runconfigs.RunTypeUpdate,
			DependsOn: []*runconfigs.DependsOn{
				{Table: "public.referral_codes", Columns: []string{"id"}},
			},
		},
		{
			Name:        "public.referral_codes.insert",
			TableSchema: "public",
			TableName:   "referral_codes",
			RunType:     runconfigs.RunTypeInsert,
			DependsOn: []*runconfigs.DependsOn{
				{Table: "public.store_customers", Columns: []string{"id"}},
			},
		},
	}

	groups, err := buildExecutionGroups(configs)
	require.NoError(t, err)

	// Should create 2 groups: 1 for regions, 1 for cycle
	assert.Equal(t, 2, len(groups))

	// Find cycle group
	var cycleGroup *ExecutionGroup
	for _, g := range groups {
		if g.IsInCycle {
			cycleGroup = g
			break
		}
	}
	require.NotNil(t, cycleGroup)

	// Cycle should depend on regions group
	assert.Equal(t, 1, len(cycleGroup.DependsOnGroups))
	assert.Contains(t, cycleGroup.DependsOnGroups, "table:public.regions")
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
