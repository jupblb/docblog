package drive

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

const (
	HideStyle = "visibility:hidden;display:none"
)

type HtmlDoc struct {
	GoogleDocMetadata

	Content []byte
}

const googleUrlPrefix = "https://www.google.com/url"

var (
	colorRegex = regexp.MustCompile(`color:[^;]+;`)
	fontRegex  = regexp.MustCompile(`font-\w+:[^;]+;`)
)

func NewHtmlDoc(metadata *GoogleDocMetadata, content []byte) (HtmlDoc, error) {
	return HtmlDoc{
		GoogleDocMetadata: *metadata,
		Content:           content,
	}, nil
}

// withFrontmatter adds frontmatter to the HTML content that contains Google Doc
// metadata along with content description.
func (doc HtmlDoc) WithFrontmatter() (HtmlDoc, error) {
	content := []byte("---\n")
	content = append(content, "layout: post\n"...)

	yamlBytes, err := yaml.Marshal(doc.GoogleDocMetadata)
	if err != nil {
		return doc, err
	}

	content = append(content, yamlBytes...)
	content = append(content, "---\n"...)
	content = append(content, doc.Content...)

	doc.Content = content
	return doc, nil
}

// WithFixedContent modifies the HTML content in the following way:
//   - Removes any font-related styling
//   - Removes any additional styling from the <style> tag
//   - Removed Google redirect from URL links
//   - Removed body styling
//   - Increases header levels by 1 (by default there may be many h1 tags)
//   - Fixes asset paths
//   - Removes title and subtitle, this should be taken from the frontmatter
func (doc HtmlDoc) WithFixedContent(assetPathPrefix string) (HtmlDoc, error) {
	rootNode, err := html.Parse(bytes.NewReader(doc.Content))
	if err != nil {
		return doc, err
	}

	doc.modifyContent(rootNode, assetPathPrefix)

	var b bytes.Buffer
	if err := html.Render(&b, rootNode); err != nil {
		return doc, err
	}
	doc.Content = b.Bytes()

	return doc, nil
}

func (doc HtmlDoc) modifyContent(node *html.Node, assetPathPrefix string) {
	if node.Type == html.ElementNode {
		// Drop font family
		for i, attr := range node.Attr {
			if attr.Key == "style" {
				val := attr.Val + ";"
				val = colorRegex.ReplaceAllString(val, "")
				val = fontRegex.ReplaceAllString(val, "")
				node.Attr[i].Val = val
			}
		}

		switch node.Data {
		case "a":
			// Google redirect links have a timestamp that is refreshed on each read
			for i, attr := range node.Attr {
				if attr.Key == "href" && strings.HasPrefix(attr.Val, googleUrlPrefix) {
					if url, err := url.Parse(attr.Val); err == nil {
						if query := url.Query(); query != nil {
							if q := query.Get("q"); q != "" {
								node.Attr[i].Val = q
							}
						}
					}
				}
			}
		case "body":
			// Remove body styling, otherwise it affects entire page
			for i, attr := range node.Attr {
				if attr.Key == "style" {
					node.Attr[i].Val = ""
				}
			}
		case "h1", "h2", "h3", "h4", "h5", "h6":
			node.Data = fmt.Sprintf("h%d", int(node.Data[1]-'0')+1)
		case "img":
			// Fix image paths
			for i, attr := range node.Attr {
				if attr.Key == "src" {
					node.Attr[i].Val = "/" +
						NormalizedAssetPath(assetPathPrefix, doc.Id, attr.Val)
				}
			}
		case "p":
			// Remove title and subtitle, it should be taken from the frontmatter
			var isTitle, isSubtitle bool
			for _, attr := range node.Attr {
				if attr.Key == "class" {
					if attr.Val == "title" {
						isTitle = true
					}
					if attr.Val == "subtitle" {
						isTitle = true
					}
				}
			}

			if isTitle || isSubtitle {
				for i, attr := range node.Attr {
					if attr.Key == "style" {
						node.Attr[i].Val += ";" + HideStyle
					}
				}
			}
		case "style":
			node.Parent.RemoveChild(node)
		}
	}

	var wg sync.WaitGroup
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		wg.Add(1)
		go func(n *html.Node) {
			doc.modifyContent(n, assetPathPrefix)
			wg.Done()
		}(child)
	}
	wg.Wait()
}
