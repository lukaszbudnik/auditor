package server

import (
	"fmt"

	"github.com/lukaszbudnik/auditor/model"
)

type mockStore struct {
	errorThreshold int
	counter        int
	audit          []model.Block
}

func (ms *mockStore) Save(block *model.Block) error {
	if ms.errorThreshold > 0 && ms.counter == ms.errorThreshold {
		return fmt.Errorf("Error %v", ms.errorThreshold)
	}
	ms.audit = append(ms.audit, *block)
	ms.counter++
	return nil
}

func (ms *mockStore) Read(limit int64, lastBlock *model.Block) ([]model.Block, error) {
	if ms.errorThreshold > 0 && ms.counter == ms.errorThreshold {
		return nil, fmt.Errorf("Error %v", ms.errorThreshold)
	}
	ms.counter++
	return ms.audit, nil
}

func (ms *mockStore) Close() {
}
