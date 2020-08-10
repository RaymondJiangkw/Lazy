package utils

func WaitSync(chans []<-chan struct{}) {
	for _, ch := range chans {
		if ch == nil {
			continue
		}
		<-ch
	}
}
