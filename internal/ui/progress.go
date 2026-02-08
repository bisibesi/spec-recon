package ui

import (
	"fmt"
	"io"
	"os"

	"github.com/schollz/progressbar/v3"
)

// ProgressBar wraps the progressbar library with our custom styling
type ProgressBar struct {
	bar    *progressbar.ProgressBar
	phase  string
	total  int
	output io.Writer
}

// Phase represents a stage in the analysis pipeline
type Phase string

const (
	PhaseScanning   Phase = "Scanning"
	PhaseParsing    Phase = "Parsing"
	PhaseFiltering  Phase = "Filtering"
	PhaseLinking    Phase = "Linking"
	PhaseBuilding   Phase = "Building"
	PhaseGenerating Phase = "Generating"
)

// NewProgressBar creates a new progress bar for a specific phase
func NewProgressBar(phase Phase, total int) *ProgressBar {
	return NewProgressBarWithOutput(phase, total, os.Stdout)
}

// NewProgressBarWithOutput creates a new progress bar with custom output
func NewProgressBarWithOutput(phase Phase, total int, output io.Writer) *ProgressBar {
	bar := progressbar.NewOptions(total,
		progressbar.OptionSetWriter(output),
		progressbar.OptionSetDescription(fmt.Sprintf("[%s]", phase)),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerHead:    "█",
			SaucerPadding: "░",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetPredictTime(true),
	)

	return &ProgressBar{
		bar:    bar,
		phase:  string(phase),
		total:  total,
		output: output,
	}
}

// Add increments the progress bar by n
func (pb *ProgressBar) Add(n int) error {
	return pb.bar.Add(n)
}

// Increment increments the progress bar by 1
func (pb *ProgressBar) Increment() error {
	return pb.bar.Add(1)
}

// Set sets the progress bar to a specific value
func (pb *ProgressBar) Set(n int) error {
	return pb.bar.Set(n)
}

// SetTotal updates the total count of the progress bar
func (pb *ProgressBar) SetTotal(total int) {
	pb.bar.ChangeMax(total)
}

// Finish completes the progress bar
func (pb *ProgressBar) Finish() error {
	return pb.bar.Finish()
}

// Describe updates the description of the progress bar
func (pb *ProgressBar) Describe(description string) {
	pb.bar.Describe(fmt.Sprintf("[%s] %s", pb.phase, description))
}

// Clear clears the progress bar from the terminal
func (pb *ProgressBar) Clear() error {
	return pb.bar.Clear()
}

// Pipeline represents a multi-phase progress tracking system
type Pipeline struct {
	phases   []Phase
	current  int
	bars     []*ProgressBar
	disabled bool
	output   io.Writer
}

// NewPipeline creates a new pipeline progress tracker
func NewPipeline(phases []Phase) *Pipeline {
	return &Pipeline{
		phases:   phases,
		current:  -1,
		bars:     make([]*ProgressBar, 0, len(phases)),
		disabled: false,
		output:   os.Stdout,
	}
}

// NewPipelineWithOutput creates a new pipeline with custom output
func NewPipelineWithOutput(phases []Phase, output io.Writer) *Pipeline {
	return &Pipeline{
		phases:   phases,
		current:  -1,
		bars:     make([]*ProgressBar, 0, len(phases)),
		disabled: false,
		output:   output,
	}
}

// Disable disables the progress bar output
func (p *Pipeline) Disable() {
	p.disabled = true
}

// NextPhase moves to the next phase and returns a new progress bar
func (p *Pipeline) NextPhase(total int) *ProgressBar {
	// Finish current phase if exists
	if p.current >= 0 && p.current < len(p.bars) {
		p.bars[p.current].Finish()
	}

	p.current++
	if p.current >= len(p.phases) {
		return nil
	}

	if p.disabled {
		return &ProgressBar{
			bar:    progressbar.NewOptions(-1, progressbar.OptionSetWriter(io.Discard)),
			phase:  string(p.phases[p.current]),
			total:  total,
			output: io.Discard,
		}
	}

	bar := NewProgressBarWithOutput(p.phases[p.current], total, p.output)
	p.bars = append(p.bars, bar)
	return bar
}

// Finish completes all phases
func (p *Pipeline) Finish() {
	if p.current >= 0 && p.current < len(p.bars) {
		p.bars[p.current].Finish()
	}
}

// PrintSummary prints a summary of the pipeline phases
func (p *Pipeline) PrintSummary(message string) {
	if !p.disabled {
		fmt.Fprintln(p.output, message)
	}
}

// Spinner represents a simple spinner for indeterminate progress
type Spinner struct {
	description string
	chars       []rune
	index       int
	output      io.Writer
}

// NewSpinner creates a new spinner with a description
func NewSpinner(description string) *Spinner {
	return &Spinner{
		description: description,
		chars:       []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'},
		index:       0,
		output:      os.Stdout,
	}
}

// Tick advances the spinner
func (s *Spinner) Tick() {
	fmt.Fprintf(s.output, "\r%c %s", s.chars[s.index], s.description)
	s.index = (s.index + 1) % len(s.chars)
}

// Stop stops and clears the spinner
func (s *Spinner) Stop() {
	fmt.Fprintf(s.output, "\r%s\r", "                                                  ")
}
