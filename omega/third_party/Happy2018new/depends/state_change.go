package Happy2018new_depends

// 向其他线程通知状态更改
func (s *StateChange) SendSignal() {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, value := range s.signal {
		if len(value) == 0 {
			value <- struct{}{}
		}
	}
}

// 每一个线程如需要接受状态更改的通知，
// 则需要调用该函数进行注册。
// 它返回一个管道，用于通知状态更改。
//
// 另，一旦注册便不可撤销，直到程序退出
func (s *StateChange) Register() <-chan struct{} {
	s.lock.Lock()
	defer s.lock.Unlock()
	new := make(chan struct{}, 1)
	s.signal = append(s.signal, new)
	return new
}
