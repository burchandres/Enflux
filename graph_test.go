package enflux

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTwoChainGraph(t *testing.T) {
	// Create a new routine graph
	rg := NewRoutineGraph()
	// Create ctx and cancel func
	ctx, cancelFunc := context.WithCancel(context.Background())

	// Create add routine
	add1Routine := NewRoutine(
		WithName("add-one"),
		WithFunc(AddFunc{x: 1}),
		WithContext(ctx),
	)
	// Setup input channel for add1Routine
	addInputChannel := make(chan Data)
	add1Routine.InputChannel = addInputChannel
	// Create multiply routine
	multiply2Routine := NewRoutine(
		WithName("multiply-two"),
		WithFunc(MultiplyFunc{x: 2}),
		WithContext(ctx),
	)

	// Set add-one as root routine
	err := rg.AddRoutine(add1Routine, nil)
	assert.NoError(t, err)
	// Check that the graph has the add1Routine
	assert.Equal(t, 1, len(rg.routineOrder))
	assert.Equal(t, "add-one", rg.routineOrder[0].name)
	assert.Equal(t, 1, len(rg.Graph))
	assert.Equal(t, 0, len(rg.Graph["add-one"]))
	// Set multiply-two as child of add-one and leaf routine
	err = rg.AddRoutine(multiply2Routine, add1Routine)
	assert.NoError(t, err)
	// Check that the graph has the multiply2Routine
	assert.Equal(t, 2, len(rg.routineOrder))
	assert.Equal(t, "multiply-two", rg.routineOrder[1].name)
	assert.Equal(t, 2, len(rg.Graph))
	assert.Equal(t, 1, len(rg.Graph["add-one"]))
	assert.Equal(t, 0, len(rg.Graph["multiply-two"]))

	// Start the graph
	outputChannels, err := rg.Start(true)
	assert.NoError(t, err)
	// Only expect one output channel
	assert.Equal(t, 1, len(outputChannels))

	// Setup goroutine to read from output channel
	var totalNumSeen int
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case data := <-outputChannels[0]:
				output, ok := data.(MultiplyOutput)
				assert.True(t, ok)
				assert.True(t, output.x%2 == 0, "output should be even")
				assert.GreaterOrEqual(t, output.x, 1, "output should be positive")
				assert.LessOrEqual(t, output.x, 20, "output should be less than or equal to 20")
				totalNumSeen++
			}
		}
	}()

	// Send data to the input channel of the first routine
	for i := range 10 {
		fmt.Printf("sending input: %d\n", i)
		addInputChannel <- AddInput{x: 1}
	}

	// Let data flow through
	time.Sleep(time.Millisecond * 100)
	// Cancel context to signal shutdown
	cancelFunc()
	// Wait for the goroutine to finish
	wg.Wait()
	// Check that we only saw 10 outputs
	expectedTotalNumSeen := 10
	assert.Equal(t, expectedTotalNumSeen, totalNumSeen, "should have only seen 10 outputs")
}

func TestTwoLeafNodeTree(t *testing.T) {
	// Create a new routine graph
	rg := NewRoutineGraph()
	// Create ctx and cancel func
	ctx, cancelFunc := context.WithCancel(context.Background())

	// Create add one routine
	add1Routine := NewRoutine(
		WithName("add-one"),
		WithFunc(AddFunc{x: 1}),
		WithContext(ctx),
	)
	// Setup input channel for add1Routine
	addInputChannel := make(chan Data)
	add1Routine.InputChannel = addInputChannel

	// Create multiply two routine
	multiply2Routine := NewRoutine(
		WithName("multiply-two"),
		WithFunc(MultiplyFunc{x: 2}),
		WithContext(ctx),
	)
	// Create multiply three routine
	multiply3Routine := NewRoutine(
		WithName("multiply-three"),
		WithFunc(MultiplyFunc{x: 3}),
		WithContext(ctx),
	)

	// Set add-one as root routine
	err := rg.AddRoutine(add1Routine, nil)
	assert.NoError(t, err)
	// Check that the graph has the add1Routine
	assert.Equal(t, 1, len(rg.routineOrder))
	assert.Equal(t, "add-one", rg.routineOrder[0].name)
	assert.Equal(t, 1, len(rg.Graph))
	assert.Equal(t, 0, len(rg.Graph["add-one"]))
	// Set multiply-two as child of add-one
	err = rg.AddRoutine(multiply2Routine, add1Routine)
	assert.NoError(t, err)
	// Check that the graph has the multiply2Routine
	assert.Equal(t, 2, len(rg.routineOrder))
	assert.Equal(t, "multiply-two", rg.routineOrder[1].name)
	assert.Equal(t, 2, len(rg.Graph))
	assert.Equal(t, 1, len(rg.Graph["add-one"]))
	assert.Equal(t, 0, len(rg.Graph["multiply-two"]))
	// Set multiply-three as child of add-one
	err = rg.AddRoutine(multiply3Routine, add1Routine)
	assert.NoError(t, err)
	// Check that the graph has the multiply3Routine
	assert.Equal(t, 3, len(rg.routineOrder))
	assert.Equal(t, "multiply-three", rg.routineOrder[2].name)
	assert.Equal(t, 3, len(rg.Graph))
	assert.Equal(t, 2, len(rg.Graph["add-one"]))
	assert.Equal(t, 0, len(rg.Graph["multiply-two"]))
	assert.Equal(t, 0, len(rg.Graph["multiply-three"]))

	// Graph should now have the structure:
	//
	// 				add-one
	// 			   /       \
	// 	 multiply-two     multiply-three

	// Start the graph
	outputChannels, err := rg.Start(true)
	assert.NoError(t, err)
	// Only expect two output channels
	assert.Equal(t, 2, len(outputChannels))

	// Setup goroutine to read from output channels
	var wg sync.WaitGroup

	wg.Add(1)
	// The first output channel should correspond to all output of multiply-two
	var firstOutputChannelCount int
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case data := <-outputChannels[0]:
				output, ok := data.(MultiplyOutput)
				assert.True(t, ok)
				assert.True(t, output.x%2 == 0, "output should be even")
				assert.GreaterOrEqual(t, output.x, 2, "output should be positive")
				assert.LessOrEqual(t, output.x, 20, "output should be less than or equal to 20")
				firstOutputChannelCount++
			}
		}
	}()

	wg.Add(1)
	// The second output channel should correspond to all output of multiply-three
	var secondOutputChannelCount int
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case data := <-outputChannels[1]:
				output, ok := data.(MultiplyOutput)
				assert.True(t, ok)
				assert.True(t, output.x%3 == 0, "output should be divisible by 3")
				assert.GreaterOrEqual(t, output.x, 3, "output should be positive")
				assert.LessOrEqual(t, output.x, 30, "output should be less than or equal to 30")
				secondOutputChannelCount++
			}
		}
	}()

	// Send data to the input channel of the root routine
	for i := range 10 {
		fmt.Printf("sending input: %d\n", i)
		addInputChannel <- AddInput{x: 1}
	}
	// Let data flow through
	time.Sleep(time.Millisecond * 100)
	// Cancel context to signal shutdown
	cancelFunc()
	// Wait for the goroutines to finish
	wg.Wait()
	// Check that we only saw 10 outputs on each channel
	expectedFirstOutputChannelCount := 10
	expectedSecondOutputChannelCount := 10
	assert.Equal(t, expectedFirstOutputChannelCount, firstOutputChannelCount, "should have only seen 10 outputs on first channel")
	assert.Equal(t, expectedSecondOutputChannelCount, secondOutputChannelCount, "should have only seen 10 outputs on second channel")
}

func TestFanOutFanInGraph(t *testing.T) {
	// Create a new routine graph
	rg := NewRoutineGraph()
	// Create context and cancel func
	ctx, cancelFunc := context.WithCancel(context.Background())

	// Create add one routine
	add1Routine := NewRoutine(
		WithName("add-one"),
		WithFunc(AddFunc{x: 1}),
		WithContext(ctx),
	)

	// Setup input channel for add1Routine
	addInputChannel := make(chan Data)
	add1Routine.InputChannel = addInputChannel
	// Set add-one as root routine
	err := rg.AddRoutine(add1Routine, nil)
	assert.NoError(t, err)
	// Check that the graph has exactly one node
	assert.Equal(t, 1, len(rg.routineOrder))
	assert.Equal(t, 1, len(rg.Graph))
	// Check that the one node is add-one
	assert.Equal(t, add1Routine.name, rg.routineOrder[0].name)
	// add-one should have no children
	assert.Equal(t, 0, len(rg.Graph[add1Routine.name]))

	// Create multiply two routine
	multiply2Routine := NewRoutine(
		WithName("multiply-two"),
		WithFunc(MultiplyFunc{x: 2}),
		WithContext(ctx),
	)

	// Set multiply-two as child of add-one
	err = rg.AddRoutine(multiply2Routine, add1Routine)
	assert.NoError(t, err)
	// Check that the graph has 2 nodes
	assert.Equal(t, 2, len(rg.routineOrder))
	assert.Equal(t, 2, len(rg.Graph))
	// Check that the second node is multiply-two
	assert.Equal(t, rg.routineOrder[1].name, multiply2Routine.name)
	// Check that add-one has one child node
	assert.Equal(t, 1, len(rg.Graph[add1Routine.name]))
	// Check that the child node is multiply-two
	assert.Equal(t, rg.Graph[add1Routine.name][0].name, multiply2Routine.name)
	// Check that multiply-two has no child nodes
	assert.Equal(t, 0, len(rg.Graph[multiply2Routine.name]))

	// Create multiply three routine
	multiply3Routine := NewRoutine(
		WithName("multiply-three"),
		WithFunc(MultiplyFunc{x: 3}),
		WithContext(ctx),
	)

	// Set multiply-three as child of add-one
	err = rg.AddRoutine(multiply3Routine, add1Routine)
	assert.NoError(t, err)
	// Check that the graph has 3 nodes
	assert.Equal(t, 3, len(rg.routineOrder))
	assert.Equal(t, 3, len(rg.Graph))
	// Check that the third node is multiply-three
	assert.Equal(t, rg.routineOrder[2].name, multiply3Routine.name)
	// Check that add-one has two child nodes
	assert.Equal(t, 2, len(rg.Graph[add1Routine.name]))
	// Check that the child nodes are multiply-two and multiply-three -- in that order
	assert.Equal(t, rg.Graph[add1Routine.name][0].name, multiply2Routine.name)
	assert.Equal(t, rg.Graph[add1Routine.name][1].name, multiply3Routine.name)
	// Check that multiply-two has no child nodes
	assert.Equal(t, 0, len(rg.Graph[multiply2Routine.name]))
	// Check that multiply-three has no child nodes
	assert.Equal(t, 0, len(rg.Graph[multiply3Routine.name]))

	// Create multiply-ten routine
	squareRoutine := NewRoutine(
		WithName("multiply-ten"),
		WithFunc(MultiplyFunc{x: 10}),
		WithContext(ctx),
	)

	// Set multiply-ten as child of multiply-two
	err = rg.AddRoutine(squareRoutine, multiply2Routine)
	assert.NoError(t, err)
	// Check that the graph has 4 nodes
	assert.Equal(t, 4, len(rg.routineOrder))
	assert.Equal(t, 4, len(rg.Graph))
	// Check that the fourth node is multiply-ten
	assert.Equal(t, rg.routineOrder[3].name, squareRoutine.name)
	// Check that multiply-two has one child node
	assert.Equal(t, 1, len(rg.Graph[multiply2Routine.name]))
	// Check that the child node is multiply-ten
	assert.Equal(t, rg.Graph[multiply2Routine.name][0].name, squareRoutine.name)
	// Check that multiply-three has no child nodes
	assert.Equal(t, 0, len(rg.Graph[multiply3Routine.name]))
	// Check that multiply-ten has no child nodes
	assert.Equal(t, 0, len(rg.Graph[squareRoutine.name]))

	// Set multiply-ten as child of multiply-three
	err = rg.AddRoutine(squareRoutine, multiply3Routine)
	assert.NoError(t, err)
	// Check that the graph still only has 4 nodes
	assert.Equal(t, 4, len(rg.routineOrder))
	assert.Equal(t, 4, len(rg.Graph))
	// Check that multiply-three still has one child node
	assert.Equal(t, 1, len(rg.Graph[multiply3Routine.name]))
	// Check that the child node is multiply-ten
	assert.Equal(t, rg.Graph[multiply3Routine.name][0].name, squareRoutine.name)
	// Check that multiply-two still has only one child node
	assert.Equal(t, 1, len(rg.Graph[multiply2Routine.name]))
	// Check that the child node is still multiply-ten
	assert.Equal(t, rg.Graph[multiply2Routine.name][0].name, squareRoutine.name)
	// Check that multiply-ten has no child nodes
	assert.Equal(t, 0, len(rg.Graph[squareRoutine.name]))

	// Graph should now have the structure:
	//
	// 				add-one
	// 			   /       \
	// 	 multiply-two     multiply-three
	// 	           \       /
	// 	         multiply-ten

	// Start the graph
	outputChannels, err := rg.Start(true)
	assert.NoError(t, err)
	// Only expect one output channel
	assert.Equal(t, 1, len(outputChannels))

	// Setup goroutine to read from output channel
	var totalNumSeen int
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case data := <-outputChannels[0]:
				output, ok := data.(MultiplyOutput)
				assert.True(t, ok)
				assert.True(t, output.x%20 == 0 || output.x%30 == 0, "output should be divisible by 20 or 30")
				assert.GreaterOrEqual(t, output.x, 20, "output should be at least 20(1)")
				assert.LessOrEqual(t, output.x, 300, "output should be at most 30(10)")
				totalNumSeen++
			}
		}
	}()

	// Send data to the input channel of the root routine
	for i := range 10 {
		fmt.Printf("sending input: %d\n", i)
		addInputChannel <- AddInput{x: 1}
	}
	// Let data flow through
	time.Sleep(time.Millisecond * 100)
	// Cancel context to signal shutdown
	cancelFunc()
	// Wait for the goroutine to finish
	wg.Wait()
	// Check that we only saw 20 outputs
	expectedTotalNumSeen := 20
	assert.Equal(t, expectedTotalNumSeen, totalNumSeen, "should have only seen 20 outputs")
}

func TestNoLoop(t *testing.T) {
	// Create a new routine graph
	rg := NewRoutineGraph()

	// Create empty context
	ctx := context.Background()

	// Create add one routine
	add1Routine := NewRoutine(
		WithName("add-one"),
		WithFunc(AddFunc{x: 1}),
		WithContext(ctx),
	)
	// Set as root routine
	err := rg.AddRoutine(add1Routine, nil)
	assert.NoError(t, err)

	// Create multiply two routine
	multiply2Routine := NewRoutine(
		WithName("multiply-two"),
		WithFunc(MultiplyFunc{x: 2}),
		WithContext(ctx),
	)
	// Set as child of add-one
	err = rg.AddRoutine(multiply2Routine, add1Routine)
	assert.NoError(t, err)

	// Now try to add add-one as child of multiply-two
	err = rg.AddRoutine(add1Routine, multiply2Routine)
	assert.Error(t, err)
}
