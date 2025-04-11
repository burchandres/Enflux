package enflux

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

/*****************************************
	Simple AddFunc Example for Testing
*****************************************/

type AddInput struct{ x int }

func (a AddInput) IsValid() bool { return true }

type AddOutput struct{ x int }

func (a AddOutput) IsValid() bool { return true }

type AddFunc struct{ x int }

func (a AddFunc) Run(data Data) Data {
	input, ok := data.(AddInput)
	if !ok {
		fmt.Println("invalid input type for add")
		return InvalidData{}
	}
	fmt.Println("returning valid output for add")
	return AddOutput{x: input.x + a.x}
}

func TestSingleRoutine(t *testing.T) {
	outputChannelNum := 2
	ctx, cancelFunc := context.WithCancel(context.Background())
	addZeroRoutine := NewRoutine(
		WithName("add-zero"),
		WithContext(ctx),
		WithFunc(AddFunc{x: 0}),
		WithScale(2),
	)
	inputAddChannel := make(chan Data)
	addZeroRoutine.InputChannel = inputAddChannel
	outputAddChannels := make([]chan Data, 0, outputChannelNum)
	for range outputChannelNum {
		outputAddChannels = append(outputAddChannels, make(chan Data))
	}
	addZeroRoutine.OutputChannels = outputAddChannels
	// Start the routine
	addZeroRoutine.Start()

	var wg sync.WaitGroup

	// Start goroutines to read from output channels
	var totalNumSeen int
	for i := range outputChannelNum {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case output := <-outputAddChannels[j]:
					assert.True(t, output.IsValid(), "output should be valid")
					addOutput, ok := output.(AddOutput)
					assert.True(t, ok, "output should be of type AddOutput")
					assert.LessOrEqual(t, addOutput.x, 9, "output should be less than or equal to 9")
					totalNumSeen++
				}
			}
		}(i)
	}

	// Send data through the input channel for the above goroutines to have something to read
	for i := range 10 {
		input := AddInput{x: i}
		fmt.Printf("sending input: %d\n", input.x)
		inputAddChannel <- input
	}
	// Sleep to allow goroutines to process
	time.Sleep(time.Millisecond * 100)
	// cancel context to signal shutdown
	cancelFunc()
	// Wait for everything to wrap up
	wg.Wait()
	// Ensure we processed all the data
	expectedTotalNumSeen := 20
	assert.Equal(t, expectedTotalNumSeen, totalNumSeen, "total number of outputs should be equal to 20")
}

/********************************
	Multiply Func for Testing
********************************/

type MultiplyOutput struct{ x int }

func (m MultiplyOutput) IsValid() bool { return true }

type MultiplyFunc struct{ x int }

func (m MultiplyFunc) Run(data Data) Data {
	switch input := data.(type) {
	case AddOutput:
		fmt.Println("valid data type AddOutput received")
		return MultiplyOutput{x: input.x * m.x}
	case MultiplyOutput:
		fmt.Println("valid data type MultiplyOutput received")
		return MultiplyOutput{x: input.x * m.x}
	default:
		fmt.Println("invalid data received, returning invalid data")
		return InvalidData{}
	}
}

func TestTwoRoutines(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	addOneRoutine := NewRoutine(
		WithName("add-zero"),
		WithContext(ctx),
		WithFunc(AddFunc{x: 1}),
		WithScale(2),
	)
	multiplyTwoRoutine := NewRoutine(
		WithName("multiply-two"),
		WithContext(ctx),
		WithFunc(MultiplyFunc{x: 2}),
		WithScale(2),
	)

	// Set input and output channels of AddRoutine
	inputAddChannel := make(chan Data)
	addOneRoutine.InputChannel = inputAddChannel
	outputAddChannel := make(chan Data)
	addOneRoutine.OutputChannels = []chan Data{outputAddChannel}
	// Set output of AddRoutine as input of MultiplyRoutine and also set output of MultiplyRoutine
	outputMultiplyChannel := make(chan Data)
	multiplyTwoRoutine.InputChannel = outputAddChannel
	multiplyTwoRoutine.OutputChannels = []chan Data{outputMultiplyChannel}
	// Start the routines
	addOneRoutine.Start()
	multiplyTwoRoutine.Start()

	var wg sync.WaitGroup

	var totalNumSeen int
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case output := <-outputMultiplyChannel:
				assert.True(t, output.IsValid(), "output should be valid")
				multiplyOutput, ok := output.(MultiplyOutput)
				assert.True(t, ok, "output should be of type MultiplyOutput")
				assert.True(t, multiplyOutput.x%2 == 0, "output should be even")
				assert.GreaterOrEqual(t, multiplyOutput.x, 1, "output should be greater than or equal to 0")
				assert.LessOrEqual(t, multiplyOutput.x, 20, "output should be less than or equal to 20")
				totalNumSeen++
			}
		}
	}()

	for i := range 10 {
		input := AddInput{x: i}
		fmt.Printf("sending input: %d\n", input.x)
		inputAddChannel <- input
	}
	// Allow time for the routines to process incoming data
	time.Sleep(time.Millisecond * 100)
	// cancel context to signal shutdown
	cancelFunc()
	// Wait for everything to wrap up
	wg.Wait()
	// Only one output channel of chain so totalNumSeen should be equal to 10
	expectedTotalNumSeen := 10
	assert.Equal(t, expectedTotalNumSeen, totalNumSeen, "total number of outputs should be equal to 20")
}
