package site_setup

import (
	"github.com/aghape/cli"
	"github.com/aghape/core"
	"github.com/aghape/plug"
	"github.com/aghape/sites"
	"github.com/moisespsena/go-default-logger"
	"github.com/moisespsena/go-error-wrap"
	"github.com/moisespsena/go-path-helpers"
	"github.com/spf13/cobra"
)

var (
	PKG = path_helpers.GetCalledDir()
	log = defaultlogger.NewLogger(path_helpers.GetCalledDir())
)

type Plugin struct {
	plug.EventDispatcher
	SitesReaderKey string
}

func (p *Plugin) RequireOptions() []string {
	return []string{p.SitesReaderKey}
}

func (p *Plugin) OnRegister() {
	cli.OnRegister(p, func(e *cli.RegisterEvent) {
		var (
			setupCmd *cobra.Command

			SitesReader = e.Options().GetInterface(p.SitesReaderKey).(core.SitesReaderInterface)
			cmd         = &sites.CmdUtils{SitesReader: SitesReader}
			dis         = e.PluginDispatcher()
		)
		setupCmd = cmd.Sites(&cobra.Command{
			Use:   "site-setup",
			Short: "Configure site for first use",
		}, func(cmd *cobra.Command, site core.SiteInterface, args []string) error {
			log.Infof("Configure site %q:", site.Name())
			if err := Trigger(dis, E_SETUP, site, setupCmd); err != nil {
				return errwrap.Wrap(err, "Site %q: event %q", site.Name(), e.Name())
			}
			log.Infof("Configure %q done.", site.Name())
			return nil
		})
		Trigger(e.PluginDispatcher(), E_REGISTER, nil, setupCmd)
		e.RootCmd.AddCommand(setupCmd)
	})
}
