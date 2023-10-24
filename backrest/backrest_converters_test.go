package backrest

import "testing"

func TestConvertBoolToFloat64(t *testing.T) {
	type args struct {
		value bool
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{"ConvertBoolToFloat64True",
			args{true},
			1,
		},
		{"ConvertBoolToFloat64False",
			args{false},
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertBoolToFloat64(tt.args.value); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestConvertBoolPointerToFloat64(t *testing.T) {
	type args struct {
		value *bool
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{"ConvertBoolPointerToFloat64True",
			args{valToPtr(bool(true))},
			1,
		},
		{"ConvertBoolPointerToFloat64False",
			args{valToPtr(bool(false))},
			0,
		},
		{"ConvertBoolPointerToFloat64Nil",
			args{nil},
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertBoolPointerToFloat64(tt.args.value); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestConvertInt64PointerToFloat64(t *testing.T) {
	type args struct {
		value *int64
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{"ConvertInt64PointerToFloat64One",
			args{valToPtr(int64(1))},
			1,
		},
		{"ConvertInt64PointerToFloat64Nil",
			args{nil},
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertInt64PointerToFloat64(tt.args.value); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestConvertAnnotationPointerToFloat64(t *testing.T) {
	type args struct {
		value *annotation
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{"ConvertAnnotationPointerToFloat64NotNil",
			args{valToPtr(annotation{"testkey": "testvalue"})},
			1,
		},
		{"ConvertAnnotationPointerToFloat64Nil",
			args{nil},
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertAnnotationPointerToFloat64(tt.args.value); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestConvertDatabaseRefPointerToFloat(t *testing.T) {
	type args struct {
		value *[]databaseRef
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{"ConvertDatabaseRefPointerToFloatNotNil",
			args{valToPtr([]databaseRef{{"postgres", 13425}})},
			1,
		},
		{"ConvertDatabaseRefPointerToFloatNil",
			args{nil},
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertDatabaseRefPointerToFloat(tt.args.value); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}

func TestConvertEmptyLSNValueLabel(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"ConvertEmptyLSNValueLabelNil",
			args{""},
			"-",
		},
		{"ConvertEmptyLSNValueLabelNotNil",
			args{"0/2000028"},
			"0/2000028",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertEmptyLSNValueLabel(tt.args.value); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}
