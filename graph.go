// Package Enflux is a Go package for building structured, DAG-based data analytics pipelines.
// It introduces the concept of a RoutineGraph: a directed acyclic graph with a single root node and one or more terminal nodes.
// Each node, called a Routine, encapsulates a discrete unit of processing logic.
//
// Enflux is ideal for defining complex, modular workflows where data flows from a root through a network of connected routines.
// It supports branching, fan-out, and fan-in patterns, making it well-suited for real-time analytics, ETL systems, and custom pipeline engines.
//
// Features:
//
// - Directed acyclic graph (DAG) execution model
//
// - Single entrypoint (root node), multiple terminal nodes
//
// - Modular Routine nodes with customizable behavior
//
// - Support for parallel execution paths
//
// - Minimal dependencies and easy integration
package enflux

import (
	"fmt"
	"log/slog"
)

// The RoutineGraph is a directed acyclic graph of routines with a single root node and potentially multiple leaf nodes.
//
// Leaf nodes are defined as routines in the graph with no children routines to pass data to.
//
// It is assumed each routine is unique and has a unique name.
//
// The graph is shut down by cancelling the context of the graph (i.e. the same context is used for all routines).
// Though, it doesn't necessarily have to be the same context for all routine nodes.
type RoutineGraph struct {
	Graph        map[string][]*Routine // No loops, should be a simple graph
	routineOrder []*Routine            // Order in which the routines were added
}

func NewRoutineGraph() *RoutineGraph {
	return &RoutineGraph{
		Graph: make(map[string][]*Routine),
	}
}

// Start will start all routines in the graph.
//
// If withOutputChannels is true, it will return a slice of channels for the leaf routines else nil.
func (rg *RoutineGraph) Start(withOutputChannels bool) ([]chan Data, error) {
	if len(rg.routineOrder) == 0 {
		return nil, fmt.Errorf("empty routine graph")
	}
	for _, routine := range rg.routineOrder {
		routine.Start()
	}
	var outputChannels []chan Data
	if withOutputChannels {
		// Gather all the leaf routines
		leafRoutines := rg.GetLeafRoutines()
		outputChannels = make([]chan Data, 0, len(leafRoutines))
		// Here we just care about the leaves' output channels for data collection
		for _, leafRoutine := range leafRoutines {
			ch := make(chan Data)
			leafRoutine.OutputChannels = append(leafRoutine.OutputChannels, ch)
			outputChannels = append(outputChannels, ch)
		}
	}
	return outputChannels, nil
}

// Adds the relationship between parentRoutine and childRoutine
// implying the flow of data: parentRoutine --> childRoutine.
//
// An error is returned if the relationship violates the directed acyclic graph property.
//
// If parentRoutine is nil, childRoutine is set as root of the graph -- only if the graph is empty, else an error is raised.
func (rg *RoutineGraph) AddRoutine(childRoutine, parentRoutine *Routine) error {
	// If parentRoutine is empty/nil, intention is to set childRoutine as root
	if parentRoutine == nil || parentRoutine.name == EmptyRoutineName {
		// Check if the graph is empty
		if len(rg.routineOrder) > 0 {
			return fmt.Errorf("routineGraph already has a root routine %s, parentRoutine must be provided", rg.routineOrder[0].name)
			// Else, set the childRoutine as root
		} else {
			rg.routineOrder = append(rg.routineOrder, childRoutine)
			rg.Graph[childRoutine.name] = []*Routine{}
			return nil
		}
	}
	// Verify the parentRoutine is already in the graph
	if children, ok := rg.Graph[parentRoutine.name]; !ok {
		return fmt.Errorf("parentRoutine %s is not in the graph", parentRoutine.name)
	} else {
		// Verify that this relationship does not already exist
		for _, child := range children {
			if child.name == childRoutine.name {
				return fmt.Errorf("childRoutine %s is already associated to parentRoutine %s", childRoutine.name, parentRoutine.name)
			}
		}
		// Add the childRoutine to the parentRoutine's list of children
		rg.Graph[parentRoutine.name] = append(rg.Graph[parentRoutine.name], childRoutine)
		// Add the childRoutine to the graph if it's not already there
		if _, ok = rg.Graph[childRoutine.name]; !ok {
			// Initialize the childRoutine in the graph
			rg.routineOrder = append(rg.routineOrder, childRoutine)
			rg.Graph[childRoutine.name] = []*Routine{}
			// Create a channel between the two routines
			ch := make(chan Data)
			// Set input channel of child and output channel of parent
			slog.Info(fmt.Sprintf("setting output channel of %s as input channel of %s", parentRoutine.name, childRoutine.name))
			childRoutine.InputChannel = ch
			parentRoutine.OutputChannels = append(parentRoutine.OutputChannels, ch)
		} else {
			// If the childRoutine is already in the graph check that the relationship childRoutine -> parentRoutine is not already there
			for _, routine := range rg.Graph[childRoutine.name] {
				if routine.name == parentRoutine.name {
					return fmt.Errorf("the relationship %s -> %s already exists, adding the intended relationship creates a loop", childRoutine.name, parentRoutine.name)
				}
			}
			slog.Info(fmt.Sprintf("childRoutine %s already in graph so setting its input channel as output channel of parentRoutine %s", childRoutine.name, parentRoutine.name))
			parentRoutine.OutputChannels = append(parentRoutine.OutputChannels, childRoutine.InputChannel)
		}
	}
	return nil
}

// Returns leaf routine(s).
func (rg *RoutineGraph) GetLeafRoutines() []*Routine {
	leaves := make([]*Routine, 0)
	// Leaves are defined as routines with no children
	for _, routine := range rg.routineOrder {
		if len(rg.Graph[routine.name]) == 0 {
			leaves = append(leaves, routine)
		}
	}
	return leaves
}
