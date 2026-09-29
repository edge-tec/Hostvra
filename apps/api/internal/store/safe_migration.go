package store

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrDestructiveMigration is returned when SQL contains statements forbidden by the Zero Data Loss policy
	ErrDestructiveMigration = errors.New("destructive migration blocked by zero-data-loss policy")
)

// StripCommentsAndLiterals strips SQL comments and string literals so keywords inside comments or strings aren't falsely flagged
func StripCommentsAndLiterals(sql string) string {
	var sb strings.Builder
	runes := []rune(sql)
	n := len(runes)
	i := 0

	for i < n {
		// Single line comment: -- ... \n
		if i+1 < n && runes[i] == '-' && runes[i+1] == '-' {
			i += 2
			for i < n && runes[i] != '\n' {
				i++
			}
			continue
		}
		// Multi line comment: /* ... */
		if i+1 < n && runes[i] == '/' && runes[i+1] == '*' {
			i += 2
			for i+1 < n && !(runes[i] == '*' && runes[i+1] == '/') {
				i++
			}
			i += 2 // skip */
			continue
		}
		// String literal: '...'
		if runes[i] == '\'' {
			i++
			for i < n {
				if runes[i] == '\'' {
					if i+1 < n && runes[i+1] == '\'' {
						i += 2 // escaped quote ''
						continue
					}
					i++ // closing quote
					break
				}
				i++
			}
			sb.WriteRune(' ')
			continue
		}
		// Dollar-quoted strings in PostgreSQL: $$...$$ or $tag$...$tag$
		if runes[i] == '$' {
			tagEnd := i + 1
			for tagEnd < n && ((runes[tagEnd] >= 'a' && runes[tagEnd] <= 'z') || (runes[tagEnd] >= 'A' && runes[tagEnd] <= 'Z') || (runes[tagEnd] >= '0' && runes[tagEnd] <= '9') || runes[tagEnd] == '_') {
				tagEnd++
			}
			if tagEnd < n && runes[tagEnd] == '$' {
				tag := string(runes[i : tagEnd+1])
				i = tagEnd + 1
				closeIdx := strings.Index(string(runes[i:]), tag)
				if closeIdx != -1 {
					i += closeIdx + len([]rune(tag))
					sb.WriteRune(' ')
					continue
				}
			}
		}

		sb.WriteRune(runes[i])
		i++
	}
	return sb.String()
}

// ValidateSafeMigration inspects migration SQL statements and rejects any destructive commands
// such as DROP DATABASE, DROP SCHEMA, DROP TABLE, TRUNCATE, or dangerous unconditional DELETE FROM.
func ValidateSafeMigration(sqlContent string) error {
	cleaned := StripCommentsAndLiterals(sqlContent)
	upper := strings.ToUpper(cleaned)

	// 1. Block DROP DATABASE
	if match, _ := regexp.MatchString(`\bDROP\s+DATABASE\b`, upper); match {
		return fmt.Errorf("%w: DROP DATABASE is strictly forbidden in automated updates", ErrDestructiveMigration)
	}

	// 2. Block DROP SCHEMA
	if match, _ := regexp.MatchString(`\bDROP\s+SCHEMA\b`, upper); match {
		return fmt.Errorf("%w: DROP SCHEMA is strictly forbidden in automated updates", ErrDestructiveMigration)
	}

	// 3. Block DROP TABLE (unless it explicitly targets temporary tables)
	dropTableRegex := regexp.MustCompile(`\bDROP\s+TABLE\s+(?:IF\s+EXISTS\s+)?([^\s;]+)`)
	if matches := dropTableRegex.FindAllStringSubmatch(upper, -1); len(matches) > 0 {
		for _, m := range matches {
			tbl := strings.Trim(m[1], `"'`)
			if !strings.HasPrefix(tbl, "TEMP") && !strings.HasPrefix(tbl, "PG_TEMP") {
				return fmt.Errorf("%w: DROP TABLE %s is strictly forbidden during update", ErrDestructiveMigration, tbl)
			}
		}
	}

	// 4. Block TRUNCATE
	if match, _ := regexp.MatchString(`\bTRUNCATE\b`, upper); match {
		return fmt.Errorf("%w: TRUNCATE is strictly forbidden during update", ErrDestructiveMigration)
	}

	// 5. Block unconditional DELETE FROM (without WHERE)
	deleteRegex := regexp.MustCompile(`\bDELETE\s+FROM\s+([A-Za-z0-9_.]+)`)
	if locs := deleteRegex.FindAllStringIndex(upper, -1); len(locs) > 0 {
		for _, loc := range locs {
			stmtTail := upper[loc[0]:]
			semicolon := strings.Index(stmtTail, ";")
			stmt := stmtTail
			if semicolon != -1 {
				stmt = stmtTail[:semicolon]
			}
			if !strings.Contains(stmt, "WHERE") {
				return fmt.Errorf("%w: unconditional DELETE FROM without WHERE clause is strictly forbidden", ErrDestructiveMigration)
			}
		}
	}

	return nil
}
