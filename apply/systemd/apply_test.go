package systemd

import (
	"testing"

	"github.com/worldiety/nago-runner/configuration"
)

func TestRefusePurge(t *testing.T) {
	svc := []Service{NewService("abc")}

	if !refusePurge(configuration.Runner{}, svc) {
		t.Fatal("empty configuration must never remove existing services")
	}

	if refusePurge(configuration.Runner{Applications: []configuration.Application{{InstID: "def"}}}, svc) {
		t.Fatal("a non-empty configuration may remove undeclared services")
	}

	if refusePurge(configuration.Runner{}, nil) {
		t.Fatal("nothing to remove, nothing to refuse")
	}
}

func TestIsDeclaredIgnoresCase(t *testing.T) {
	cfg := configuration.Runner{Applications: []configuration.Application{{InstID: "AbC"}}}
	if !isDeclared(cfg, NewService("abc")) {
		t.Fatal("expected case-insensitive match")
	}

	if isDeclared(cfg, NewService("abd")) {
		t.Fatal("unexpected match")
	}
}
