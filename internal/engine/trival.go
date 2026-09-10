package engine

type TriVal int

const (
	TriValFalse TriVal = iota
	TriValTrue
	TriValUnknown
)

func (t TriVal) And(other TriVal) TriVal {
	if t == TriValFalse || other == TriValFalse {
		return TriValFalse
	}
	if t == TriValTrue && other == TriValTrue {
		return TriValTrue
	}
	return TriValUnknown
}

func (t TriVal) Or(other TriVal) TriVal {
	if t == TriValTrue || other == TriValTrue {
		return TriValTrue
	}
	if t == TriValFalse && other == TriValFalse {
		return TriValFalse
	}
	return TriValUnknown
}

func (t TriVal) Not() TriVal {
	if t == TriValTrue {
		return TriValFalse
	}
	if t == TriValFalse {
		return TriValTrue
	}
	return TriValUnknown
}
