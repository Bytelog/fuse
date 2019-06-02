package bytepool

func Put(b *Buffer) {

}

func Get() *Buffer {
	return nil
}

type Pool struct {

}

func (p *Pool) Get() *Buffer {
	return nil
}

func (p Pool) Put(b *Buffer) {

}
