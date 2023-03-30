package promise

type Semaphore struct {
	Parent *Semaphore
	ticket line
	dark   line
}

func NewSemaphore(available uint64) *Semaphore {
	return (&Semaphore{}).Init(available)
}

func (s *Semaphore) Init(available uint64) *Semaphore {
	s.ticket = make(line, available)
	s.dark = make(line, available)
	//
	s.Post(available)
	return s
}

func (s *Semaphore) Then(child *Semaphore) *Semaphore {
	if child != nil {
		child.Parent = s
	}
	return s
}

func (s *Semaphore) Acquire() {
	for s != nil {
		s.Get(1)
		s = s.Parent
	}
}

func (s *Semaphore) Release() {
	for s != nil {
		s.Post(1)
		s = s.Parent
	}
}

func (s *Semaphore) Get(n uint64) {
	for i := uint64(0); i < n; i++ {
		<-s.ticket
	}
}

func (s *Semaphore) Post(n uint64) {
	go func() {
		for i := uint64(0); i < n; i++ {
			select {
			case <-s.dark:
			default:
				select {
				case s.ticket <- signal:
				case <-s.dark:
				}
			}
		}
	}()
}

func (s *Semaphore) Reduce(n uint64) {
	go func() {
		for i := uint64(0); i < n; i++ {
			s.dark <- signal
		}
	}()
}
