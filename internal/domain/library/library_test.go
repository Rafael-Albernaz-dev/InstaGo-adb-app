package library_test

import (
	"testing"

	"github.com/Rafael-Albernaz-dev/InstaGo-adb-app/internal/domain/library"
)

func TestFormatBRL(t *testing.T) {
	cases := []struct {
		cents int64
		want  string
	}{
		{0, "R$ 0,00"},
		{9999, "R$ 99,99"},
		{24990, "R$ 249,90"},
		{125050, "R$ 1.250,50"},
	}
	for _, tc := range cases {
		if got := library.FormatBRL(tc.cents); got != tc.want {
			t.Fatalf("FormatBRL(%d) = %q, want %q", tc.cents, got, tc.want)
		}
	}
}

func TestCompanyProfileValidate(t *testing.T) {
	valid := library.CompanyProfile{
		ID:              library.DefaultCompanyID,
		Brand:           "Rural Conecta",
		InstagramHandle: "ruralconectamg",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid profile, got %v", err)
	}

	invalid := valid
	invalid.Brand = "  "
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected error when brand is empty")
	}
}
