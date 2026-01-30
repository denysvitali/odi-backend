package indexer

import (
	"context"
	"sync"

	"github.com/denysvitali/odi-backend/pkg/models"
)

type Worker struct {
	id  int
	ch  chan models.ScannedPage
	idx *Indexer
}

func NewWorker(id int, ch chan models.ScannedPage) Worker {
	return Worker{id: id, ch: ch}
}

func (w *Worker) do(ctx context.Context, page models.ScannedPage) {
	log.Debugf("[W%d]: processing %s", w.id, page.ID())

	// Process image
	err := w.idx.Index(ctx, page)
	if err != nil {
		log.Errorf("[W%d]: %s cannot be processed: %v", w.id, page.ID(), err)
	}

	log.Debugf("[W%d]: done processing %s", w.id, page.ID())
}

func (w *Worker) Start(ctx context.Context, wg *sync.WaitGroup) {
	if w.idx == nil {
		log.Errorf("unable to start worker: w.idx is nil")
		return
	}
	for v := range w.ch {
		w.do(ctx, v)
	}
	log.Infof("done processing all")
	wg.Done()
}

func (w *Worker) SetIndexer(idx *Indexer) {
	w.idx = idx
}
