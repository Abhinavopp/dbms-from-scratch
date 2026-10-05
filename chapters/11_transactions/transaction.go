package chapter11

type Transaction struct {
	writes map[string]string
	state  bool
}

func Begin() *Transaction {
	return &Transaction{writes: make(map[string]string), state: true}
}

func (t *Transaction) Set(key, value string) {
	if !t.state {
		return
	}
	t.writes[key] = value
}

func (t *Transaction) Commit() map[string]string {
	if !t.state {
		return nil
	}
	out := make(map[string]string, len(t.writes))
	for k, v := range t.writes {
		out[k] = v
	}
	t.state = false
	return out
}

func (t *Transaction) Rollback() {
	t.writes = make(map[string]string)
	t.state = false
}
