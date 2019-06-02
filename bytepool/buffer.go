package bytepool

import "io"

type Buffer struct {

}

func (b *Buffer) Len() int {
	return 0
}

func (b *Buffer) ReadFrom(r io.Reader) (int64, error) {
	return 0, nil
}

func (b *Buffer) Write(p []byte) (int, error) {
	return 0, nil
}

func (b *Buffer) WriteTo(w io.Writer) (int64, error) {
	return 0, nil
}

