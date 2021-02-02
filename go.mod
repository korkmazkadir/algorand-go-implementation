module github.com/korkmazkadir/algorand-go-implementation

go 1.15

replace github.com/korkmazkadir/algorand-go-implementation/agreement => ./agreement

replace github.com/korkmazkadir/algorand-go-implementation/blockchain => ./blockchain

replace github.com/korkmazkadir/algorand-go-implementation/config => ./config

replace github.com/korkmazkadir/go-rpc-node => /home/kadir/Git/go-rpc-node

require (
	github.com/korkmazkadir/coordinator v0.0.0-20201123151243-7796dde68aad
	github.com/korkmazkadir/go-rpc-node v0.0.0-00010101000000-000000000000
	gonum.org/v1/gonum v0.8.2
)
