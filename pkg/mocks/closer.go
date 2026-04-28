package mocks

type Closer struct {
	CloseFunc func() error
	Calls     int
}

func (m *Closer) Close() error {
	m.Calls++
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
