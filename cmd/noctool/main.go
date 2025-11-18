package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var (
	endpointSeal         = "/v1/pdf/sign"
	endpointValidate     = "/v1/pdf/validate"
	endpointGetSealedPDF = "/v1/pdf/%s"

	pdf = "JVBERi0xLjAKMSAwIG9iajw8L1R5cGUvQ2F0YWxvZy9QYWdlcyAyIDAgUj4+ZW5kb2JqCjIgMCBvYmo8PC9UeXBlL1BhZ2VzL0tpZHNbMyAwIFJdL0NvdW50IDE+PmVuZG9iagozIDAgb2JqPDwvVHlwZS9QYWdlL1BhcmVudCAyIDAgUi9SZXNvdXJjZXM8PD4+L01lZGlhQm94WzAgMCA5IDldPj5lbmRvYmoKeHJlZgowIDQKMDAwMDAwMDAwMCA2NTUzNSBmIAowMDAwMDAwMDA5IDAwMDAwIG4gCjAwMDAwMDAwNTIgMDAwMDAgbiAKMDAwMDAwMDEwMSAwMDAwMCBuIAp0cmFpbGVyPDwvUm9vdCAxIDAgUi9TaXplIDQ+PgpzdGFydHhyZWYKMTc0CiUlRU9G"

	accessRequestBody = map[string]any{
		"access_token": []map[string]any{
			{
				"flags": []string{
					"bearer",
				},
				"access": []map[string]any{
					{
						"type": "eduseal",
					},
				},
			},
		},
		"client": map[string]any{
			"key": "masv_prod_3",
		},
	}
)

type validationResponse struct {
	Data struct {
		ValidationBackend string `json:"validation_backend"`
		IntactSignature   bool   `json:"intact_signature"`
		ValidSignature    bool   `json:"valid_signature"`
		TransactionID     string `json:"transaction_id"`
	} `json:"data"`
}

type Client struct {
	httpClient           *http.Client
	env                  string
	serviceBaseURL       string
	accessTransactionURL string
	accessToken          string
	testCase             string
	sealedPDF            string
	transactionID        string
	validationResponse   validationResponse
}

func main() {
	envFlag := flag.String("env", "prod", "preset environment: test, prod")
	testCaseFlag := flag.String("testcase", "ladok", "test case to run: seal, validate")
	flag.Parse()

	fmt.Println("flags", "env:", *envFlag, "testcase:", *testCaseFlag)

	clientCert, err := tls.LoadX509KeyPair("/tmp/client.cert", "/tmp/client.key")
	if err != nil {
		fmt.Printf("could not load client key pair: %v\n", err)
		os.Exit(1)
	}

	caCert, err := os.ReadFile("/etc/ssl/certs/ca-certificates.crt")
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caCert)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:      pool,
			Certificates: []tls.Certificate{clientCert},
			CipherSuites: []uint16{
				//  TLS_AES_256_GCM_SHA384,
				tls.TLS_AES_256_GCM_SHA384,
			},
		},
		Proxy: http.ProxyFromEnvironment,
	}

	client := Client{
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
		env:                *envFlag,
		testCase:           *testCaseFlag,
		validationResponse: validationResponse{},
	}

	switch client.env {
	case "test":
		client.serviceBaseURL = "https://api-test.eduseal.sunet.se"
		client.accessTransactionURL = "https://auth-test.sunet.se/transaction"
	case "prod":
		client.serviceBaseURL = "https://api.eduseal.sunet.se"
		client.accessTransactionURL = "https://auth.sunet.se/transaction"
	default:
		fmt.Printf("unknown environment: %s\n", client.env)
		os.Exit(1)
	}

	if err := client.getAccessToken(); err != nil {
		fmt.Printf("could not get access token: %v\n", err)
		os.Exit(1)
	}

	switch client.testCase {
	case "ladok":
		if err := client.checkPDFSealing(); err != nil {
			fmt.Printf("could not seal PDF: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("unknown test case: %s\n", client.testCase)
		os.Exit(1)
	}

	fmt.Println(*envFlag, client)
}

//type jwt struct {
//	{
//	"access_token": {
//		"value": "eyJhbGciOiJFUzI1NiIsImtpZCI6InN1bmV0X2F1dGhfMjAyMzAzIn0.eyJhdXRoX3NvdXJjZSI6ImNvbmZpZyIsImV4cCI6MTc1MDgwNzkwNywiaWF0IjoxNzUwNzcxOTA3LCJpc3MiOiJodHRwczovL2F1dGguc3VuZXQuc2UiLCJuYmYiOjE3NTA3NzE5MDcsIm9yZ2FuaXphdGlvbl9pZCI6Ijg2MDIyMyIsInJlcXVlc3RlZF9hY2Nlc3MiOlt7InR5cGUiOiJlZHVzZWFsIn1dLCJzb3VyY2UiOiJjb25maWciLCJzdWIiOiJtYXN2X3Byb2RfMyIsInZlcnNpb24iOjF9.2usKQKp0MoHKL3OwaDUx9aQ0RSynhQdLPRLLeiCJ_yV2hNOPQzJUqu-5kEsYfB3XoJwRdXjlS4XIo1IqhpmSTg",
//		"access": [
//			{
//				"type": "eduseal"
//			}
//		],
//		"expires_in": 36000,
//		"flags": [
//			"bearer"
//		]
//	}
//}
//	Issuer   string `json:"iss"`

func (c *Client) getAccessToken() error {
	requestBody, err := json.Marshal(accessRequestBody)
	if err != nil {
		return err
	}

	fmt.Println("url", c.accessTransactionURL, "body", string(requestBody))

	resp, err := c.httpClient.Post(c.accessTransactionURL, "application/json", bytes.NewBuffer(requestBody))

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get token, status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	payload := map[string]any{}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return err
	}

	accessToken := payload["access_token"].(map[string]any)

	c.accessToken = accessToken["value"].(string)
	fmt.Println("Access Token:", c.accessToken)

	return nil
}

func (c *Client) checkPDFSealing() error {
	fmt.Println("Sealing PDF...")

	requestBody := map[string]any{
		"pdf": pdf,
	}

	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.serviceBaseURL+endpointSeal, bytes.NewBuffer(requestBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to seal PDF, status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	responsePayload := map[string]any{}
	if err := json.Unmarshal(bodyBytes, &responsePayload); err != nil {
		return err
	}

	data := responsePayload["data"].(map[string]any)

	var ok bool
	c.transactionID, ok = data["transaction_id"].(string)
	if !ok {
		return fmt.Errorf("transaction_id not found in response")
	}

	fmt.Println("Transaction ID:", c.transactionID)

	if err := c.fetchSealedPDF(); err != nil {
		return err
	}

	fmt.Println("Sealed PDF:", c.sealedPDF)

	if err := c.validatePDF(); err != nil {
		return err
	}

	fmt.Println("PDF validated successfully", c.validationResponse)

	return nil
}

func (c *Client) fetchSealedPDF() error {
	fmt.Println("Fetching sealed PDF...")
	stop := time.Now().Add(11 * time.Second)

	for time.Now().Before(stop) {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(c.serviceBaseURL+endpointGetSealedPDF, c.transactionID), nil)
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.accessToken)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get sealed PDF, status: %s", resp.Status)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		responsePayload := map[string]any{}
		if err := json.Unmarshal(bodyBytes, &responsePayload); err != nil {
			return err
		}

		var ok bool
		c.sealedPDF, ok = responsePayload["sealed_pdf"].(string)
		if !ok {
			return fmt.Errorf("sealed_pdf not found in response")
		}

		time.Sleep(100 * time.Millisecond)
	}

	return errors.New("timed out waiting for sealed PDF")
}

func (c *Client) validatePDF() error {
	fmt.Println("Validating PDF...")

	requestBody := map[string]any{
		"pdf": c.sealedPDF,
	}

	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.serviceBaseURL+endpointValidate, bytes.NewBuffer(requestBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to validate PDF, status: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(bodyBytes, c.validationResponse); err != nil {
		return err
	}
	return nil
}

//{
//	"data": {
//		"validation_backend": "car-prod-2.eduseal.sunet.se",
//		"intact_signature": true,
//		"valid_signature": true,
//		"transaction_id": "1a91de28-d2ee-453e-a0d7-bde721175e64"
//	}
//}
