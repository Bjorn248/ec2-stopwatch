package main

import (
	"fmt"
	"github.com/hashicorp/vault/api"
)

// Declare Vault Connection Variables
var (
	vaultconfig *api.Config
	vaultclient *api.Client
	vaulterror  error
)

func createVaultToken(vaultclient *api.Client, email string) (string, error) {
	err := createVaultPolicy(vaultclient, email)
	if err != nil {
		fmt.Printf("Error creating vault policy: '%s'", err)
	}
	tcr := &api.TokenCreateRequest{
		Policies:    []string{email},
		DisplayName: email}
	ta := vaultclient.Auth().Token()
	s, err := ta.Create(tcr)
	if err != nil {
		return "", err
	}
	return s.Auth.ClientToken, nil
}

func createVaultPolicy(vaultclient *api.Client, email string) error {
	sys := vaultclient.Sys()
	rules := fmt.Sprintf("path \"secret/%s/*\" {\n  policy = \"write\"\n}", email)
	return sys.PutPolicy(email, rules)
}
