package fuse

func (t fuse_in_header) Request() Request {
	return Request{
		NodeID: t.nodeid,
		UID:    t.uid,
		GID:    t.gid,
		PID:    t.pid,
	}
}
