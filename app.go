package cliutil

import (
	"log/slog"

	"github.com/mikeschinkel/go-dt/appinfo"
)

type App struct {
	Writer  Writer
	Logger  *slog.Logger
	Info    appinfo.AppInfo
	Options Options
	RunFunc func([]string) error
}

type AppArgs struct {
	Writer  Writer
	Logger  *slog.Logger
	Info    appinfo.AppInfo
	Options Options
}

func NewApp(args AppArgs) App {
	return App{
		Writer:  args.Writer,
		Logger:  args.Logger,
		Info:    args.Info,
		Options: args.Options,
	}
}

func (a App) Run(args []string) error {
	if a.RunFunc == nil {
		panic("cliutil.App.Run: No RunFunc specified (RunFunc==nil)")
	}
	return a.RunFunc(args)
}
