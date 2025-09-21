package operations

import "time"

type functionalCheck struct {
	id       string
	name     string
	domain   Domain
	mode     CheckMode
	interval time.Duration
	fn       func() (Status, string, map[string]interface{})
}

func newFunctionalCheck(id, name string, domain Domain, mode CheckMode, interval time.Duration, fn func() (Status, string, map[string]interface{})) Check {
	if interval <= 0 {
		interval = time.Minute
	}
	return &functionalCheck{
		id:       id,
		name:     name,
		domain:   domain,
		mode:     mode,
		interval: interval,
		fn:       fn,
	}
}

func (c *functionalCheck) ID() string              { return c.id }
func (c *functionalCheck) Domain() Domain          { return c.domain }
func (c *functionalCheck) Mode() CheckMode         { return c.mode }
func (c *functionalCheck) Interval() time.Duration { return c.interval }

func (c *functionalCheck) Run() Result {
	status, summary, details := c.fn()
	return Result{
		ID:      c.id,
		Domain:  c.domain,
		Name:    c.name,
		Status:  status,
		Summary: summary,
		Details: details,
	}
}
