package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	endpointSeal         = "/api/v1/pdf/sign"
	endpointValidate     = "/api/v1/pdf/validate"
	endpointGetSealedPDF = "/api/v1/pdf/%s"

	pdf = "JVBERi0xLjAKMSAwIG9iajw8L1R5cGUvQ2F0YWxvZy9QYWdlcyAyIDAgUj4-ZW5kb2JqCjIgMCBvYmo8PC9UeXBlL1BhZ2VzL0tpZHNbMyAwIFJdL0NvdW50IDE-PmVuZG9iagozIDAgb2JqPDwvVHlwZS9QYWdlL1BhcmVudCAyIDAgUi9SZXNvdXJjZXM8PC9Gb250PDwvRjE8PC9UeXBlL0ZvbnQvU3VidHlwZS9UeXBlMS9CYXNlRm9udC9IZWx2ZXRpY2E-Pj4-Pj4vTWVkaWFCb3hbMCAwIDIwMCA1MF0vQ29udGVudHMgNCAwIFI-PmVuZG9iago0IDAgb2JqPDwvTGVuZ3RoIDUyPj5zdHJlYW0KQlQgL0YxIDEwIFRmIDUgMjAgVGQgKFNVTkVUIEVkdVNlYWwgdGVzdCBQREYpIFRqIEVUCmVuZHN0cmVhbQplbmRvYmoKeHJlZgowIDUKMDAwMDAwMDAwMCA2NTUzNSBmIAowMDAwMDAwMDA5IDAwMDAwIG4gCjAwMDAwMDAwNTIgMDAwMDAgbiAKMDAwMDAwMDEwMSAwMDAwMCBuIAowMDAwMDAwMjUxIDAwMDAwIG4gCnRyYWlsZXI8PC9Sb290IDEgMCBSL1NpemUgNT4-CnN0YXJ0eHJlZgozNDgKJSVFT0YK"
)

type validationResponse struct {
	Data struct {
		ValidationBackend string `json:"validation_backend"`
		IntactSignature   bool   `json:"intact_signature"`
		ValidSignature    bool   `json:"valid_signature"`
		TransactionID     string `json:"transaction_id"`
	} `json:"data"`
}

type oauthConfig struct {
	AccessToken []oauthAccessToken `json:"access_token" yaml:"access_token"`
	Client      oauthClient        `json:"client" yaml:"client"`
}

type oauthAccessToken struct {
	Flags  []string      `json:"flags" yaml:"flags"`
	Access []oauthAccess `json:"access" yaml:"access"`
}

type oauthAccess struct {
	Type string `json:"type" yaml:"type"`
}

type oauthClient struct {
	Key string `json:"key" yaml:"key"`
}

type Config struct {
	OAuth         oauthConfig `json:"oauth" yaml:"oauth"`
	Env           string      `json:"env,omitempty" yaml:"env,omitempty"`
	TestCase      string      `json:"testcase,omitempty" yaml:"testcase,omitempty"`
	Save          bool        `json:"save,omitempty" yaml:"save,omitempty"`
	ClientCert    string      `json:"client_cert,omitempty" yaml:"client_cert,omitempty"`
	ClientCertKey string      `json:"client_cert_key,omitempty" yaml:"client_cert_key,omitempty"`
}

type fetchResponse struct {
	Data   fetchResponseData `json:"data"`
	Status string            `json:"status,omitempty"`
}

type fetchResponseData struct {
	TransactionID string `json:"transaction_id"`
	Data          string `json:"data"` // base64-encoded PDF
	SealerBackend string `json:"sealer_backend"`
	SealedPDF     string `json:"sealed_pdf,omitempty"` // alternative field name
	Status        string `json:"status,omitempty"`
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
	config               Config
}

func main() {
	configFlag := flag.String("config", "", "path to YAML config file")
	flag.Parse()

	config, err := loadAccessRequestBody(*configFlag)
	if err != nil {
		fmt.Printf("\033[31m✗\033[0m could not load access request body: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("flags", "env:", config.Env, "testcase:", config.TestCase)

	if config.ClientCert == "" || config.ClientCertKey == "" {
		fmt.Println("\033[31m✗\033[0m Error: client_cert and client_cert_key are required in config file")
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("\033[32m✓\033[0m Loading client certificate from %s and %s\n", config.ClientCert, config.ClientCertKey)
	clientCert, err := tls.LoadX509KeyPair(config.ClientCert, config.ClientCertKey)
	if err != nil {
		fmt.Printf("\033[31m✗\033[0m could not load client key pair: %v\n", err)
		os.Exit(1)
	}

	// Parse and display client certificate details
	if len(clientCert.Certificate) > 0 {
		_, err := x509.ParseCertificate(clientCert.Certificate[0])
		if err != nil {
			fmt.Printf("Warning: Could not parse client certificate: %v\n", err)
		}
	}

	// Load ISRG Root X1 (Let's Encrypt root cert) for server verification
	pool := x509.NewCertPool()
	isrgCert, err := os.ReadFile("/etc/ssl/certs/ISRG_Root_X1.pem")
	if err != nil {
		fmt.Printf("could not read ISRG Root X1 certificate: %v\n", err)
		os.Exit(1)
	}
	if ok := pool.AppendCertsFromPEM(isrgCert); ok {
		fmt.Println("\033[32m✓\033[0m ISRG Root X1 certificate loaded successfully")
	} else {
		fmt.Println("\033[31m✗\033[0m Warning: Failed to add ISRG Root X1 certificate")
		os.Exit(1)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            pool,
			Certificates:       []tls.Certificate{clientCert},
			MinVersion:         tls.VersionTLS12,
			MaxVersion:         tls.VersionTLS13,
			InsecureSkipVerify: false,
			Renegotiation:      tls.RenegotiateOnceAsClient,
			GetClientCertificate: func(cri *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				return &clientCert, nil
			},
		},
		Proxy: http.ProxyFromEnvironment,
	}

	client := Client{
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
		env:                config.Env,
		testCase:           config.TestCase,
		validationResponse: validationResponse{},
		config:             config,
	}

	switch client.env {
	case "test":
		client.serviceBaseURL = "https://test-api.eduseal.sunet.se"
		client.accessTransactionURL = "https://auth-test.sunet.se/transaction"
	case "prod":
		client.serviceBaseURL = "https://api.eduseal.sunet.se"
		client.accessTransactionURL = "https://auth.sunet.se/transaction"
	default:
		fmt.Printf("\033[31m✗\033[0m unknown environment: %s\n", client.env)
		os.Exit(1)
	}

	if err := client.getAccessToken(); err != nil {
		fmt.Printf("\033[31m✗\033[0m could not get access token: %v\n", err)
		os.Exit(1)
	}

	switch client.testCase {
	case "ladok":
		if err := client.checkPDFSealing(config.Save); err != nil {
			fmt.Printf("\033[31m✗\033[0m could not seal PDF: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("\033[31m✗\033[0m unknown test case: %s\n", client.testCase)
		os.Exit(1)
	}
}

func loadAccessRequestBody(configPath string) (Config, error) {
	if configPath == "" {
		return Config{}, fmt.Errorf("-config flag is required")
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(configData, &config); err != nil {
		return Config{}, fmt.Errorf("failed to parse config YAML: %v", err)
	}

	return config, nil
}

func (c *Client) getAccessToken() error {
	requestBody, err := json.Marshal(c.config.OAuth)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.accessTransactionURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		fmt.Printf("\n\033[31m✗\033[0m Error making request: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Printf("\033[31m✗\033[0m Error Response Body: %s\n", string(bodyBytes))
		fmt.Printf("\033[31m✗\033[0m Error Response Body Size: %d bytes\n", len(bodyBytes))
		fmt.Printf("=== END ACCESS TOKEN REQUEST (FAILED) ===\n\n")
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

	return nil
}

func (c *Client) checkPDFSealing(shouldSave bool) error {
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

	fmt.Printf("  Transaction ID: %s\n", c.transactionID)

	if err := c.fetchSealedPDF(); err != nil {
		return err
	}

	if err := c.validatePDF(); err != nil {
		return err
	}

	fmt.Println("\033[32m\u2713\033[0m PDF validated successfully")
	fmt.Printf("  Transaction ID: %s\n", c.validationResponse.Data.TransactionID)
	fmt.Printf("  Backend: %s\n", c.validationResponse.Data.ValidationBackend)
	fmt.Printf("  Intact Signature: %v\n", c.validationResponse.Data.IntactSignature)
	fmt.Printf("  Valid Signature: %v\n", c.validationResponse.Data.ValidSignature)

	if err := c.savePDF(shouldSave); err != nil {
		return err
	}

	return nil
}

func (c *Client) savePDF(shouldSave bool) error {
	if !shouldSave {
		return nil
	}

	filename := c.transactionID + ".pdf"

	pdfBytes, err := base64.StdEncoding.DecodeString(c.sealedPDF)
	if err != nil {
		return fmt.Errorf("failed to decode PDF: %v", err)
	}

	if err := os.WriteFile(filename, pdfBytes, 0644); err != nil {
		return fmt.Errorf("failed to write PDF file: %v", err)
	}

	fmt.Printf("\033[32m✓\033[0m PDF saved to %s\n", filename)
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

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var fetchResp fetchResponse
		if err := json.Unmarshal(bodyBytes, &fetchResp); err != nil {
			return err
		}

		// Check for PDF in data.data field (primary field)
		if fetchResp.Data.Data != "" && len(fetchResp.Data.Data) > 100 {
			c.sealedPDF = fetchResp.Data.Data
			return nil
		}

		// Check for PDF in data.sealed_pdf field (alternative field)
		if fetchResp.Data.SealedPDF != "" {
			c.sealedPDF = fetchResp.Data.SealedPDF
			return nil
		}

		// If we got a 200 but no PDF, the job might still be processing
		// Check for status field to see if job is complete
		if fetchResp.Status != "" || fetchResp.Data.Status != "" {
			status := fetchResp.Status
			if status == "" {
				status = fetchResp.Data.Status
			}
			if status == "completed" || status == "done" || status == "success" {
				// Job is done but no PDF? This is an error
				return fmt.Errorf("job completed but PDF not found in response")
			}
		}

		// Not ready yet, wait and retry
		time.Sleep(500 * time.Millisecond)
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

	if err := json.Unmarshal(bodyBytes, &c.validationResponse); err != nil {
		return err
	}
	return nil
}
