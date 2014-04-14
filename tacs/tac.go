package tacs

type Tac struct {
  err error
}

func NewTac() *Tac {
  return &Tac{}
}
