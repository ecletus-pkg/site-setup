package site_setup

import (
	"context"

	"github.com/ecletus/cli"
	"github.com/ecletus/core"
	"github.com/ecletus/ecletus"
	"github.com/ecletus/ecletus/plugin"
	"github.com/ecletus/plug"
	"github.com/ecletus/sites"
	defaultlogger "github.com/moisespsena-go/default-logger"
	errwrap "github.com/moisespsena-go/error-wrap"
	path_helpers "github.com/moisespsena-go/path-helpers"
	"github.com/moisespsena-go/pluggable"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/ecletus/db"
)

var (
	PKG = path_helpers.GetCalledDir()
	log = defaultlogger.GetOrCreateLogger(path_helpers.GetCalledDir())
)

type Plugin struct {
	plug.EventDispatcher
	SitesRegisterKey string
}

func (p *Plugin) RequireOptions() []string {
	return []string{p.SitesRegisterKey}
}

func (p *Plugin) OnRegister() {
	cli.OnRegister(p, func(e *cli.RegisterEvent) {
		var (
			setupCmd      *cobra.Command
			options       = e.Options()
			eclt          = options.GetInterface(ecletus.ECLETUS).(*ecletus.Ecletus)
			SitesRegister = options.GetInterface(p.SitesRegisterKey).(*core.SitesRegister)
			cmd           = &sites.CmdUtils{SitesRegister: SitesRegister}
			dis           = e.PluginDispatcher()
		)
		var cmdGen = cmd.Alone
		if !SitesRegister.Alone {
			cmdGen = cmd.Sites
		}
		setupCmd = cmdGen(&cobra.Command{
			Use:   "site-setup OPTIONS...",
			Short: "Configure site for first use",
		}, func(cmd *cobra.Command, site *core.Site, args []string) (err error) {
			var (
				hooksEnabled, _ = cmd.Flags().GetBool("hooks")
				onlyDbSchema, _ = cmd.Flags().GetBool("db-schema")
				plugins         []*plugin.Plugin
				pluginErr       = func(plug *plugin.Plugin, err error) error {
					return errors.Wrapf(err, "plugin %q", plug.Name)
				}
			)

			if hooksEnabled {
				if err = plugin.Load(".", "SiteSetup", log, pluggable.Options{}, func(plug *plugin.Plugin) error {
					plugins = append(plugins, plug)
					return nil
				}); err != nil {
					return
				}
			}

			log.Infof("Configure site %q:", site.Name())
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			var ctx = context.Background()
			if dryRun {
				ctx = context.WithValue(ctx, db.OptCommitDisabled, true)
			}
			if onlyDbSchema {
				ctx = context.WithValue(ctx, db.OptDataDisabled, true)
			}

			if err = eclt.Container.Init(); err != nil {
				return
			}

			for _, plug := range plugins {
				if h, ok := plug.Obj.(PreMigratePlugin); ok {
					if err = h.PreMigrate(site); err != nil {
						return errwrap.Wrap(pluginErr(plug, err), "pre migrate")
					}
				}
			}
			if err = eclt.Migrate(ctx); err != nil {
				return errwrap.Wrap(err, "Site %q: migrate", site.Name())
			}

			for _, plug := range plugins {
				if h, ok := plug.Obj.(PostMigratePlugin); ok {
					if err = h.PostMigrate(site); err != nil {
						return errwrap.Wrap(pluginErr(plug, err), "post migrate")
					}
				}
			}

			for _, plug := range plugins {
				if h, ok := plug.Obj.(PreSetupPlugin); ok {
					if err = h.PreSetup(site); err != nil {
						return errwrap.Wrap(pluginErr(plug, err), "pre setup")
					}
				}
			}

			if err = Trigger(dis, E_SETUP, site, setupCmd); err != nil {
				return errwrap.Wrap(err, "Site %q: event %q", site.Name(), e.Name())
			}

			for _, plug := range plugins {
				if h, ok := plug.Obj.(PostSetupPlugin); ok {
					if err = h.PostSetup(site); err != nil {
						return errwrap.Wrap(pluginErr(plug, err), "post setup")
					}
				}
			}

			log.Infof("Configure %q done.", site.Name())
			return nil
		})
		flags := setupCmd.Flags()
		flags.Bool("dry-run", false, "commit database changes disable")
		flags.Bool("db-schema", false, "only database schema")
		flags.BoolP("hooks", "H", false, "load hooks")
		Trigger(e.PluginDispatcher(), E_REGISTER, nil, setupCmd)
		e.RootCmd.AddCommand(setupCmd)
	})
}

type PostMigratePlugin interface {
	PostMigrate(site *core.Site) (err error)
}

type PreMigratePlugin interface {
	PreMigrate(site *core.Site) (err error)
}

type PreSetupPlugin interface {
	PreSetup(site *core.Site) (err error)
}

type PostSetupPlugin interface {
	PostSetup(site *core.Site) (err error)
}
