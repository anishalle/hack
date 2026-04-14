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
