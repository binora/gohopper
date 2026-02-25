package webapi

type GHResponse struct {
	Paths     []ResponsePath `json:"paths,omitempty"`
	Errors    []error        `json:"-"`
	DebugInfo string         `json:"-"`
	Hints     PMap           `json:"hints,omitempty"`
}

func NewGHResponse() GHResponse {
	return GHResponse{Hints: NewPMap()}
}

func (r *GHResponse) Add(path ResponsePath) {
	r.Paths = append(r.Paths, path)
}

func (r *GHResponse) AddError(err error) {
	r.Errors = append(r.Errors, err)
}

func (r GHResponse) HasErrors() bool {
	return len(r.Errors) > 0
}

func (r GHResponse) GetBest() *ResponsePath {
	if len(r.Paths) == 0 {
		return nil
	}
	return &r.Paths[0]
}

func (r GHResponse) GetAll() []ResponsePath {
	return r.Paths
}

func (r *GHResponse) AddDebugInfo(debug string) {
	if r.DebugInfo == "" {
		r.DebugInfo = debug
		return
	}
	r.DebugInfo += "; " + debug
}
