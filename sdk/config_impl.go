package sdk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/percona/percona-backup-mongodb/pbm/config"
	"github.com/percona/percona-backup-mongodb/pbm/connect"
	"github.com/percona/percona-backup-mongodb/pbm/ctrl"
)

type configServiceImpl struct {
	conn connect.Client
	cmds *commandServiceImpl
	log  *slog.Logger
}

var _ ConfigService = (*configServiceImpl)(nil)

func (s *configServiceImpl) Get(ctx context.Context) (*Config, error) {
	cfg, err := config.GetConfig(ctx, s.conn)
	if err != nil {
		if errors.Is(err, config.ErrMissedConfig) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get config: %w", err)
	}

	result := convertConfig(cfg)
	return &result, nil
}

func (s *configServiceImpl) SetYAML(ctx context.Context, r io.Reader) error {
	if err := s.cmds.checkLock(ctx); err != nil {
		return err
	}

	cfg, err := config.Parse(r)
	if err != nil {
		return cleanParseError("invalid config", err)
	}

	s.log.InfoContext(ctx, "setting config")
	if err := config.SetConfig(ctx, s.conn, cfg); err != nil {
		return fmt.Errorf("set config: %w", err)
	}
	return nil
}

func (s *configServiceImpl) GetYAML(ctx context.Context) ([]byte, error) {
	cfg, err := config.GetConfig(ctx, s.conn)
	if err != nil {
		if errors.Is(err, config.ErrMissedConfig) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get config yaml: %w", err)
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal config yaml: %w", err)
	}
	return b, nil
}

func (s *configServiceImpl) ListProfiles(ctx context.Context) ([]StorageProfile, error) {
	profiles, err := config.ListProfiles(ctx, s.conn)
	if err != nil {
		return nil, fmt.Errorf("list profiles: %w", err)
	}

	result := make([]StorageProfile, len(profiles))
	for i := range profiles {
		result[i] = convertStorageProfile(&profiles[i])
	}
	return result, nil
}

func (s *configServiceImpl) GetProfile(ctx context.Context, name string) (*StorageProfile, error) {
	profile, err := config.GetProfile(ctx, s.conn, name)
	if err != nil {
		if errors.Is(err, config.ErrMissedConfigProfile) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get profile %q: %w", name, err)
	}

	result := convertStorageProfile(profile)
	return &result, nil
}

func (s *configServiceImpl) GetProfileYAML(ctx context.Context, name string) ([]byte, error) {
	profile, err := config.GetProfile(ctx, s.conn, name)
	if err != nil {
		if errors.Is(err, config.ErrMissedConfigProfile) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get profile yaml %q: %w", name, err)
	}

	b, err := yaml.Marshal(profile)
	if err != nil {
		return nil, fmt.Errorf("marshal profile yaml: %w", err)
	}
	return b, nil
}

func (s *configServiceImpl) SetProfile(ctx context.Context, name string, r io.Reader) (CommandResult, error) {
	if err := s.cmds.checkLock(ctx); err != nil {
		return CommandResult{}, fmt.Errorf("set profile %q: %w", name, err)
	}

	cfg, err := config.Parse(r)
	if err != nil {
		return CommandResult{}, cleanParseError("invalid profile config", err)
	}

	cmd := AddProfileCommand{Name: name, storage: cfg.Storage}
	pbmCmd, err := convertAddProfileCommandToPBM(cmd)
	if err != nil {
		return CommandResult{}, fmt.Errorf("set profile %q: %w", name, err)
	}

	s.log.InfoContext(ctx, "setting profile", "name", name)
	result, err := s.cmds.dispatch(ctx, pbmCmd)
	if err != nil {
		return CommandResult{}, fmt.Errorf("set profile %q: %w", name, err)
	}
	return result, nil
}

func (s *configServiceImpl) RemoveProfile(ctx context.Context, name string) (CommandResult, error) {
	cmd := RemoveProfileCommand{Name: name}
	if err := s.cmds.validateAndCheckLock(ctx, cmd); err != nil {
		return CommandResult{}, fmt.Errorf("remove profile %q: %w", name, err)
	}

	s.log.InfoContext(ctx, "removing profile", "name", name)
	result, err := s.cmds.dispatch(ctx, convertRemoveProfileCommandToPBM(cmd))
	if err != nil {
		return CommandResult{}, fmt.Errorf("remove profile %q: %w", name, err)
	}
	return result, nil
}

func (s *configServiceImpl) Resync(ctx context.Context, cmd ResyncCommand) (CommandResult, error) {
	if err := s.cmds.validateAndCheckLock(ctx, cmd); err != nil {
		return CommandResult{}, fmt.Errorf("resync: %w", err)
	}

	var pbmCmd ctrl.Cmd
	switch c := cmd.(type) {
	case ResyncMain:
		pbmCmd = convertResyncMainToPBM(c)
	case ResyncProfile:
		pbmCmd = convertResyncProfileToPBM(c)
	case ResyncAllProfiles:
		pbmCmd = convertResyncAllProfilesToPBM(c)
	default:
		panic(fmt.Sprintf("unreachable: unknown ResyncCommand type %T", cmd))
	}

	s.log.InfoContext(ctx, "resyncing storage", "command", fmt.Sprintf("%T", cmd))
	result, err := s.cmds.dispatch(ctx, pbmCmd)
	if err != nil {
		return CommandResult{}, fmt.Errorf("resync: %w", err)
	}
	return result, nil
}

// cleanParseError unwraps a yaml.TypeError from config.Parse errors and
// builds a user-friendly message. The raw yaml.TypeError.Errors entries
// contain Go type names (e.g. "in type config.Config") which are PBM
// internals — we strip those before exposing to consumers.
// TODO(pbm-fix): PBM's config.Parse wraps yaml errors with pkg/errors;
// if that changes, this unwrap may need updating.
func cleanParseError(prefix string, err error) error {
	var typeErr *yaml.TypeError
	if !errors.As(err, &typeErr) {
		return fmt.Errorf("%s: %w", prefix, err)
	}

	cleaned := make([]string, len(typeErr.Errors))
	for i, e := range typeErr.Errors {
		// Strip " in type <package.Type>" suffix added by yaml.v2.
		if idx := strings.LastIndex(e, " in type "); idx >= 0 {
			e = e[:idx]
		}
		cleaned[i] = e
	}
	return fmt.Errorf("%s: %s", prefix, strings.Join(cleaned, "; "))
}
