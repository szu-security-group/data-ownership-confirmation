package comparison

import (
	"errors"
	"sync"
)

// BuildPerCategory retains Build's grouping, component and root calculations,
// and uses one construction goroutine per category.
// It does not set GOMAXPROCS. Existing comparison experiments still call Build.
func (f *Forest) BuildPerCategory(xs []Item) error {
	groups := make([][]Item, len(f.Parts))
	for _, i := range xs {
		c := f.cat(i)
		if c < 0 || c >= len(groups) {
			return errors.New("invalid category")
		}
		groups[c] = append(groups[c], i)
	}
	errs := make(chan error, len(groups))
	var wg sync.WaitGroup
	for j := range groups {
		wg.Add(1)
		go func(j int) {
			defer wg.Done()
			if e := f.Parts[j].Build(groups[j]); e != nil {
				errs <- e
				return
			}
			f.Roots[j] = f.Parts[j].Root()
		}(j)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		return e
	}
	f.aggregate(-1)
	return nil
}
