package rawpb

type Option func(*RawPB)

func Begin(f func() error) Option {
	return func(p *RawPB) {
		p.beginFunc = f
	}
}

func End(f func() error) Option {
	return func(p *RawPB) {
		p.endFunc = f
	}
}
