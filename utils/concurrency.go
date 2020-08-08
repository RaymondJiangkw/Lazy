package utils

func WaitSync(chans []<-chan struct{}) {
	for _, ch := range chans {
		<-ch
	}
}
