package responder

func WithTraceID(id string) Option {
	return func(m *Meta) {
		m.TraceId = id
	}
}

func WithTook(ms int64) Option {
	return func(m *Meta) {
		m.Took = ms
	}
}

func WithPagination(p *PaginationMeta) Option {
	return func(m *Meta) {
		m.Pagination = p
	}
}

func NewMeta(opts ...Option) *Meta {
	meta := Meta{}
	for _, opt := range opts {
		opt(&meta)
	}
	return &meta
}
