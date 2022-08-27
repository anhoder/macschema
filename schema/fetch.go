package schema

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

func FetchTopic(ctx context.Context, l Lookup) Topic {
	u, _ := url.Parse(l.URL)

	ctx, cancel := chromedp.NewContext(ctx)
	defer cancel()
	ctx, cancel = context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	var t Topic
	t.LastFetch = time.Now()
	t.LastVersion = Version
	t.Path = strings.Replace(l.URL, BaseURL, "/documentation/", 1)

	mustRun(ctx,
		chromedp.Navigate(u.String()),
		chromedp.WaitVisible(`main div.topictitle`),
	)
	dur := time.Duration(1 * time.Second)
	short, _ := context.WithTimeout(ctx, dur)
	go chromedp.Run(short, chromedp.Text(`main div.topictitle h1.title`, &t.Title))
	short, _ = context.WithTimeout(ctx, dur)
	go chromedp.Run(short, chromedp.Text(`main div.topictitle span.eyebrow`, &t.Type))
	short, _ = context.WithTimeout(ctx, dur)
	go chromedp.Run(short, chromedp.Text(`main div.description div.abstract.content`, &t.Description))
	short, _ = context.WithTimeout(ctx, dur)
	go chromedp.Run(short, chromedp.Text(`#declaration pre.source`, &t.Declaration))
	short, _ = context.WithTimeout(ctx, dur)
	go chromedp.Run(short, textList(`#app div.sidebar div.head-wrapper a h2`, &t.Frameworks))
	short, _ = context.WithTimeout(ctx, dur)
	go chromedp.Run(short, textList(`main div.availability span span`, &t.Platforms))

	short, _ = context.WithTimeout(ctx, dur)
	err := chromedp.Run(short, chromedp.WaitVisible(`#topics`))
	if err == nil {
		for _, section := range nodes(ctx, "#topics div.contenttable-section", nil) {
			var title string
			mustRun(ctx, chromedp.Text("div.section-title h3.title", &title, chromedp.ByQuery, chromedp.FromNode(section)))
			//fmt.Println(title)
			for _, topic := range nodes(ctx, "div.section-content div.topic a.link", section) {
				var ok bool
				l := Link{Section: title}
				mustRun(ctx,
					chromedp.Text(topic.FullXPathByID(), &l.Name),
					chromedp.AttributeValue(topic.FullXPathByID(), "href", &l.Path, &ok),
				)
				t.Topics = append(t.Topics, l)
			}

		}
	}

	if t.Type == "" && t.Declaration != "" {
		if t.Declaration[0] == '-' {
			t.Type = "Instance Method"
		} else if t.Declaration[0] == '+' {
			t.Type = "Type Method"
		}
	}

	// topics := "#topics div.contenttable-section div.section-content div.topic a.link"
	// if os.Getenv("ALLOW_DEPRECATED") == "" {
	// 	topics = topics + ":not(.deprecated)"
	// }
	// for _, n := range nodes(ctx, topics) {
	// 	var t Topic
	// 	var ok bool
	// 	if err := chromedp.Run(ctx,
	// 		chromedp.TextContent(n.FullXPathByID(), &t.Name),
	// 		chromedp.AttributeValue(n.FullXPathByID(), "href", &t.URL, &ok),
	// 	); err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	// Handle related consts/enums later
	// 	if strings.HasPrefix(t.Name, "NS") ||
	// 		strings.HasPrefix(t.Name, "CG") ||
	// 		strings.HasPrefix(t.Name, "UI") ||
	// 		strings.HasPrefix(t.Name, "WK") {
	// 		continue
	// 	}

	// 	// Manual fix for less than perfect selector
	// 	if strings.HasPrefix(t.Name, "API Reference") {
	// 		continue
	// 	}

	// 	if t.URL != "" {
	// 		t.URL = fmt.Sprintf("https://developer.apple.com%s", t.URL)
	// 	}

	// 	if strings.HasPrefix(t.Name, "+ ") {
	// 		class.TypeMethods = append(class.TypeMethods, TypeMethod{Name: t.Name[2:], URL: t.URL})
	// 	} else if strings.HasPrefix(t.Name, "- ") {
	// 		class.InstanceMethods = append(class.InstanceMethods, InstanceMethod{Name: t.Name[2:], URL: t.URL})
	// 	} else {
	// 		class.Properties = append(class.Properties, Property{Name: t.Name, URL: t.URL})
	// 	}
	// }

	return t
}

func nodes(ctx context.Context, sel string, fromNode *cdp.Node) []*cdp.Node {
	var nodes []*cdp.Node
	task := chromedp.Nodes(sel, &nodes)
	if fromNode != nil {
		task = chromedp.Nodes(sel, &nodes, chromedp.ByQueryAll, chromedp.FromNode(fromNode))
	}
	mustRun(ctx, task)
	return nodes
}

func textList(sel string, lst *[]string) chromedp.Tasks {
	var nodes []*cdp.Node
	return chromedp.Tasks{
		chromedp.Nodes(sel, &nodes),
		chromedp.ActionFunc(func(ctx context.Context) error {
			for _, n := range nodes {

				txt := innerText(n)
				if txt != "" {
					*lst = append(*lst, txt)
				}
			}
			return nil
		}),
	}
}

func innerText(node *cdp.Node) string {
	var t []string
	for _, c := range node.Children {
		switch c.NodeType {
		case cdp.NodeTypeText:
			t = append(t, strings.Trim(c.NodeValue, " \n"))
		case cdp.NodeTypeElement:
			t = append(t, innerText(c))
		}
	}
	return strings.Trim(strings.Join(t, ""), " \n")
}
