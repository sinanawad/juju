// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package externalcontrollerupdater

import (
	"context"
	"io"
	"reflect"
	"time"

	"github.com/juju/clock"
	"github.com/juju/errors"
	"github.com/juju/names/v6"
	"github.com/juju/worker/v4"
	"github.com/juju/worker/v4/catacomb"

	"github.com/juju/juju/api"
	"github.com/juju/juju/api/controller/crosscontroller"
	"github.com/juju/juju/core/crossmodel"
	"github.com/juju/juju/core/watcher"
	internallogger "github.com/juju/juju/internal/logger"
	internalworker "github.com/juju/juju/internal/worker"
	"github.com/juju/juju/rpc"
)

var logger = internallogger.GetLogger("juju.worker.externalcontrollerupdater")

// ExternalControllerUpdaterClient defines the interface for watching changes
// to the local controller's external controller records, and obtaining and
// updating their values. This will communicate only with the local controller.
type ExternalControllerUpdaterClient interface {
	WatchExternalControllers(ctx context.Context) (watcher.StringsWatcher, error)
	ExternalControllerInfo(ctx context.Context, controllerUUID string) (*crossmodel.ControllerInfo, error)
	SetExternalControllerInfo(context.Context, crossmodel.ControllerInfo) error
}

// ExternalControllerWatcherClientCloser extends the ExternalControllerWatcherClient
// interface with a Close method, for closing the API connection associated with
// the client.
type ExternalControllerWatcherClientCloser interface {
	ExternalControllerWatcherClient
	io.Closer
}

// ExternalControllerWatcherClient defines the interface for watching changes
// to and obtaining the current API information for a controller. This will
// communicate with an external controller.
type ExternalControllerWatcherClient interface {
	WatchControllerInfo(ctx context.Context) (watcher.NotifyWatcher, error)
	ControllerInfo(ctx context.Context) (*crosscontroller.ControllerInfo, error)
}

// NewExternalControllerWatcherClientFunc is a function type that
// returns an ExternalControllerWatcherClientCloser, given an
// *api.Info. The api.Info should be for making a controller-only
// connection to a remote/external controller.
type NewExternalControllerWatcherClientFunc func(context.Context, *api.Info) (ExternalControllerWatcherClientCloser, error)

// New returns a new external controller updater worker.
func New(
	externalControllers ExternalControllerUpdaterClient,
	newExternalControllerWatcherClient NewExternalControllerWatcherClientFunc,
	clock clock.Clock,
) (worker.Worker, error) {
	runner, err := worker.NewRunner(worker.RunnerParams{
		Name: "external-controller-updater",
		// One of the controller watchers fails should not prevent the others
		// from running.
		IsFatal: func(error) bool { return false },

		// If the API connection fails, try again in 1 minute.
		RestartDelay: time.Minute,
		Clock:        clock,
		Logger:       internalworker.WrapLogger(logger),
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	w := updaterWorker{
		watchExternalControllers:           externalControllers.WatchExternalControllers,
		externalControllerInfo:             externalControllers.ExternalControllerInfo,
		setExternalControllerInfo:          externalControllers.SetExternalControllerInfo,
		newExternalControllerWatcherClient: newExternalControllerWatcherClient,
		runner:                             runner,
	}
	if err := catacomb.Invoke(catacomb.Plan{
		Name: "external-controller-updater",
		Site: &w.catacomb,
		Work: w.loop,
		Init: []worker.Worker{w.runner},
	}); err != nil {
		return nil, errors.Trace(err)
	}
	return &w, nil
}

type updaterWorker struct {
	catacomb catacomb.Catacomb
	runner   *worker.Runner

	watchExternalControllers           func(ctx context.Context) (watcher.StringsWatcher, error)
	externalControllerInfo             func(ctx context.Context, controllerUUID string) (*crossmodel.ControllerInfo, error)
	setExternalControllerInfo          func(context.Context, crossmodel.ControllerInfo) error
	newExternalControllerWatcherClient NewExternalControllerWatcherClientFunc
}

// Kill is part of the worker.Worker interface.
func (w *updaterWorker) Kill() {
	w.catacomb.Kill(nil)
}

// Wait is part of the worker.Worker interface.
func (w *updaterWorker) Wait() error {
	return w.catacomb.Wait()
}

func (w *updaterWorker) loop() error {
	ctx, cancel := w.scopedContext()
	defer cancel()

	watcher, err := w.watchExternalControllers(ctx)
	if err != nil {
		return errors.Annotate(err, "watching external controllers")
	}
	_ = w.catacomb.Add(watcher)

	watchers := names.NewSet()
	for {
		select {
		case <-w.catacomb.Dying():
			return w.catacomb.ErrDying()

		case ids, ok := <-watcher.Changes():
			if !ok {
				return w.catacomb.ErrDying()
			}

			if len(ids) == 0 {
				continue
			}

			logger.Debugf(ctx, "external controllers changed: %q", ids)
			tags := make([]names.ControllerTag, len(ids))
			for i, id := range ids {
				if !names.IsValidController(id) {
					return errors.Errorf("%q is not a valid controller tag", id)
				}
				tags[i] = names.NewControllerTag(id)
			}

			for _, tag := range tags {
				// We're informed when an external controller
				// is added or removed, so treat as a toggle.
				if watchers.Contains(tag) {
					logger.Infof(ctx, "stopping watcher for external controller %q", tag.Id())
					_ = w.runner.StopAndRemoveWorker(tag.Id(), w.catacomb.Dying())
					watchers.Remove(tag)
					continue
				}
				logger.Infof(ctx, "starting watcher for external controller %q", tag.Id())
				watchers.Add(tag)
				if err := w.runner.StartWorker(ctx, tag.Id(), func(ctx context.Context) (worker.Worker, error) {
					return newControllerWatcher(
						tag,
						w.setExternalControllerInfo,
						w.externalControllerInfo,
						w.newExternalControllerWatcherClient,
					)
				}); err != nil {
					return errors.Annotatef(err, "starting watcher for external controller %q", tag.Id())
				}
			}
		}
	}
}

func (w *updaterWorker) scopedContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(w.catacomb.Context(context.Background()))
}

// controllerWatcher is a worker that watches for changes to the external
// controller with the given tag. The external controller must be known
// to the local controller.
type controllerWatcher struct {
	catacomb catacomb.Catacomb

	tag                                names.ControllerTag
	setExternalControllerInfo          func(context.Context, crossmodel.ControllerInfo) error
	externalControllerInfo             func(ctx context.Context, controllerUUID string) (*crossmodel.ControllerInfo, error)
	newExternalControllerWatcherClient NewExternalControllerWatcherClientFunc
}

func newControllerWatcher(
	tag names.ControllerTag,
	setExternalControllerInfo func(context.Context, crossmodel.ControllerInfo) error,
	externalControllerInfo func(ctx context.Context, controllerUUID string) (*crossmodel.ControllerInfo, error),
	newExternalControllerWatcherClient NewExternalControllerWatcherClientFunc,
) (*controllerWatcher, error) {
	cw := &controllerWatcher{
		tag:                                tag,
		setExternalControllerInfo:          setExternalControllerInfo,
		externalControllerInfo:             externalControllerInfo,
		newExternalControllerWatcherClient: newExternalControllerWatcherClient,
	}

	if err := catacomb.Invoke(catacomb.Plan{
		Name: "external-controller-watcher",
		Site: &cw.catacomb,
		Work: cw.loop,
	}); err != nil {
		return nil, errors.Trace(err)
	}

	return cw, nil
}

// Kill is part of the worker.Worker interface.
func (w *controllerWatcher) Kill() {
	w.catacomb.Kill(nil)
}

// Wait is part of the worker.Worker interface.
func (w *controllerWatcher) Wait() error {
	err := w.catacomb.Wait()
	if errors.Cause(err) == rpc.ErrShutdown {
		// RPC shutdown errors need to be ignored.
		return nil
	}
	return err
}

func (w *controllerWatcher) loop() error {
	ctx, cancel := w.scopedContext()
	defer cancel()

	// We get the API info from the local controller initially.
	info, err := w.externalControllerInfo(ctx, w.tag.Id())
	if errors.Is(err, errors.NotFound) {
		return nil
	} else if err != nil {
		return errors.Annotate(err, "getting cached external controller info")
	}
	logger.Debugf(ctx, "controller info for controller %q: %v", w.tag.Id(), info)

	var nw watcher.NotifyWatcher
	var client ExternalControllerWatcherClientCloser
	defer func() {
		if client != nil {
			_ = client.Close()
		}
	}()

	for {
		if client == nil {
			apiInfo := &api.Info{
				Addrs:  info.Addrs,
				CACert: info.CACert,
				Tag:    names.NewUserTag(api.AnonymousUsername),
			}
			client, nw, err = w.connectAndWatch(ctx, apiInfo)
			if err == w.catacomb.ErrDying() {
				return err
			} else if err != nil {
				return errors.Trace(err)
			}
			_ = w.catacomb.Add(nw)
		}

		select {
		case <-w.catacomb.Dying():
			return w.catacomb.ErrDying()
		case _, ok := <-nw.Changes():
			if !ok {
				return w.catacomb.ErrDying()
			}

			newInfo, err := client.ControllerInfo(ctx)
			if err != nil {
				return errors.Annotate(err, "getting external controller info")
			}
			if reflect.DeepEqual(newInfo.Addrs, info.Addrs) {
				continue
			}

			// API addresses have changed. Save the details to the
			// local controller and stop the existing notify watcher
			// and set it to nil, so we'll restart it with the new
			// addresses.
			if err := w.setExternalControllerInfo(ctx, crossmodel.ControllerInfo{
				ControllerUUID: w.tag.Id(),
				Alias:          info.Alias,
				Addrs:          newInfo.Addrs,
				CACert:         info.CACert,
			}); err != nil {
				return errors.Annotate(err, "caching external controller info")
			}

			logger.Infof(ctx, "new controller info for controller %q: addresses changed: new %v, prev %v", w.tag.Id(), newInfo.Addrs, info.Addrs)

			// Set the new addresses in the info struct so that
			// we can reuse it in the next iteration.
			info.Addrs = newInfo.Addrs

			if err := worker.Stop(nw); err != nil {
				return errors.Trace(err)
			}
			if err := client.Close(); err != nil {
				return errors.Trace(err)
			}
			client = nil
			nw = nil
		}
	}
}

// connectAndWatch connects to the specified controller and watches for changes.
// It aborts if signalled, which prevents the watcher loop from blocking any shutdown
// of the watcher the may be requested by the parent worker.
func (w *controllerWatcher) connectAndWatch(ctx context.Context, apiInfo *api.Info) (ExternalControllerWatcherClientCloser, watcher.NotifyWatcher, error) {
	type result struct {
		client ExternalControllerWatcherClientCloser
		nw     watcher.NotifyWatcher
	}

	response := make(chan result)
	errs := make(chan error)

	go func() {
		client, err := w.newExternalControllerWatcherClient(ctx, apiInfo)
		if err != nil {
			select {
			case <-ctx.Done():
			case errs <- errors.Annotate(err, "getting external controller client"):
			}
			return
		}

		nw, err := client.WatchControllerInfo(ctx)
		if err != nil {
			_ = client.Close()
			select {
			case <-ctx.Done():
			case errs <- errors.Annotate(err, "watching external controller"):
			}
			return
		}

		select {
		case <-ctx.Done():
			_ = client.Close()
		case response <- result{client: client, nw: nw}:
		}
	}()

	select {
	case <-ctx.Done():
		return nil, nil, w.catacomb.ErrDying()
	case err := <-errs:
		return nil, nil, errors.Trace(err)
	case r := <-response:
		return r.client, r.nw, nil
	}
}

func (w *controllerWatcher) scopedContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(w.catacomb.Context(context.Background()))
}
