# fuse
Direct FUSE implementation in Go


## Goals
- Provide a correct filesystem api
- Avoid unnecessary allocations/overhead.
- Track and support latest versions of FUSE.
- Reduce required complexity for filesystem implementation.
- Full support for "exotic" operations (copy file range, locking, etc).


## Stretch Goals
- BSD, OSX, and Windows support


## Non-goals
- Example filesystem implementations beyond trivial tests.
