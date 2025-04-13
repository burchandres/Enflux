# Enflux
Enflux is a Go package for building structured, DAG-based data analytics pipelines.
It introduces the concept of a RoutineGraph: a directed acyclic graph with a single root node and one or more terminal nodes.
Each node, called a Routine, encapsulates a discrete unit of processing logic.

Enflux is ideal for defining complex, modular workflows where data flows from a root through a network of connected routines.
It supports branching, fan-out, and fan-in patterns, making it well-suited for real-time analytics, ETL systems, and custom pipeline engines.

Features:
  - Directed acyclic graph (DAG) execution model
  - Single entrypoint (root node), multiple terminal nodes
  - Modular Routine nodes with customizable behavior
  - Support for parallel execution paths
  - Support for parallel execution on a Routine scale
  - Minimal dependencies and easy integration

# Install
To install the package run 

```zsh
go get github.com/burchandres/enflux
```

# Usage
The `Data` interface only requires that you implement an `IsValid()` method which returns a bool. If the output is `false` then that object is no longer passed along the pipeline and drops at the node it's read at.  The tests provide simple examples of what type of structs the `Data` interface could be.

Very similar to what's in the tests, an example is provided below for a potential use case.

```go

// Example implementation of a struct that implements the Data interface
type MathData struct {
  x int
}

func (m MathData) IsValid() bool {return true}

// Example implementation of a RoutineFunc
type MultiplyFunc struct {
  x int
}

func (m MultiplyFunc) Run(data Data) Data {
  // assert that we get the input we want
  input, ok := data.type(MathData)
  if !ok {
    return InValidData{} // included struct whose IsValid method returns false
  }
  return MathData{x: m.x * input.x}
}

// A separate, yet very similar, RoutineFunc implementation
type AddFunc struct {
  x int
}

func (m AddFunc) Run(data Data) Data {
  input, ok := data.type(MathData)
  if !ok {
    return InvalidData{}
  }
  return MathData{x: m.x + input.x}
}
```

Then, generating the graph would follow some pattern like so:

```go
rg := NewRoutineGraph()
ctx, cancelFunc := context.WithCancel(context.Background())

add1Routine := NewRoutine(
  WithName("add-one"),
  WithFunc(AddFunc{x: 1}),
  WithContext(ctx),
)
multiply2Routine := NewRoutine(
  WithName("multiply-two"),
  WithFunc(MultiplyFunc{x: 2}),
  WithContext(ctx),
)

if err := rg.AddRoutine(add1Routine, nil); err != nil {
  panic() // if it's not vital to program's run then maybe don't panic
}
if err = rg.AddRoutine(multiply2Routine, add1Routine); err != nil {
  panic()
}

// Setup input channel
inputCh := make(chan Data)
add1Routine.InputChannel = inputCh
```

With the above code, one pattern of reading data off would be:

```go
// Start the routineGraph
withOutputChannels := true
// if given true then gives an array of channels
// where there is a one-to-one correspondonce of channels to terminal nodes
// and the channels are ordered same way terminal nodes were added to the graph
outputChannels, err := rg.Start(withOutputChannels)
if err != nil {
  panic()
}

var wg sync.WaitGroup

wg.Add(1)
go func() {
  defer wg.Done()
  // not looping through all channels since there's only one
  for {
    select {
    case <- ctx.Done():
      slog.Info("context cancelled, done reading off output")
      return
    case data := <-outputChannels[0]:
      output, ok := data.type(MathData)
      if !ok {
        slog.Error("invalid data type returned from terminal node")
      }
      slog.Info("received output", "output", output.x)
    }
  }
}()
```
Cancelling the context would shutdown the graph and stop the flow of data.
