package zefix

type Hit[T any] struct {
	Index  string `json:"_index"`
	Type   string `json:"_type"`
	ID     string `json:"_id"`
	Score  int    `json:"_score"`
	Source T      `json:"_source"`
}

type Value struct {
	Value int `json:"value"`
}

type Hits[T any] struct {
	Total Value `json:"total"`
	Hits  []Hit[T]
}

type OpensearchResult[T any] struct {
	Took int `json:"took"`
	Hits Hits[T]
}
