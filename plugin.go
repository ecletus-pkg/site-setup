package site_setup

import (
	"context"

	"github.com/ecletus/cli"
	"github.com/ecletus/core"
	"github.com/ecletus/ecletus"
	"github.com/ecletus/plug"
	"github.com/ecletus/sites"
	defaultlogger "github.com/moisespsena-go/default-logger"
	errwrap "github.com/moisespsena-go/error-wrap"
	path_helpers "github.com/moisespsena-go/path-helpers"
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
			Use:   "site-setup OPTIONS..",
			Short: "Configure site for first use",
		}, func(cmd *cobra.Command, site *core.Site, args []string) (err error) {
			log.Infof("Configure site %q:", site.Name())
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			var ctx = context.Background()
			if dryRun {
				ctx = context.WithValue(ctx, db.OptCommitDisabled, true)
			}
			if err = eclt.Migrate(ctx); err != nil {
				return errwrap.Wrap(err, "Site %q: migrate", site.Name())
			}
			if err = Trigger(dis, E_SETUP, site, setupCmd); err != nil {
				return errwrap.Wrap(err, "Site %q: event %q", site.Name(), e.Name())
			}
			log.Infof("Configure %q done.", site.Name())
			return nil
		})
		setupCmd.Flags().Bool("dry-run", false, "commit database changes disable")
		Trigger(e.PluginDispatcher(), E_REGISTER, nil, setupCmd)
		e.RootCmd.AddCommand(setupCmd)
	})
}
