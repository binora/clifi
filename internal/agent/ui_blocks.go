package agent

type UIBlockKind string

const (
	UIBlockTable UIBlockKind = "table"
	UIBlockKV    UIBlockKind = "kv"
)

type UIBlock struct {
	Kind  UIBlockKind `json:"kind"`
	Table *UITable    `json:"table,omitempty"`
	KV    *UIKV       `json:"kv,omitempty"`
}

type UITable struct {
	Title   string     `json:"title,omitempty"`
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

type UIKV struct {
	Title string   `json:"title,omitempty"`
	Items []KVItem `json:"items"`
}

type KVItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
