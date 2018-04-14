package monitor

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
)

type tokenStr struct {
	LastRequestNumber map[string]int    `json:"lrn"`
	WaitinQ           []string          `json:"wq"`
	Data              map[string][]byte `json:"data"`
	Sender            string            `json:"sender"`
}

func (token *tokenStr) marshall() ([]byte, error) {
	data, err := json.Marshal(token)
	if err != nil {
		log.Error("Failed to marshall token: ", err)
		return []byte{}, err
	}
	return data, err
}

func (token *tokenStr) unMarshal(data []byte) error {
	err := json.Unmarshal(data, token)
	if err != nil {
		log.Error("Failed to unmarshall token: ", err)
		return err
	}
	return nil
}

func (token *tokenStr) saveData(data map[string]interface{}) {
	for k, v := range data {
		token.Data[k], _ = json.Marshal(v)
	}
}

func (token *tokenStr) loadData(data *map[string]interface{}) {
	for key := range *data {
		if val2, ok := token.Data[key]; ok {
			json.Unmarshal(val2, (*data)[key])
		}
	}
}

func createFirstToken(hosts []string) *tokenStr {
	token := tokenStr{
		Sender:            "",
		Data:              map[string][]byte{},
		WaitinQ:           []string{},
		LastRequestNumber: map[string]int{},
	}
	for _, h := range hosts {
		token.LastRequestNumber[h] = 0
	}

	return &token
}
