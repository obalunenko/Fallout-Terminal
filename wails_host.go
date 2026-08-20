package main

import (
	"context"
	"errors"
	"io/fs"
	"sync"
	"time"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

const wailsShutdownTimeout = 5 * time.Second

func init() {
	application.RegisterEvent[domain.ServerInfo](serverInfoEvent)
	application.RegisterEvent[int](clientCountEvent)
	application.RegisterEvent[*domain.PublicHackState](hackStateEvent)
	application.RegisterEvent[*domain.MasterCoordinationState](coordinationStateEvent)
	application.RegisterEvent[SessionStateEvent](sessionStateEvent)
	application.RegisterEvent[PublicAccessSnapshot](publicAccessStatusEvent)
}

func newWailsApplication(frontendAssets fs.FS) *application.App {
	return application.New(wailsApplicationOptions(frontendAssets))
}

func wailsApplicationOptions(frontendAssets fs.FS) application.Options {
	return application.Options{
		Name:        "Fallout Terminal",
		Description: "Fallout Terminal — Master Control",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(frontendAssets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	}
}

func newMasterWindow(host *application.App) *application.WebviewWindow {
	window := host.Window.NewWithOptions(masterWindowOptions())
	registerMasterWindowQuitOnClose(window, host)
	return window
}

type masterWindowCloseRegistrar interface {
	RegisterHook(events.WindowEventType, func(*application.WindowEvent)) func()
	OnWindowEvent(events.WindowEventType, func(*application.WindowEvent)) func()
}

type applicationQuitter interface {
	Quit()
}

func registerMasterWindowQuitOnClose(window masterWindowCloseRegistrar, host applicationQuitter) {
	var quitOnce sync.Once
	requestQuit := func(*application.WindowEvent) {
		quitOnce.Do(func() {
			// Wails v3 beta may close its final Darwin NSWindow without asking
			// NSApplication to terminate. Request application shutdown explicitly
			// before returning control to AppKit so the service cleanup path runs
			// for the red close button/Cmd+W too.
			host.Quit()
		})
	}
	window.RegisterHook(events.Common.WindowClosing, requestQuit)
	// A native/scripted NSWindow close can bypass WindowShouldClose. Observe
	// AppKit's post-close notification as a fallback while sharing the same
	// exactly-once quit intent.
	window.OnWindowEvent(events.Mac.WindowWillClose, requestQuit)
}

func masterWindowOptions() application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
		Title:            "Fallout Terminal — Master Control",
		Width:            1200,
		Height:           780,
		MinWidth:         900,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(11, 13, 10),
		URL:              "/",
	}
}

// wailsServiceRegistrar is the narrow host seam needed after core composition.
// The concrete Wails application stays in root composition and the core App is
// never registered as a frontend service.
type wailsServiceRegistrar interface {
	RegisterService(application.Service)
}

type coreLifecycle interface {
	Start(context.Context) error
	Shutdown(context.Context) error
}

type wailsEventEmitter interface {
	Emit(string, ...any) bool
}

type wailsEventSink struct {
	events wailsEventEmitter
}

func newWailsEventSink(events wailsEventEmitter) *wailsEventSink {
	return &wailsEventSink{events: events}
}

func (sink *wailsEventSink) Emit(name string, payload any) error {
	if sink == nil || sink.events == nil {
		return errors.New("Wails event manager is unavailable")
	}
	sink.events.Emit(name, payload)
	return nil
}

// wailsLifecycleService adapts framework lifecycle callbacks to the unbound
// core. Its method names are Wails lifecycle hooks, not authored bridge calls.
type wailsLifecycleService struct {
	core coreLifecycle
}

func newWailsLifecycleService(core coreLifecycle) *wailsLifecycleService {
	return &wailsLifecycleService{core: core}
}

func (service *wailsLifecycleService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	// Application-owned failures are recorded by the core in RuntimeStatus and
	// leave the master window available to explain the failure. Returning them
	// here would make Wails abort before that status can be presented.
	_ = service.core.Start(ctx)
	return nil
}

func (service *wailsLifecycleService) ServiceShutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), wailsShutdownTimeout)
	defer cancel()
	return service.core.Shutdown(ctx)
}

func registerWailsServices(host wailsServiceRegistrar, core *App) {
	host.RegisterService(application.NewService(newWailsLifecycleService(core)))
	host.RegisterService(application.NewService(newDesktopService(core)))
}
