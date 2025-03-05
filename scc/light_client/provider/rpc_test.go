package provider

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/0xsoniclabs/sonic/tests"
	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/require"
)

func TestRPCProvider_CanRetrieveCommitteeCertificates(t *testing.T) {

	// start network
	net, err := tests.StartIntegrationTestNet(t.TempDir())
	require.NoError(t, err)
	client, err := net.GetClient()
	require.NoError(t, err)

	provider := NewRPCProvider(nil, client)

	// call
	_, err = provider.GetCommitteeCertificate(0, 0)

	// assert
	require.NoError(t, err)
}

func Test_GetRpcUrls(t *testing.T) {
	require := require.New(t)
	const url = "https://chainlist.org/chain/146"
	resp, err := http.Get(url)
	require.NoError(err, "Failed to fetch data")
	defer resp.Body.Close()

	require.Equal(http.StatusOK, resp.StatusCode, "Failed to fetch data")

	// body, err := ioutil.ReadAll(resp.Body)
	// require.NoError(err, "Failed to read data")

	// Process the body to extract RPC URLs
	// Step 2: Load the HTML document
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	require.NoError(err, "Failed to parse HTML")

	// Step 3: Find and extract RPC URLs
	// fmt.Println("List of RPC Servers:")
	// doc.Find("a[href^='https://']").Each(func(index int, item *goquery.Selection) {
	// 	href, exists := item.Attr("href")
	// 	if exists && (isRPCURL(href)) {
	// 		fmt.Println(href)
	// 	}
	// })
	// doc.Find("a[href^='wss://']").Each(func(index int, item *goquery.Selection) {
	// 	href, exists := item.Attr("href")
	// 	if exists && (isRPCURL(href)) {
	// 		fmt.Println(href)
	// 	}
	// })

	// Step 3: Find tables and search for "RPC Server Address"
	doc.Find("table").Each(func(index int, table *goquery.Selection) {
		// Check if the table contains a header with "RPC Server Address"
		table.Find("th").Each(func(i int, th *goquery.Selection) {
			if strings.Contains(strings.ToLower(th.Text()), "rpc server address") {
				fmt.Println("Found table with RPC Server Address column:")

				// Extract RPC addresses from the table
				table.Find("tr").Each(func(j int, row *goquery.Selection) {
					row.Find("td").Each(func(k int, cell *goquery.Selection) {
						text := strings.TrimSpace(cell.Text())
						if strings.HasPrefix(text, "http") { // Likely an RPC URL
							fmt.Println(text)
						}
					})
				})
			}
		})
	})
}

// // Step 4: Helper function to filter RPC URLs
// func isRPCURL(url string) bool {
// 	return len(url) > 0 && (contains(url, "rpc") || contains(url, "sonic") || contains(url, "net"))
// }

// // Simple substring check
// func contains(s, substr string) bool {
// 	return len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr)
// }
