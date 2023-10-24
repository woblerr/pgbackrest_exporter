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

// Convert empty LSN value label.
func convertEmptyLSNValueLabel(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
