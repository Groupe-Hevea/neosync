package datasync_workflow

import "fmt"

// buildTableName constructs a fully qualified table name from schema and table.
// If schema is empty, returns just the table name.
func buildTableName(schema, table string) string {
	if schema == "" {
		return table
	}
	return fmt.Sprintf("%s.%s", schema, table)
}
