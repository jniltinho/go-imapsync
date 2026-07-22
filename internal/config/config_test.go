package config

import (
	"testing"

	"go-imapsync/internal/secret"
)

func TestValidateMissing(t *testing.T) {
	var c Config
	c.Defaults()
	if err := c.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateOK(t *testing.T) {
	c := Config{
		Host1: Side{Host: "a.example", User: "u1", Password: secret.New("p1"), SSL: true},
		Host2: Side{Host: "b.example", User: "u2", Password: secret.New("p2"), SSL: true},
	}
	c.Defaults()
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.Host1.Port != 993 || c.Host2.Port != 993 {
		t.Fatalf("ports = %d/%d, want 993", c.Host1.Port, c.Host2.Port)
	}
	if len(c.UseHeader) != 2 {
		t.Fatalf("UseHeader = %v", c.UseHeader)
	}
}

func TestDefaultPortPlain(t *testing.T) {
	c := Config{
		Host1: Side{Host: "a", User: "u", Password: secret.New("p"), SSL: false, TLS: true},
		Host2: Side{Host: "b", User: "u", Password: secret.New("p"), SSL: false, TLS: true},
	}
	c.Defaults()
	if c.Host1.Port != 143 {
		t.Fatalf("port = %d, want 143", c.Host1.Port)
	}
}
