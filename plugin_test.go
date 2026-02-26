package elgon

import "testing"

type samplePlugin struct {
	name string
	init func(*App)
}

func (p samplePlugin) Name() string { return p.name }
func (p samplePlugin) Init(a *App) error {
	if p.init != nil {
		p.init(a)
	}
	return nil
}

func TestRegisterPlugins(t *testing.T) {
	app := New(Config{DisableHealthz: true})
	called := false
	if err := app.RegisterPlugins(samplePlugin{name: "x", init: func(_ *App) { called = true }}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected plugin init call")
	}
	if len(app.Plugins()) != 1 {
		t.Fatalf("expected one plugin, got %d", len(app.Plugins()))
	}
}

func TestRegisterPluginsDuplicate(t *testing.T) {
	app := New(Config{DisableHealthz: true})
	if err := app.RegisterPlugins(samplePlugin{name: "dup"}); err != nil {
		t.Fatal(err)
	}
	if err := app.RegisterPlugins(samplePlugin{name: "dup"}); err == nil {
		t.Fatal("expected duplicate plugin error")
	}
}
