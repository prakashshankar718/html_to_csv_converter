package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/html"
)

type CsvContent struct {
	contents [][]string
}

func main() {
	r := gin.Default()

	//  automatically escapes HTML in templates, which helps prevent XSS (Cross-Site Scripting) attacks.
	//  While this isn't strictly validation, it ensures that any potentially harmful HTML content is safely escaped when rendered.
	r.SetFuncMap(template.FuncMap{
		"safe": func(htmlContent string) template.HTML {
			return template.HTML(htmlContent)
		},
	})

	// Load HTML files from the templates directory
	r.LoadHTMLGlob("templates/*")

	r.StaticFile("/favicon.ico", "./assets/favicon.ico")

	// to host CSS, JavaScript, or images
	// r.Static("/assets", "./assets")

	// Define a route to serve the HTML file
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	// r.GET("/favicon.ico", func(c *gin.Context) {
	// 	c.HTML(200, "favicon.ico", nil)
	// })

	r.POST("/api/html", func(c *gin.Context) {
		bodyBytes, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot read request body"})
			return
		}

		// Convert the body to a string
		content := string(bodyBytes)
		content = strings.TrimPrefix(content, "content=")

		fmt.Println("validating html content")
		// validating content
		if err := isValidHTML(content); err != nil {
			fmt.Println("Invalid HTML:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			fmt.Println("HTML is valid")
		}

		fmt.Println("decoding content")
		// decoding content
		content, err = url.QueryUnescape(content)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			fmt.Println("Error decoding content:", err)
			return
		}

		// check if content has table
		fmt.Println("converting content...")
		csvString, err := convertToCsv(content)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			fmt.Println("Error parsing HTML:", err)
		}

		c.JSON(http.StatusOK, csvString)
		fmt.Println("convertion completed")
		// Respond with the received data
		// c.JSON(http.StatusOK, gin.H{
		// 	"csv": "csv",
		// })
	})

	r.Run() // listen and serve on 0.0.0.0:8080
}

func isValidHTML(htmlContent string) error {
	reader := strings.NewReader(htmlContent)
	_, err := html.Parse(reader)
	return err
}

func convertToCsv(htmlContent string) (csvString string, err error) {
	reader := strings.NewReader(htmlContent)
	doc, err := html.Parse(reader)
	if err != nil {
		return
	}

	node, hasNode := findTable(doc)
	if !hasNode {
		return csvString, errors.New("no table")
	}
	fmt.Println("table node", node.Data)

	csvContent := &CsvContent{
		contents: make([][]string, 0),
	}
	getTableData(node, csvContent)

	csvString, err = getTableInCsv(csvContent.contents)
	// fmt.Println(csvString)
	return
}

func getTableInCsv(csvContent [][]string) (csvString string, err error) {
	var buffer bytes.Buffer

	// Create a CSV writer that writes to the buffer
	writer := csv.NewWriter(&buffer)

	// Write all rows to the CSV writer
	for _, row := range csvContent {
		if err := writer.Write(row); err != nil {
			return csvString, err
			// log.Fatalf("Failed to write row: %s", err)
		}
	}

	// Flush the writer to make sure all data is written to the buffer
	writer.Flush()

	// Check if there were any errors during the write
	if err := writer.Error(); err != nil {
		return csvString, err
		// log.Fatalf("Error during CSV writing: %s", err)
	}

	// Get the CSV string from the buffer
	csvString = buffer.String()
	return
}
func getTableData(n *html.Node, csvContent *CsvContent) {
	// printFullTableNode(n)
	if n.Type != html.ElementNode {
		return
	}
	// tbody
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Data == "tr" {
			row := make([]string, 0)
			for c1 := c.FirstChild; c1 != nil; c1 = c1.NextSibling {
				if c1.Data == "td" || c1.Data == "th" {
					row = append(row, c1.FirstChild.Data)
					// fmt.Println("td c.Data", c1.FirstChild.Data)
				}
			}
			// fmt.Println("row", row)
			if len(row) > 0 {
				(*csvContent).contents = append((*csvContent).contents, row)
			}
		}
		getTableData(c, csvContent)
	}

	return
}

func printFullTableNode(n *html.Node) {
	if n.Type == html.ElementNode {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			fmt.Println("c.Data", c.Data)
			printFullTableNode(c)
		}
	}
	return
}

func findTable(n *html.Node) (node *html.Node, has bool) {
	if n.Type == html.ElementNode && n.Data == "table" {
		return n, true
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if node, ok := findTable(c); ok {
			return node, true
		}
	}
	return n, false
}

// func traverse(n *html.Node) bool {
// 	if n.Type == html.ElementNode && n.Data == "table" {
// 		return true
// 	}
// 	for c := n.FirstChild; c != nil; c = c.NextSibling {
// 		if traverse(c) {
// 			return true
// 		}
// 	}
// 	return false
// }

// func containsTableTag(htmlContent string) (bool, error) {
// 	reader := strings.NewReader(htmlContent)
// 	doc, err := html.Parse(reader)
// 	if err != nil {
// 		return false, err
// 	}
// 	return traverse(doc), nil
// }

// sample html

// <!DOCTYPE html>
// <html>
// <head>
// <style>
// table {
//   font-family: arial, sans-serif;
//   border-collapse: collapse;
//   width: 100%;
// }

// td, th {
//   border: 1px solid #dddddd;
//   text-align: left;
//   padding: 8px;
// }

// tr:nth-child(even) {
//   background-color: #dddddd;
// }
// </style>
// </head>
// <body>

// <h2>HTML Table</h2>

// <table>
//   <tr>
//     <th>Company</th>
//     <th>Contact</th>
//     <th>Country</th>
//   </tr>
//   <tr>
//     <td>Alfreds Futterkiste</td>
//     <td>Maria Anders</td>
//     <td>Germany</td>
//   </tr>
//   <tr>
//     <td>Centro comercial Moctezuma</td>
//     <td>Francisco Chang</td>
//     <td>Mexico</td>
//   </tr>
//   <tr>
//     <td>Ernst Handel</td>
//     <td>Roland Mendel</td>
//     <td>Austria</td>
//   </tr>
//   <tr>
//     <td>Island Trading</td>
//     <td>Helen Bennett</td>
//     <td>UK</td>
//   </tr>
//   <tr>
//     <td>Laughing Bacchus Winecellars</td>
//     <td>Yoshi Tannamuri</td>
//     <td>Canada</td>
//   </tr>
//   <tr>
//     <td>Magazzini Alimentari Riuniti</td>
//     <td>Giovanni Rovelli</td>
//     <td>Italy</td>
//   </tr>
// </table>

// </body>
// </html>
