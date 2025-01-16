package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	//"os"

	"strings"

	"golang.org/x/net/html"
	//"github.com/gocolly/colly/v2"
)

// CauseList represents the structure of each cause list entry
type CauseList struct {
	DateOfHearing string
	Description   string
	PDFLink       string
}

func getDescription(link string) string {
	link1 := strings.ToLower(link) // Convert to lowercase for case-insensitive comparison

	switch {
	case strings.Contains(link1, "advance"):
		return "JUDGE MISCELLANEOUS ADVANCE"
	case strings.Contains(link1, "m_j_1"):
		return "JUDGE MISCELLANEOUS MAIN"
	case strings.Contains(link1, "m_j_2"):
		return "JUDGE MISCELLANEOUS SUPPL"
	case strings.Contains(link1, "f_j_1"):
		return "JUDGE REGULAR MAIN"
	case strings.Contains(link1, "f_j_2"):
		return "JUDGE REGULAR SUPPL"
	case strings.Contains(link1, "m_c_1"):
		return "CHAMBER MAIN"
	case strings.Contains(link1, "m_c_2"):
		return "CHAMBER SUPPL"
	case strings.Contains(link1, "m_s_1"):
		return "SINGLE JUDGE MAIN"
	case strings.Contains(link1, "m_s_2"):
		return "SINGLE JUDGE SUPPL"
	case strings.Contains(link1, "m_cc_1"):
		return "REVIEW & CURATIVE MAIN"
	case strings.Contains(link1, "m_cc_2"):
		return "REVIEW & CURATIVE SUPPL"
	case strings.Contains(link1, "m_r_1"):
		return "REGISTRAR MAIN"
	case strings.Contains(link1, "m_r_2"):
		return "REGISTRAR SUPPL"
	default:
		return ""
	}
}

// getDateOfHearing extracts the date of hearing from the link using a regex pattern
func getDateOfHearing(link string) string {
	// Split the URL by the slash `/`
	parts := strings.Split(link, "/")

	// Iterate over the parts and check for the part with the date format "YYYY-MM-DD"
	for _, part := range parts {
		// We expect the date to be in the format "YYYY-MM-DD", so we can check the length
		if len(part) == 10 && strings.Contains(part, "-") {
			// Found the date part
			return part
		}
	}

	// Return an empty string if no date is found
	return ""
}

// ParseCauselist fetches the HTML from S3 and extracts cause list information

// ParseCauselist fetches cause list information from HTML content
func ParseCauselist(htmlContent string) (map[string]CauseList, error) {
	z := html.NewTokenizer(strings.NewReader(htmlContent))

	// Dictionary to hold parsed cause list entries
	var currentLink, currentDesc, currentDate string
	var inTableRow bool // Track if we're inside a <tr> element
	var links []string

	log.Println(currentDate)
	// Traversing the HTML tokens
	for {
		tt := z.Next()

		switch tt {
		case html.ErrorToken:
			// End of the document
			return causeListMap, nil

		case html.StartTagToken:
			t := z.Token()

			// Detect the start of a table row
			if t.Data == "tr" {
				inTableRow = true
			}

			// Detect the anchor tag for PDF link
			if t.Data == "a" && inTableRow {
				for _, attr := range t.Attr {
					if attr.Key == "href" && strings.Contains(attr.Val, ".pdf") {
						// Construct the full URL
						currentLink = attr.Val
						links = append(links, currentLink) // Add to links list
					}
				}
			}

			// Detect the table data cells for description and hearing date
			if t.Data == "td" && inTableRow {
				// Peek ahead for date or description based on the content
				z.Next()
				textToken := z.Token()
				//log.Println("textToken: ",textToken)

				if textToken.Type == html.TextToken {
					// Adjust based on your actual HTML structure
					if strings.Contains(textToken.Data, "/2024") {
						currentDate = textToken.Data
					} else if len(currentDesc) == 0 {
						currentDesc = textToken.Data
					}
				}
			}

		case html.EndTagToken:
			t := z.Token()
			// Detect the end of a table row
			if t.Data == "tr" && inTableRow && len(links) > 0 {
				for _, link := range links {
					causelistEntry := CauseList{
						DateOfHearing: getDateOfHearing(link),
						Description:   getDescription(link),
						PDFLink:       link,
					}

					// Create a unique ID for each entry
					trimmedLink := trimPDFLink(link)
					uniqueID := string(trimmedLink)

					// Store the entry in the map

					causeListMap[uniqueID] = causelistEntry
					log.Println("causelistEntry", causelistEntry)
				}

				//he current row values

				currentLink, currentDesc, currentDate = "", "", ""
				links = nil // Clear the links for the next row
				inTableRow = false
			}
		}
	}
}

// // FetchAndParseCauselist fetches the HTML from S3 using the s3_path and parses the cause list
func FetchAndParseCauselist(data map[string]string) (map[string]CauseList, error) {
	s3Path := data["s3_path"]

	// Fetch the HTML content from the S3 path
	resp, err := http.Get(s3Path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HTML content: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch HTML content, status: %v", resp.Status)
	}

	// Read the response body (HTML content)
	htmlContent, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTML content: %v", err)
	}
	log.Println("hello 222")
	ans, err := ParseCauselist(string(htmlContent))
	return ans, err
}

func trimPDFLink(link string) string {
	// Split the URL by "/"
	parts := strings.Split(link, "/")

	// Get the last two parts (2024-10-16 and M_R_2.pdf)
	date := parts[len(parts)-2]
	fileName := parts[len(parts)-1]

	// Remove the ".pdf" extension from the file name
	fileName = strings.TrimSuffix(fileName, ".pdf")

	// Combine them to form the unique ID
	uniqueID := fmt.Sprintf("%s/%s", date, fileName)

	return uniqueID
}
