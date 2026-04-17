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

// Convert pointer ([]backupLockRepo) to slice.
// pgBackRest >= v2.59 with active backup: return real per-repo data from lockRepo.
// pgBackRest >= v2.32 without active backup: return stable keys from stanza repo list with zero values.
// pgBackRest < v2.32: return default slice with key=0 and zero values.
// Using stanza repo keys for v2.32-v2.58 is a tradeoff: metrics have value 0 between backups,
// but avoids label flapping when a backup starts.
func convertBackupLockRepoPointerToSlice(lockRepo *[]backupLockRepo, stanzaRepo *[]repo) []backupLockRepo {
	if lockRepo != nil {
		return *lockRepo
	}
	if stanzaRepo != nil {
		repos := make([]backupLockRepo, len(*stanzaRepo))
		for i, r := range *stanzaRepo {
			repos[i] = backupLockRepo{Key: r.Key}
		}
		return repos
	}
	return []backupLockRepo{{}}
}

// Convert empty LSN value label.
func convertEmptyLSNValueLabel(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
