package webhook

import (
    "net/http"
	"encoding/json"
	"bytes"
	"strconv"
)

func SendWebHook (message string, address string) (string, error) {
	values := map[string]string{"message": message}
    json_data, err := json.Marshal(values)
    if err != nil {
        return "", err
    }
    resp, err := http.Post(address, "application/json",
        bytes.NewBuffer(json_data))

    if err != nil {
        return "", err
    }

	defer resp.Body.Close()

	return strconv.Itoa(resp.StatusCode), nil
}