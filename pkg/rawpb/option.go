package rawpb

type Option func(*RawPB)

func Begin(f func()) Option {
	return func(p *RawPB) {
		p.beginFunc = f
	}
}

func End(f func()) Option {
	return func(p *RawPB) {
		p.endFunc = f
	}
}
