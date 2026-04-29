package provider

import "testing"

type providerTestRenderable struct{}

func (providerTestRenderable) Render() string { return "rendered" }

func TestProvider_PublicContracts(t *testing.T) {
	spec := ElementSpec{Tag: "button", Attrs: map[string]string{"type": "submit"}, Text: "Save"}
	if spec.Tag != "button" || spec.Attrs["type"] != "submit" {
		t.Fatalf("unexpected element spec: %+v", spec)
	}
	var renderable Renderable = providerTestRenderable{}
	if renderable.Render() != "rendered" {
		t.Fatal("expected renderable implementation to return rendered content")
	}
}
