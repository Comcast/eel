/**
 * Copyright 2015 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package jtl

import (
	"strconv"

	. "github.com/Comcast/eel/eel/util"
)

// Worker is a worker in the pool
type Worker struct {
	id          int
	work        chan *WorkRequest
	WorkerQueue chan chan *WorkRequest
	quitChan    chan bool
}

// WorkRequest is a work request
type WorkRequest struct {
	Message string
	Ctx     Context
}

// WorkDispatcher dispatches work requests to workers in the pool using channels
type WorkDispatcher struct {
	WorkQueue   chan *WorkRequest
	WorkerQueue chan chan *WorkRequest
	workers     []*Worker
	quitChan    chan bool
}

// NewWorker creates a new worker
func NewWorker(id int, workerQueue chan chan *WorkRequest) *Worker {
	worker := Worker{
		id:          id,
		work:        make(chan *WorkRequest),
		WorkerQueue: workerQueue,
		quitChan:    make(chan bool),
	}
	return &worker
}

func GetWorkDispatcher(ctx Context) *WorkDispatcher {
	if ctx.Value(EelDispatcher) != nil {
		return ctx.Value(EelDispatcher).(*WorkDispatcher)
	}
	return nil
}

// Start starts a worker which will then listen on its private channel for work requests
func (w *Worker) Start() {
	go func() {
		for {
			w.WorkerQueue <- w.work
			select {
			case work := <-w.work:
				stats := work.Ctx.Value(EelTotalStats).(*ServiceStats)
				//w.ctx.Log.Info("action", "received_work", "id", strconv.Itoa(w.id))
				msg, err := NewJDocFromString(work.Message)
				if err != nil {
					work.Ctx.Log().Error("status", "400", "action", "rejected", "error_type", "rejected", "cause", "invalid_json", "error", err.Error(), "content", work.Message)
					stats.IncErrors()
				} else {
					handleEvent(work.Ctx, stats, msg, work.Message, false, false)
				}
				//w.ctx.Log.Info("action", "handled_work", "id", strconv.Itoa(w.id))
			case <-w.quitChan:
				Gctx.Log().Info("action", "stopping_worker", "id", strconv.Itoa(w.id))
				return
			}
		}
	}()
}

// Stop stops a worker via quit channel
func (w *Worker) Stop() {
	go func() {
		w.quitChan <- true
	}()
}

// NewWorkDispatcher creates a new worker pool with nworkers workers and a work queue depth of queueDepth
func NewWorkDispatcher(nworkers int, queueDepth int) *WorkDispatcher {
	disp := new(WorkDispatcher)
	disp.WorkQueue = make(chan *WorkRequest, queueDepth)
	disp.WorkerQueue = make(chan chan *WorkRequest, nworkers)
	disp.workers = make([]*Worker, nworkers)
	disp.quitChan = make(chan bool)
	return disp
}

// Start starts the event loop of a new work dispatcher
func (disp *WorkDispatcher) Start(ctx Context) {
	ctx.Log().Info("action", "starting_workers", "count", len(disp.workers))
	for i := 0; i < len(disp.workers); i++ {
		disp.workers[i] = NewWorker(i, disp.WorkerQueue)
		disp.workers[i].Start()
	}
	go func() {
		for {
			select {
			case work := <-disp.WorkQueue:
				//ctx.Log.Info("action", "received_work_request")
				//go func() {
				worker := <-disp.WorkerQueue
				//ctx.Log.Info("action", "dispatched_work_request")
				worker <- work
				//}()
			case <-disp.quitChan:
				return
			}
		}
	}()
}

// Stop stops the worker pool
func (disp *WorkDispatcher) Stop(ctx Context) {
	if disp.workers != nil && disp.quitChan != nil {
		ctx.Log().Info("action", "stopping_workers", "count", len(disp.workers))
		for _, w := range disp.workers {
			w.Stop()
		}
		disp.quitChan <- true
	}
}
