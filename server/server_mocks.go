package server

import (
	"fmt"
	"log"
	"reflect"

	"github.com/lukaszbudnik/auditor/model"
)

type mockStore struct {
	errorThreshold int
	counter        int
	audit          []model.Block
}

func (ms *mockStore) Save(block interface{}) error {
	if ms.errorThreshold > 0 && ms.counter == ms.errorThreshold {
		return fmt.Errorf("Error %v", ms.errorThreshold)
	}
	ms.audit = append(ms.audit, *block.(*model.Block))
	ms.counter++
	return nil
}

func (ms *mockStore) Read(result interface{}, limit int64, last interface{}) error {
	if ms.errorThreshold > 0 && ms.counter == ms.errorThreshold {
		return fmt.Errorf("Error %v", ms.errorThreshold)
	}

	resultv := reflect.ValueOf(result)
	slicev := resultv.Elem()
	slicev = slicev.Slice(0, slicev.Cap())

	for _, b := range ms.audit {
		log.Println(b)
		slicev = reflect.Append(slicev, reflect.ValueOf(b))
	}
	resultv.Elem().Set(slicev.Slice(0, len(ms.audit)))

	ms.counter++
	return nil
}

func (ms *mockStore) Close() {
}
