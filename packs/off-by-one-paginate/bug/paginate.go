package paginate

func Paginate(items []string, page, pageSize int) []string {
	if len(items) == 0 {
		return nil
	}
	if page < 1 {
		page = 1
	}
	start := (page - 1) * pageSize
	if start >= len(items) {
		return nil
	}
	end := start + pageSize
	return items[start:end]
}
