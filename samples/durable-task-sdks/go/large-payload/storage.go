package main

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/microsoft/durabletask-go/payload"
)

// This is Azurite's PUBLIC development-only account, not an Azure credential.
// https://github.com/Azure/Azurite#default-storage-account
const developmentStorage = "DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;" +
	"AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;" +
	"BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;"

func storageOptions(container string) (payload.AzureBlobStoreOptions, error) {
	connectionString := strings.TrimSpace(os.Getenv("AZURE_STORAGE_CONNECTION_STRING"))
	endpoint := strings.TrimSpace(os.Getenv("AZURE_STORAGE_BLOB_ENDPOINT"))
	options := payload.AzureBlobStoreOptions{Container: container, MaxPayloadBytes: maxPayloadBytes}
	if connectionString != "" && endpoint != "" {
		return options, errors.New("set only AZURE_STORAGE_CONNECTION_STRING or AZURE_STORAGE_BLOB_ENDPOINT")
	}
	var client *azblob.Client
	var err error
	if endpoint != "" {
		credential, credentialErr := azidentity.NewDefaultAzureCredential(nil)
		if credentialErr != nil {
			return options, credentialErr
		}
		options.AccountURL, options.Credential = endpoint, credential
		client, err = azblob.NewClient(endpoint, credential, nil)
	} else {
		if connectionString == "" || strings.EqualFold(connectionString, "UseDevelopmentStorage=true") {
			connectionString = developmentStorage
		}
		options.ConnectionString = connectionString
		client, err = azblob.NewClientFromConnectionString(connectionString, nil)
	}
	if err != nil {
		return options, fmt.Errorf("configure Blob storage: %w", err)
	}
	address, err := url.Parse(client.URL())
	if err != nil {
		return options, errors.New("invalid Blob service URL")
	}
	options.AllowInsecureHTTP = address.Scheme == "http" &&
		(strings.EqualFold(address.Hostname(), "localhost") || net.ParseIP(address.Hostname()).IsLoopback())
	return options, nil
}
