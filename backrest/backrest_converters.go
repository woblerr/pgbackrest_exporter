package backrest

// Convert bool to float64.
func convertBoolToFloat64(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

// Convert pointer (bool) to float64.
func convertBoolPointerToFloat64(value *bool) float64 {
	if value != nil {
		return convertBoolToFloat64(*value)
	}
	return 0
}

// Convert pointer (int64) to float64.
func convertInt64PointerToFloat64(value *int64) float64 {
	if value != nil {
		return float64(*value)
	}
	return 0
}

// Convert pointer (annotation) to float64.
func convertAnnotationPointerToFloat64(value *annotation) float64 {
	if value != nil {
		return float64(len(*value))
	}
	return 0
}

// Convert pointer (databaseRef) to float64.
func convertDatabaseRefPointerToFloat(value *[]databaseRef) float64 {
	if value != nil {
		return float64(len(*value))
	}
	return 0
}

// Convert pointer ([]lockBackupRepo) to slice.
// Decision matrix:
//   - pgBackRest < v2.32: stanzaRepo and lockRepo are absent; return key=0 with zero values.
//   - pgBackRest v2.32-v2.58: stanzaRepo is present and lockRepo is absent; return every configured repo with zero values.
//   - pgBackRest >= v2.59 without an active backup: stanzaRepo is present and lockRepo is absent; return every configured repo with zero values.
//   - pgBackRest >= v2.59 with an active backup: stanzaRepo and lockRepo are present; return every configured repo and overlay active backup progress by key.
//   - lockRepo present without stanzaRepo is inconsistent input; use the pgBackRest < v2.32 fallback.
//
// Returning every configured repo keeps the metric label set stable when only a subset has an active backup.
func convertLockBackupRepoPointerToSlice(lockRepo *[]lockBackupRepo, stanzaRepo *[]repo) []lockBackupRepo {
	if stanzaRepo == nil {
		return []lockBackupRepo{{}}
	}

	repos := make([]lockBackupRepo, 0, len(*stanzaRepo))
	repoIndex := make(map[int]int, len(*stanzaRepo))
	for _, r := range *stanzaRepo {
		repoIndex[r.Key] = len(repos)
		repos = append(repos, lockBackupRepo{Key: r.Key})
	}

	if lockRepo != nil {
		for _, lock := range *lockRepo {
			if index, ok := repoIndex[lock.Key]; ok {
				repos[index] = lock
			}
		}
	}

	return repos
}

// Convert empty LSN value label.
func convertEmptyLSNValueLabel(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
