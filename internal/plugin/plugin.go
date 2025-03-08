// internal/plugin/plugin.go
package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/jasonKoogler/comma/internal/config"
)

// Plugin represents a loadable extension to Comma
type Plugin interface {
	// Initialize is called when the plugin is loaded
	Initialize(ctx *config.AppContext) error

	// Name returns the plugin's name
	Name() string

	// Version returns the plugin's version
	Version() string

	// Shutdown is called when the application is terminating
	Shutdown() error
}

// Hook points where plugins can register callbacks
const (
	HookPreCommit    = "pre-commit"
	HookPostCommit   = "post-commit"
	HookPreGenerate  = "pre-generate"
	HookPostGenerate = "post-generate"
)

// Manager handles plugin loading and execution
type Manager struct {
	plugins     map[string]Plugin
	hooks       map[string][]Hook
	ctx         *config.AppContext
	pluginsDir  string
	initialized bool
	mu          sync.RWMutex
}

// Hook represents a callback function registered by a plugin
type Hook struct {
	PluginName string
	Callback   func(args ...interface{}) error
}

// NewManager creates a new plugin manager
func NewManager(ctx *config.AppContext) *Manager {
	pluginsDir := filepath.Join(ctx.ConfigDir, "plugins")

	// Ensure plugins directory exists
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create plugins directory: %v\n", err)
	}

	return &Manager{
		plugins:    make(map[string]Plugin),
		hooks:      make(map[string][]Hook),
		ctx:        ctx,
		pluginsDir: pluginsDir,
	}
}

// Initialize loads and initializes all available plugins
func (m *Manager) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.initialized {
		return nil
	}

	// List plugin files (*.so)
	files, err := os.ReadDir(m.pluginsDir)
	if err != nil {
		return fmt.Errorf("failed to read plugins directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".so" {
			continue
		}

		pluginPath := filepath.Join(m.pluginsDir, file.Name())
		if err := m.loadPlugin(pluginPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to load plugin %s: %v\n", file.Name(), err)
		}
	}

	m.initialized = true
	return nil
}

// loadPlugin loads a single plugin from the given path
func (m *Manager) loadPlugin(path string) error {
	// Load plugin
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look up exported Plugin symbol
	symPlugin, err := p.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("plugin doesn't export 'Plugin' symbol: %w", err)
	}

	// Assert that the symbol is a Plugin
	plugin, ok := symPlugin.(Plugin)
	if !ok {
		return fmt.Errorf("plugin doesn't implement Plugin interface")
	}

	// Initialize the plugin
	if err := plugin.Initialize(m.ctx); err != nil {
		return fmt.Errorf("plugin initialization failed: %w", err)
	}

	// Register the plugin
	m.plugins[plugin.Name()] = plugin
	fmt.Printf("Loaded plugin: %s v%s\n", plugin.Name(), plugin.Version())

	return nil
}

// RegisterHook registers a callback for a specific hook
func (m *Manager) RegisterHook(hookName, pluginName string, callback func(args ...interface{}) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[pluginName]; !exists {
		return fmt.Errorf("plugin not registered: %s", pluginName)
	}

	hook := Hook{
		PluginName: pluginName,
		Callback:   callback,
	}

	m.hooks[hookName] = append(m.hooks[hookName], hook)
	return nil
}

// ExecuteHook executes all callbacks registered for a specific hook
func (m *Manager) ExecuteHook(hookName string, args ...interface{}) []error {
	m.mu.RLock()
	hooks := m.hooks[hookName]
	m.mu.RUnlock()

	var errors []error

	for _, hook := range hooks {
		if err := hook.Callback(args...); err != nil {
			errors = append(errors, fmt.Errorf("plugin %s hook execution failed: %w", hook.PluginName, err))
		}
	}

	return errors
}

// Shutdown gracefully shuts down all plugins
func (m *Manager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for name, p := range m.plugins {
		if err := p.Shutdown(); err != nil {
			lastErr = fmt.Errorf("failed to shut down plugin %s: %w", name, err)
			fmt.Fprintf(os.Stderr, "Warning: %v\n", lastErr)
		}
	}

	return lastErr
}
