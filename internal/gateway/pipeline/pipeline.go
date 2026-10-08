package pipeline

// Stage defines an executable unit of the request processing pipeline
type Stage interface {
	Name() string
	Execute(c *Context) error
}

// Pipeline coordinates the sequential execution of registered gateway stages
type Pipeline struct {
	stages []Stage
}

// New creates an ordered pipeline from the given stages
func New(stages ...Stage) *Pipeline {
	return &Pipeline{stages: stages}
}

// Execute processes the request sequentially through all registered stages until handled or errored
func (p *Pipeline) Execute(c *Context) error {
	for _, stage := range p.stages {
		if c.Handled {
			break
		}
		if err := stage.Execute(c); err != nil {
			return err
		}
	}
	return nil
}
