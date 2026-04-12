package tui

import (
	"context"
	"errors"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"golang.org/x/sync/errgroup"

	sdk "github.com/jcechace/pbmate/sdk/v2"
)

// configData holds the result of a single config poll cycle.
type configData struct {
	config   *sdk.Config
	yaml     []byte
	profiles []sdk.StorageProfile
	err      error
}

// configDataMsg wraps configData as a BubbleTea message.
type configDataMsg struct{ configData }

// profileYAMLMsg carries a lazily-fetched profile YAML.
type profileYAMLMsg struct {
	name string
	yaml []byte
	err  error
}

// fetchConfigCmd returns a tea.Cmd that fetches config data concurrently.
func fetchConfigCmd(ctx context.Context, client *sdk.Client) tea.Cmd {
	return func() tea.Msg {
		var d configData
		var errs firstErrCollector

		g, gctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			v, err := client.Config.Get(gctx)
			d.config = v
			if err != nil && !errors.Is(err, sdk.ErrNotFound) {
				errs.set(err)
			}
			return nil
		})

		g.Go(func() error {
			v, err := client.Config.GetYAML(gctx)
			d.yaml = v
			if err != nil && !errors.Is(err, sdk.ErrNotFound) {
				errs.set(err)
			}
			return nil
		})

		g.Go(func() error {
			v, err := client.Config.ListProfiles(gctx)
			d.profiles = v
			errs.set(err)
			return nil
		})

		_ = g.Wait()
		d.err = errs.err
		return configDataMsg{d}
	}
}

// fetchProfileYAMLCmd returns a tea.Cmd that fetches the YAML for a
// single storage profile by name.
func fetchProfileYAMLCmd(ctx context.Context, client *sdk.Client, name string) tea.Cmd {
	return func() tea.Msg {
		yaml, err := client.Config.GetProfileYAML(ctx, name)
		return profileYAMLMsg{name: name, yaml: yaml, err: err}
	}
}

// applyConfigCmd returns a tea.Cmd that reads a YAML file and applies it
// as the main PBM configuration.
func applyConfigCmd(ctx context.Context, client *sdk.Client, filePath string) tea.Cmd {
	return func() tea.Msg {
		f, err := os.Open(filePath)
		if err != nil {
			return actionResultMsg{action: "apply config", err: fmt.Errorf("open %s: %w", filePath, err)}
		}
		defer func() { _ = f.Close() }()

		err = client.Config.SetYAML(ctx, f)
		return actionResultMsg{action: "apply config", err: err}
	}
}

// applyProfileCmd returns a tea.Cmd that reads a YAML file and applies it
// to a named storage profile (create or replace).
func applyProfileCmd(ctx context.Context, client *sdk.Client, name, filePath, action string) tea.Cmd {
	return func() tea.Msg {
		f, err := os.Open(filePath)
		if err != nil {
			return actionResultMsg{action: action, err: fmt.Errorf("open %s: %w", filePath, err)}
		}
		defer func() { _ = f.Close() }()

		_, err = client.Config.SetProfile(ctx, name, f)
		return actionResultMsg{action: action, err: err}
	}
}

// resyncCmd returns a tea.Cmd that dispatches a resync command to the SDK.
func resyncCmd(ctx context.Context, client *sdk.Client, cmd sdk.ResyncCommand) tea.Cmd {
	return func() tea.Msg {
		_, err := client.Config.Resync(ctx, cmd)
		return actionResultMsg{action: "resync", err: err}
	}
}

// removeProfileCmd returns a tea.Cmd that removes a named storage profile.
func removeProfileCmd(ctx context.Context, client *sdk.Client, name string) tea.Cmd {
	return func() tea.Msg {
		_, err := client.Config.RemoveProfile(ctx, name)
		return actionResultMsg{action: "remove profile", err: err}
	}
}
