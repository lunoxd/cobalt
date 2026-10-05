package inspector

import (
	"testing"
)

func TestValidateIdentifier(t *testing.T) {
	valid := []string{"projects", "api_keys", "users_1", "table_name_v2", "_private"}
	for _, v := range valid {
		if err := validateIdentifier(v); err != nil {
			t.Errorf("expected %q to be valid identifier, got error: %v", v, err)
		}
	}

	invalid := []string{
		"projects; DROP TABLE users;",
		"table name",
		"table-name",
		"table'--",
		"123table",
		"table$name",
		"",
	}
	for _, inv := range invalid {
		if err := validateIdentifier(inv); err == nil {
			t.Errorf("expected %q to fail validation, but it passed", inv)
		}
	}
}
