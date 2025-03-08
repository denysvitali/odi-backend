package indexer

func WithOpenSearchUsername(username string) Option {
	return func(i *Indexer) {
		i.opensearchUsername = username
	}
}

func WithOpenSearchPassword(password string) Option {
	return func(i *Indexer) {
		i.opensearchPassword = password
	}
}

func WithOpenSearchSkipTLS() Option {
	return func(i *Indexer) {
		i.opensearchInsecureSkipVerify = true
	}
}

func WithOcrApiCAPath(path string) Option {
	return func(i *Indexer) {
		i.ocrApiCaPath = path
	}
}
