package mutation

import (
	"go/token"
	"testing"
)

func TestStringToToken(t *testing.T) {
	tests := []struct {
		input    string
		expected token.Token
	}{
		// Arithmetic
		{"+", token.ADD},
		{"-", token.SUB},
		{"*", token.MUL},
		{"/", token.QUO},
		{"%", token.REM},
		{"++", token.INC},
		{"--", token.DEC},

		// Arithmetic assignment
		{"+=", token.ADD_ASSIGN},
		{"-=", token.SUB_ASSIGN},
		{"*=", token.MUL_ASSIGN},
		{"/=", token.QUO_ASSIGN},

		// Conditional
		{"==", token.EQL},
		{"!=", token.NEQ},
		{"<", token.LSS},
		{"<=", token.LEQ},
		{">", token.GTR},
		{">=", token.GEQ},

		// Logical
		{"&&", token.LAND},
		{"||", token.LOR},

		// Bitwise
		{"&", token.AND},
		{"|", token.OR},
		{"^", token.XOR},
		{"&^", token.AND_NOT},
		{"<<", token.SHL},
		{">>", token.SHR},

		// Bitwise assignment
		{"&=", token.AND_ASSIGN},
		{"|=", token.OR_ASSIGN},
		{"^=", token.XOR_ASSIGN},
		{"<<=", token.SHL_ASSIGN},
		{">>=", token.SHR_ASSIGN},

		// Invalid
		{"invalid", token.ILLEGAL},
		{"", token.ILLEGAL},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := stringToToken(tt.input)
			if result != tt.expected {
				t.Errorf("stringToToken(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
