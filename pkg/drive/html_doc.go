package drive

import (
	"bytes"
	"encoding/json"
	"sync"

	"golang.org/x/net/html"
)

const (
	HideStyle = "visibility:hidden;display:none"
)

type HtmlDoc struct {
	GoogleDocMetadata

	Content []byte `json:"-"`
}

func NewHtmlDoc(metadata *GoogleDocMetadata, content []byte) (HtmlDoc, error) {
	return HtmlDoc{
		GoogleDocMetadata: *metadata,
		Content:           content,
	}, nil
}

func (doc HtmlDoc) WithFrontmatter() (HtmlDoc, error) {
	jsonBytes, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return doc, err
	}
	jsonBytes = append(jsonBytes, '\n')
	doc.Content = append(jsonBytes, doc.Content...)
	return doc, nil
}

func (doc HtmlDoc) WithFixedContent() (HtmlDoc, error) {
	rootNode, err := html.Parse(bytes.NewReader(doc.Content))
	if err != nil {
		return doc, err
	}

	doc.modifyContent(rootNode)

	var b bytes.Buffer
	if err := html.Render(&b, rootNode); err != nil {
		return doc, err
	}
	doc.Content = b.Bytes()

	return doc, nil
}

func (doc HtmlDoc) modifyContent(node *html.Node) {
	if node.Type == html.ElementNode {
		switch node.Data {
		case "body":
			// Remove body styling, otherwise it affects entire page
			for i, attr := range node.Attr {
				if attr.Key == "style" {
					node.Attr[i].Val = ""
				}
			}
		case "img":
			// Fix image paths
			for i, attr := range node.Attr {
				if attr.Key == "src" {
					node.Attr[i].Val = "/" + NormalizedAssetPath(doc.Id, attr.Val)
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
		}
	}

	var wg sync.WaitGroup
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		wg.Add(1)
		go func(n *html.Node) {
			doc.modifyContent(n)
			wg.Done()
		}(child)
	}
	wg.Wait()
}
