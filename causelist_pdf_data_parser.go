package main

import (
	"bytes"
	"fmt" // For formatted I/O like Printf, Sprintf, etc.
	"io/ioutil"

	// For string manipulations like splitting, trimming, etc.\
	"log"
	"net/http" // For extracting text from PDF pages
	"regexp"
	"strings"

	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

type CauseListEntry struct {
	Sno       string // Serial number
	CaseNo    string // Case number
	DiaryNo   string // Diary number
	CaseNoMap string // Mapped case number with proper formatting
	JudgeName string // Name of the judge
	CourtNo   string // Court number
}

func caseNumberCleaner(caseNum string) (string, string, string, string) {
	// Check if "in " is present in the case number
	if strings.Contains(caseNum, "in ") {
		return "", "", "", ""
	}

	// Handle "Connected" case
	if strings.Contains(caseNum, "Connected") {
		tempSno := strings.ReplaceAll(strings.Split(caseNum, "Connected")[0], " ", "")
		tempSno = strings.ReplaceAll(tempSno, "\n", "")
		caseNum = strings.Replace(caseNum, strings.Split(caseNum, "Connected")[0], tempSno, 1)
	}

	// Clean up the case number string
	caseNum = strings.ReplaceAll(caseNum, "\n", " ")
	caseNum = strings.ReplaceAll(caseNum, "Connected", "")
	caseNum = strings.ReplaceAll(caseNum, "  ", " ")

	// Split case number into serial number (sno) and the rest
	parts := strings.SplitN(caseNum, " ", 2)
	sno := strings.TrimSpace(parts[0])
	caseNum = strings.TrimSpace(parts[1])

	// Handle "Diary No." or "Dno"
	if strings.Contains(caseNum, "Diary No.") || strings.Contains(caseNum, "Dno") {
		diaryNo := strings.Split(caseNum, " ")[len(strings.Split(caseNum, " "))-1]
		return sno, "", "", diaryNo
	}

	// Handle case numbers with "./" or "/"
	var caseNumRep string
	if strings.Contains(caseNum, "./") {
		caseNumRep = strings.Split(caseNum, "./")[1]
	} else {
		caseNumRep = strings.Split(caseNum, "/")[0]
	}
	caseNumRep = strings.TrimSpace(strings.Split(caseNumRep, " ")[len(strings.Split(caseNumRep, " "))-1])

	// Handle numbers with "-" or without "-"
	var number string
	if strings.Contains(caseNumRep, "-") {
		numberParts := strings.Split(caseNumRep, "-")
		// Pad the first part with leading zeros if it's less than 6 characters
		if len(numberParts[0]) < 6 {
			numberParts[0] = fmt.Sprintf("%06s", numberParts[0])
		}
		// Pad the second part with leading zeros if it's present and less than 6 characters
		if len(numberParts) > 1 && len(numberParts[1]) > 0 {
			numberParts[1] = fmt.Sprintf("%06s", numberParts[1])
			number = fmt.Sprintf("%s - %s", numberParts[0], numberParts[1])
		}
	} else {
		// If there's no "-", pad the number and duplicate it with a dash
		if len(caseNumRep) < 6 {
			number = fmt.Sprintf("%06s - %06s", caseNumRep, caseNumRep)
		} else {
			number = caseNumRep + " - " + caseNumRep
		}
	}

	// Replace the original case number with the formatted version
	caseNumMap := strings.Replace(caseNum, caseNumRep, number, 1)

	return sno, caseNum, caseNumMap, ""
}

func getCaseNumber(data string) []string {
	// Define the regular expression pattern
	pattern := `\d{1,}[.]?(\d{1,})?\s(\d{1,})?(Connected\s)?[a-zA-Z]+.[a-zA-Z]+..?([a-zA-Z]+)?.?.?\sNo.\s\d{1,}-?(\d{1,})?\/\d{4}\b|` +
		`\d{1,}[.]?(\d{1,})?\s?(Connected)?\s?MA\s\d{1,}-?(\d{1,})?\/\d{4}|` +
		`\d{1,}[.]?(\d{1,})?\s?(\d{1,})?(Connected)?\s?Diary\sNo.\s\d{1,}-?\/?\d{4}|` +
		`\d{1,}[.]?(\d{1,})?\s(\d{1,})?[a-zA-Z]+.[a-zA-Z]+..?([a-zA-Z]+)?.?.?\sNo.\s?.?\d{1,}-?(\d{1,})?\/\d{4}|` +
		`\d{1,}[.]?(\d{1,})?\s?(\d{1,})?(Connected)?\s?Dno\s\d{1,}-?\/?\d{4}`

	// Compile the regular expression
	re := regexp.MustCompile(pattern)

	// Find all matches in the input string
	matches := re.FindAllString(data, -1)

	// Return the list of matched strings
	return matches
}

func getCourtNoAndJudge(text string) (string, string) {
	findText := "court no. :"
	if !strings.Contains(strings.ToLower(text), "court no. :") {
		findText = "dated :"
	}
	if strings.Contains(strings.ToLower(text), "registrar court no.") {
		findText = "registrar court no."
	}

	lines := strings.Split(text, "\n")
	var judgeCheck bool
	var courtNo string
	var judges []string

	for _, line := range lines {
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "(time :") || strings.Contains(lineLower, "note:") || strings.Contains(lineLower, "this bench") {
			break
		}
		if judgeCheck {
			judges = append(judges, strings.TrimSpace(line))
		}
		if strings.Contains(lineLower, findText) {
			if strings.Contains(lineLower, "court no. :") {
				courtNo = strings.TrimSpace(strings.Split(lineLower, "court no. :")[1])
			} else if strings.Contains(lineLower, "registrar court no.") {
				courtNo = strings.TrimSpace(strings.Split(lineLower, "registrar court no.")[1])
			}
			judgeCheck = true
		}
	}

	return courtNo, strings.Join(judges, ", ")
}

func extractTextFromPDFURL(pdfURL string) (string, error) {
	// Download the PDF file
	resp, err := http.Get(pdfURL)
	if err != nil {
		return "", fmt.Errorf("failed to download PDF: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body into a byte slice
	pdfData, err := ioutil.ReadAll(resp.Body) // Read the entire response
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Create a PDF reader from the buffer
	pdfReader, err := model.NewPdfReader(bytes.NewReader(pdfData))
	if err != nil {
		return "", fmt.Errorf("failed to create PDF reader: %w", err)
	}

	var text strings.Builder // Use strings.Builder for efficient string concatenation

	// Get the number of pages in the PDF
	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return "", fmt.Errorf("failed to get number of pages: %w", err)
	}

	// Iterate through the pages and extract text
	for i := 1; i <= numPages; i++ {
		page, err := pdfReader.GetPage(i)
		if err != nil {
			return "", fmt.Errorf("failed to get page %d: %w", i, err)
		}

		// Create a text extractor for the page
		textExtractor, err := extractor.New(page) // Capture both returned values
		if err != nil {
			return "", fmt.Errorf("failed to create text extractor for page %d: %w", i, err)
		}

		// Extract text from the page
		pageText, err := textExtractor.ExtractText()
		if err != nil {
			return "", fmt.Errorf("failed to extract text from page %d: %w", i, err)
		}
		text.WriteString(pageText) // Append the text using strings.Builder // Append the text using strings.Builder
	}

	return text.String(), nil // Return the concatenated text
}

// Function to parse and extract details from PDF data
func parseCauselistPDFData(data map[string]CauseList) []CauseListEntry {
	var causelists []CauseListEntry
  
	for pdfID, entry := range data {
		// Get the PDF link, date of hearing, and description from the CauseList struct
		log.Println(pdfID)
		pdfLink := entry.PDFLink
		// Extract text from the PDF
		pdfText, err := extractTextFromPDFURL(pdfLink)
		if err != nil {
			fmt.Printf("Failed to extract text from %s: %v\n", pdfLink, err)
			continue
		}

		// Split text by a common identifier
		if strings.Contains(pdfText, "SUPREME COURT OF INDIA") {
			textSections := strings.Split(pdfText, "SUPREME COURT OF INDIA")

			for _, eachText := range textSections {
				if !strings.Contains(eachText, "No.") {
					continue
				}

				// Extract court number and judge names
				courtNo, judges := getCourtNoAndJudge(eachText)

				// Clean and extract case numbers
				caseNos := getCaseNumber(eachText)
				for _, caseNo := range caseNos {
					// Clean the case number and other details
					sno, cleanCaseNo, caseNoMap, diaryNo := caseNumberCleaner(caseNo)

					// Create the CauseListEntry struct
					causelist := CauseListEntry{
						Sno:       sno,
						CaseNo:    cleanCaseNo,
						DiaryNo:   diaryNo,
						CaseNoMap: caseNoMap,
						JudgeName: judges,
						CourtNo:   courtNo,
					}

					// Append the cause list entry to the result slice
					causelists = append(causelists, causelist)
				}
			}
		}
	}
	return causelists
}
