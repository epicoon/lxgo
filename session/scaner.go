package session

type Scaner struct {
	provider IProvider
}

func (s *Scaner) Len() int {
	return s.provider.len()
}

func (s *Scaner) IsEmpty() bool {
	return s.Len() == 0
}

func (s *Scaner) PrintContent() string {
	return s.provider.content()
}
