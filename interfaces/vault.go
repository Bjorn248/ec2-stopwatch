package interfaces

import (
    "encoding/json"
    "github.com/hashicorp/vault/api"
    "log"
)

func main() {
    config := api.DefaultConfig()
    client, err := api.NewClient(config)
}
