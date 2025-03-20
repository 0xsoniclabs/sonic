package schedule

type sender int
type nonce int

type state map[sender]nonce

type action struct {
	sender sender
	nonce  nonce
}

type transaction struct {
	main action
	auth []action
}

func (tx transaction) actions() []action {
	return append([]action{tx.main}, tx.auth...)
}

func (a action) predecessor() action {
	return action{a.sender, a.nonce - 1}
}

func (a action) successor() action {
	return action{a.sender, a.nonce + 1}
}
