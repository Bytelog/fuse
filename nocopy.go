package fuse

// noCopy can be embedded in a struct to hint go vet to warn on a copy.
// https://github.com/golang/go/issues/8005#issuecomment-190753527
type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}
