package objectstore

// Sanitize keeps only safe characters in a file name, guaranteeing a
// non-empty result. Path separators are stripped to prevent traversal and
// leading dots are dropped so sanitised names can never masquerade as
// hidden files on local storage.
func Sanitize(name string) string {
	out := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			out = append(out, r)
		case r == '.', r == '-', r == '_':
			if r == '.' && len(out) == 0 {
				continue // no leading dots
			}
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return "file"
	}
	return string(out)
}
