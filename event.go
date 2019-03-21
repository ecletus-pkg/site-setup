package site_setup

import (
	"github.com/ecletus/core"
	"github.com/ecletus/plug"
	"github.com/spf13/cobra"
)

var (
	E_SETUP    = PKG + ".setup"
	E_REGISTER = PKG + ".register"
)

type SiteSetupEvent struct {
	plug.PluginEventInterface
	Site     core.SiteInterface
	SetupCMD *cobra.Command
}

func OnSetup(dis plug.EventDispatcherInterface, cb func(e *SiteSetupEvent) error) {
	_ = dis.OnE(E_SETUP, func(e plug.EventInterface) error {
		return cb(e.(*SiteSetupEvent))
	})
}

func OnRegister(dis plug.EventDispatcherInterface, cb func(e *SiteSetupEvent)) {
	dis.On(E_REGISTER, func(e plug.EventInterface) {
		cb(e.(*SiteSetupEvent))
	})
}

func Trigger(dis plug.PluginEventDispatcherInterface, eventName string, site core.SiteInterface, cmd *cobra.Command) error {
	return dis.TriggerPlugins(&SiteSetupEvent{plug.NewPluginEvent(eventName), site, cmd})
}
