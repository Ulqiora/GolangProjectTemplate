package domain

import "GolangTemplateProject/internal/ports"

func scanner(columns map[string]any, fields []string, scan ports.ScanFunc) error {
	dest := make([]any, 0, len(fields))

	for _, fld := range fields {
		if p, ok := columns[fld]; ok {
			dest = append(dest, p)
		}
	}
	return scan(dest...)
}
