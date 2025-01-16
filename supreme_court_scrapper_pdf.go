package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/unidoc/unipdf/v3/common/license"
)

var causeListMap = make(map[string]CauseList)

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file")
	}
}

var Scraped_data_final []CauseListEntry

// Function to save Scraped_data_final to CSV
func saveToCSV(fileName string, data []CauseListEntry) error {
	// Create the CSV file
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %v", err)
	}
	defer file.Close()

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the CSV header
	header := []string{"Sno", "CaseNo", "DiaryNo", "CaseNoMap", "JudgeName", "CourtNo"}
	err = writer.Write(header)
	if err != nil {
		return fmt.Errorf("failed to write header to CSV: %v", err)
	}

	// Write the data rows
	for _, entry := range data {
		row := []string{entry.Sno, entry.CaseNo, entry.DiaryNo, entry.CaseNoMap, entry.JudgeName, entry.CourtNo}
		err = writer.Write(row)
		if err != nil {
			return fmt.Errorf("failed to write row to CSV: %v", err)
		}
	}

	return nil
}

func getSupremeCourtCauselistPDF(data map[string]string) map[string]string {
	dirName := "./causelist/pdf/supreme_court"

	// Create directory if it doesn't exist
	err := os.MkdirAll(dirName, os.ModePerm)
	if err != nil {
		fmt.Println("Error creating directory:", err)
		return data
	}

	hitDate := data["hitDate"]

	if hitDate == "" {
		log.Fatal("Hit date is empty. Unable to create filename.")
	}

	// Set up custom client with timeout
	client := http.Client{
		Timeout: 60 * time.Second, // Increase the timeout
	}

	// Retry mechanism for HTTP request
	for retry := 0; retry < 3; retry++ {
		// Create a new HTTP request
		req, err := http.NewRequest("GET", data["url"], nil)
		if err != nil {
			fmt.Println("Error creating HTTP request:", err)
			continue
		}

		// Add headers from the `curl` request
		req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
		req.Header.Set("accept-language", "en-GB,en-US;q=0.9,en;q=0.8,hi;q=0.7")
		req.Header.Set("cache-control", "no-cache")
		req.Header.Set("cookie", "pll_language=en; _ga=GA1.1.722567382.1728648220; PHPSESSID=fljmtivh7294k43nutrfub86nq; _ga_BZ6N54FGYB=GS1.1.1728970090.2.1.1728970201.0.0.0")
		req.Header.Set("pragma", "no-cache")
		req.Header.Set("priority", "u=0, i")
		req.Header.Set("sec-ch-ua", `"Google Chrome";v="129", "Not=A?Brand";v="8", "Chromium";v="129"`)
		req.Header.Set("sec-ch-ua-mobile", "?0")
		req.Header.Set("sec-ch-ua-platform", `"Windows"`)
		req.Header.Set("sec-fetch-dest", "document")
		req.Header.Set("sec-fetch-mode", "navigate")
		req.Header.Set("sec-fetch-site", "same-origin")
		req.Header.Set("sec-fetch-user", "?1")
		req.Header.Set("upgrade-insecure-requests", "1")
		req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36")

		// Make the request
		response, err := client.Do(req)
		if err != nil {
			fmt.Println("Error on HTTP request:", err)
			time.Sleep(5 * time.Second) // Backoff before retrying
			continue
		}
		defer response.Body.Close()

		if response.StatusCode == http.StatusOK {
			hitDate = strings.ReplaceAll(hitDate, "/", "-")
			fmt.Println("Response OK, saving causelist...")

			filename := fmt.Sprintf("%s/causelist_pdf_%s.html", dirName, hitDate)
			fmt.Println("Saving to file:", filename)

			body, err := io.ReadAll(response.Body)
			if err != nil {
				log.Fatalf("failed to read response body: %v", err)
			}

			err = os.WriteFile(filename, body, 0644)
			if err != nil {
				log.Fatalf("Failed to create local file: %v", err)
			}

			fileType := "wb"
			url, err := upload_on_S3(os.Getenv("S3_BUCKET_NAME"), filename, body, fileType)
			if err != nil {
				log.Fatalf("failed to upload to S3: %v", err)
			}
			data["s3_path"] = url

			log.Printf("Uploaded successfully, accessible at: %s", url)
			finalMap, err := FetchAndParseCauselist(data)
			log.Println("final map :", finalMap)
			if err != nil {
				log.Fatalf("Error parsing causelist: %v", err)
			}

			// Print the map if no error
			for key, value := range finalMap {
				fmt.Printf("Key: %s\n", key)
				fmt.Printf("Date of Hearing: %s\n", value.DateOfHearing)
				fmt.Printf("Description: %s\n", value.Description)
				fmt.Printf("PDF Link: %s\n", value.PDFLink)
			}
			size := len(finalMap)
			log.Println("Size of the created map", size)
			Scraped_data_final = parseCauselistPDFData(finalMap)
			return data
		} else {
			fmt.Printf("Received non-200 response: %d\n", response.StatusCode)
		}

	}

	fmt.Println("Failed to fetch causelist after 3 retries.")
	return data
}

func main() {
	data := map[string]string{
		"url":     "https://www.sci.gov.in/cause-list/",
		"hitDate": "10/16/2024",
	}
	key := "86e1b2ce5494c387152f820f453a655bf6b182023656f3f0c720da7d5f72a7f7"
	err := license.SetMeteredKey(string(key))
	if err != nil {
		log.Fatalf("Failed to set metered key: %s", err)
	}
	currentTime := time.Now()
	fmt.Println("Current time :", currentTime)
	getSupremeCourtCauselistPDF(data)
	log.Println("causeListMap", causeListMap)
	saveSCCauseListToRedis(causeListMap)
	err = saveToCSV("causelist_data.csv", Scraped_data_final)
	if err != nil {
		log.Fatalf("Failed to save to CSV: %s", err)
	}

	fmt.Println("Data saved to causelist_data.csv")
}
