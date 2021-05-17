package main

import (
	"encoding/csv"
	"fmt"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/ungerik/go-rss"
	"log"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"unicode/utf8"
)
const (
	htmlTagStart = 60 // Unicode `<`
	htmlTagEnd   = 62 // Unicode `>`
)
func main(){
	helpmsg:="WATCHOVER - RSS Reader\nPress F1 to view this message\nPress Ctrl+Q to exit\nTo move up and down use arrow keys\n" +
		"To switch between lists and textview use PgUp and PgDown\nPress Enter to select\n\n" +
		"Config File for feeds is standard CSV as defined in RFC4180(located at userhome/.config/watchover/)\n" +
		"Example:\n<name of feed1>,<url to feed1>\n<name of feed2>,<url to feed2>\n.\n.\n."
	currentFocus := 0
	app := tview.NewApplication()
	feeds := tview.NewList()
	news := tview.NewList()
	text := tview.NewTextView().SetText(helpmsg).SetScrollable(true).SetWordWrap(true)
	appgrid :=tview.NewGrid().SetColumns(0,-2,-3).SetRows(0).SetBorders(true)
	appgrid.AddItem(feeds,0,0,1,1,0,0,true)
	appgrid.AddItem(news,0,1,1,1,0,0,false)
	appgrid.AddItem(text,0,2,1,1,0,0,false)
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key(){
		case tcell.KeyCtrlQ:
			app.Stop()
			return nil
		case tcell.KeyPgDn:
			switch currentFocus {
			case 0:
				app.SetFocus(news)
				currentFocus = 1
				return nil
			case 1:
				app.SetFocus(text)
				currentFocus = 2
				return nil
			}
			return nil
		case tcell.KeyPgUp:
			switch currentFocus {
			case 2:
				app.SetFocus(news)
				currentFocus = 1
				return nil
			case 1:
				app.SetFocus(feeds)
				currentFocus = 0
				return nil
			}
		case tcell.KeyF1:
			text.SetText(helpmsg)
		}
		return event
	})
	rssfeeds, err := getFeeds()
	if err != nil{
		panic(err)
	}
	for i,_ := range rssfeeds{
		feeds.AddItem(rssfeeds[i][0],rssfeeds[i][1],0, func() {
			news.Clear()
			cur := feeds.GetCurrentItem()
			urlStr := rssfeeds[cur][1]
			u, err := url.Parse(urlStr)
			if err != nil{
				panic(err)
			}
			reddit := false
			if u.Host == "reddit.com" {
				reddit = true
			}
			resp, err := rss.Read(urlStr, reddit)
			if err != nil {
				fmt.Println(err)
			}
			ext := filepath.Ext(urlStr)
			if ext == ".atom" {
				text.SetText("Cannot open this feed as it is not a rss feed.")
			} else {
				channel, err := rss.Regular(resp)
				if err != nil {
					fmt.Println(err)
				}

				items := channel.Item
				for _, item := range items {
					news.AddItem(item.Title,item.Author,0, func() {
						index := news.GetCurrentItem()
						itemToUse := items[index]
						text.Clear()
						time,_ := itemToUse.PubDate.Parse()
						title := itemToUse.Title
						var author string
						if (itemToUse.Author == "") {
							author = "Unknown Author"
						}else{
							author = itemToUse.Author
						}
						var content string
						if (itemToUse.Content == ""){
							if (itemToUse.Description==""){
								content = "No Content."

							}else {
								content = "Description\n"+itemToUse.Description+"\n\nURL: " + itemToUse.Link
							}
						}else {
							content = "Content:"+itemToUse.Content
						}
						content=stripHtmlTags(content)
						text.SetText(title + "\n" + author + "\n" + time.String() + "\n\n" + content)
					})
					if err != nil {
						fmt.Println(err)
					}
				}
			}
		})
	}
	app.SetRoot(appgrid, true)
	err = app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
func getFeeds() ([][]string,error){
	u,_:=user.Current()
	confpath:=u.HomeDir+"/.config/watchover/"
	path := confpath+"feeds"
	file,err := os.Open(path)
	if err != nil{
		os.Create(path)
		panic(err)
	}
	csvReader:= csv.NewReader(file)
	return csvReader.ReadAll()
}
func stripHtmlTags(s string) string {
	// Setup a string builder and allocate enough memory for the new string.
	var builder strings.Builder
	builder.Grow(len(s) + utf8.UTFMax)
	in := false // True if we are inside an HTML tag.
	start := 0  // The index of the previous start tag character `<`
	end := 0    // The index of the previous end tag character `>`
	for i, c := range s {
		// If this is the last character and we are not in an HTML tag, save it.
		if (i+1) == len(s) && end >= start {
			builder.WriteString(s[end:])
		}
		// Keep going if the character is not `<` or `>`
		if c != htmlTagStart && c != htmlTagEnd {
			continue
		}
		if c == htmlTagStart {
			// Only update the start if we are not in a tag.
			// This make sure we strip out `<<br>` not just `<br>`
			if !in {
				start = i
			}
			in = true
			// Write the valid string between the close and start of the two tags.
			builder.WriteString(s[end:start])
			continue
		}
		// else c == htmlTagEnd
		in = false
		end = i + 1
	}
	s = builder.String()
	return s
}