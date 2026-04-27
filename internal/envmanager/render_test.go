package envmanager

import "testing"

func TestRenderDotenvEscapesSpecialCharacters(t *testing.T) {
	t.Parallel()

	rendered, err := RenderDotenv(map[string]string{
		"API_KEY": "line1\nline2\"quoted\"\\slash",
	})
	if err != nil {
		t.Fatalf("render dotenv: %v", err)
	}

	expected := "API_KEY=\"line1\\nline2\\\"quoted\\\"\\\\slash\"\n"
	if rendered != expected {
		t.Fatalf("expected %q, got %q", expected, rendered)
	}
}

func TestParseDotenvSupportsQuotedAndExportedValues(t *testing.T) {
	t.Parallel()

	values, err := ParseDotenv([]byte("export API_KEY=\"line\\nvalue\"\nDATABASE_URL='postgres://localhost/db'\nDEBUG=true # local\nMULTILINE=\"first\nsecond\"\n"))
	if err != nil {
		t.Fatalf("parse dotenv: %v", err)
	}

	if values["API_KEY"] != "line\nvalue" {
		t.Fatalf("unexpected API_KEY: %q", values["API_KEY"])
	}
	if values["DATABASE_URL"] != "postgres://localhost/db" {
		t.Fatalf("unexpected DATABASE_URL: %q", values["DATABASE_URL"])
	}
	if values["DEBUG"] != "true" {
		t.Fatalf("unexpected DEBUG: %q", values["DEBUG"])
	}
	if values["MULTILINE"] != "first\nsecond" {
		t.Fatalf("unexpected MULTILINE: %q", values["MULTILINE"])
	}
}

func TestRenderExportsEscapesSingleQuotes(t *testing.T) {
	t.Parallel()

	rendered, err := RenderExports(map[string]string{
		"API_KEY": "o'hare",
	})
	if err != nil {
		t.Fatalf("render exports: %v", err)
	}

	expected := "export API_KEY='o'\"'\"'hare'\n"
	if rendered != expected {
		t.Fatalf("expected %q, got %q", expected, rendered)
	}
}
