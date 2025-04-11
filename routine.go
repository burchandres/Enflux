package enflux

import (
	"context"
	"log/slog"
)

const EmptyRoutineName = "empty-routine"

type RoutineOptFunc func(o *RoutineOpts)

/***************************
	Main Routine Logic
***************************/

// Data represents a struct that contains the data that is passed between routines.
type Data interface {
	IsValid() bool // Checks if the data is valid
}

type InvalidData struct{}

func (i InvalidData) IsValid() bool { return false }

type Routine struct {
	RoutineOpts
	InputChannel   chan Data
	OutputChannels []chan Data
}

// Default configuration of the Routine is:
// 
// - name: EmptyRoutineName
//
// - ctx: context.Background()
//  
// - scale: 1
//
// - RoutineFunc: IdentityFunc{}
func NewRoutine(opts ...RoutineOptFunc) *Routine {
	o := defaultRoutineOpts()
	for _, opt := range opts {
		opt(o)
	}
	return &Routine{
		RoutineOpts: *o,
	}
}

// Run's the logic that the routine is responsible for.
//
// Routinely checks if the provided context has been cancelled for exiting.
// Channels are never closed by any routine as we could have a many-to-one relationship, so context cancellation is used for shutdown.
//
// If using a routine by itself the input and output channels are exposed to the user for setting and getting.
//
// Spins up r.scale (set with WithScale() optFunc) many goroutines to read off InputChannel
func (r *Routine) Start() {
	for range r.scale {
		go func() {
			slog.Info("starting...", "routine", r.name)
			for {
				select {
				case <-r.ctx.Done():
					slog.Info("exiting...", "routine", r.name)
					// Input channel closure should be handled by the sender
					return
				case input, ok := <-r.InputChannel:
					if !ok && input == nil {
						slog.Debug("input channel closed and no more data is being processed, exiting...", "routine", r.name)
						return
					}
					slog.Debug("processing input...", "input", input, "routine", r.name)
					// Continue running the routine
					output := r.Run(input)
					if output.IsValid() {
						// Send the output to all output channels
						for _, outputChannel := range r.OutputChannels {
							outputChannel <- output
						}
					} else {
						slog.Warn("invalid output", "output", output, "routine", r.name)
					}
				}
			}
		}()
	}
}

/***********************************
	Routine Configuration Code
***********************************/

// RoutineParams represents a struct that contains the parameters that Run() needs to run.
type RoutineFunc interface {
	Run(Data) Data // Calls the logic that the routine is responisble for
}

type IdentityFunc []struct{}

func (i IdentityFunc) Run(data Data) Data { return data }

type RoutineOpts struct {
	ctx context.Context
	RoutineFunc
	name string
	scale int
}

func defaultRoutineOpts() *RoutineOpts {
	return &RoutineOpts{
		name:        EmptyRoutineName,
		RoutineFunc: IdentityFunc{},
		ctx:         context.Background(),
		scale:       1,
	}
}

// Used to specify the struct whose Run() method houses the routine's core logic
func WithFunc(routineFunc RoutineFunc) RoutineOptFunc {
	return func(o *RoutineOpts) {
		o.RoutineFunc = routineFunc
	}
}

// Used to specify the unique name of the routine
func WithName(name string) RoutineOptFunc {
	return func(o *RoutineOpts) {
		o.name = name
	}
}

// Used to set the context of the routine
func WithContext(ctx context.Context) RoutineOptFunc {
	return func(o *RoutineOpts) {
		o.ctx = ctx
	}
}

// Used to set the number of goroutines the routine spins up to ingest incoming data
func WithScale(scale int) RoutineOptFunc {
	return func(o *RoutineOpts) {
		o.scale = scale
	}
}
