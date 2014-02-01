//michael isaac

package main

import (
	"fmt"
        "net/http"
        "bufio"
        "os"
        "strings"
        "encoding/json"
        "code.google.com/p/go.net/html"
        "io/ioutil"
)

type CardObject struct {
    MvID string     //multiverse ID
    Name string     //card name
    Cost string     //cost
    Type string     //types
    Power string    //power & toughness
    Rules []string  //array of rules
    Sets []string   //array of sets, the last word of each is the rarity in that set
}

type Card interface {
    GetCardData() []byte
}

//save the card a file
func (c CardObject) GetCardData() []byte {
    //b, e := json.Marshal(c)
    b, e := json.MarshalIndent(c, "", "    ")
    if(e != nil) {
        return []byte{}
    }
    //fmt.Println(string(b))
    //return string(b)
    return append(b, '\n')
}

//no document length is bad news.
func GetGathererSource() int {
    //this is the URL that will result the entire database being returned for consumption
    //http://gatherer.wizards.com/Pages/Search/Default.aspx?output=spoiler&method=text&name=+[]&special=true

    //fmt.Println("Thom says:  Be patient; this may take a very long time.  Working...")

    response, err := http.Get("http://gatherer.wizards.com/Pages/Search/Default.aspx?output=spoiler&method=text&name=+[]&special=true")

    if err != nil {
        //fmt.Println("Error in getGathererSource():\n\n" + err.Error())
        panic(err.Error())
    } else {
        document, e := ioutil.ReadAll(response.Body)
        response.Body.Close()

        if( e != nil ) {
            panic(e.Error())
        }

        e = ioutil.WriteFile("gatherer_source.html", document, 0777)
        if( e != nil ) {
            panic(e.Error())
        }

        return len(document)
        //fmt.Println(fmt.Sprintf("Thom says:  All done! The source material is %d megabytes.", (len(document) / (1024 * 1024) ) ) )
    }
}


func AssembleSourceMaterial() int {

    size := 0

    //open the source file and get its handle
    f_src, e := os.Open("gatherer_source.html")
    if(e != nil) {
        fmt.Println("App Error:\n\n" + e.Error())
        return -1
    }
    defer f_src.Close()


    //open the destination file and get its handle
    f_dst, e := os.Create("mtg_data.txt")
    if(e != nil) {
        fmt.Println("App Error:\n\n" + e.Error())
        return -1
    }
    defer f_dst.Close()


    //output/write buffer
    output := bufio.NewWriter(f_dst)

    z := html.NewTokenizer( f_src )

    tableDepth := 0
    rowDepth := 0
    cellDepth := 0

    currentCard := new(CardObject)
    currentSwitch := ""

    for {
        //set the working token and get its type (Start, End. etc)
        tokenType := z.Next()
        if( tokenType == html.ErrorToken ) {
            break
        }

        //once we have a token we need to get all of the data form it before make an function calls to tbe tokenizer.  If we dont, the token *might* change.  I havent figured out which functions trigger a change and which do not.
        token := z.Token()

        //processTokenAttributes( token )

        //<a id="ctl00_ctl00_ctl00_MainContent_SubContent_SubContent_ctl00_cardEntries_ctl225_cardLink" class="nameLink" onclick="return CardLinkAction(event, this, &#39;SameWindow&#39;);" href="../Card/Details.aspx?multiverseid=87969">
        //get the multiverse ID link if its a part of this token
        bOnName := false
        for _, a := range token.Attr {
            if ((a.Key == "class") && (a.Val == "nameLink")) {
                bOnName = true
            }
            if ((a.Key == "href") && (bOnName == true)) {
                //this is really the start of a new card.
                currentCard = new(CardObject)

                //i should reall check to make sure we have the proper array size
                currentCard.MvID = strings.Split(a.Val, "=")[1]
            }
        }

        //this will return the Text between the current StartToken and EndToken
        tokenText := strings.TrimSpace( string( token.Data ) )

        //Get the name of the tag: a, span, table, etc.
        tokenTag := token.DataAtom.String()
        if((string(tokenTag) == "") && (tokenText == "")) {
            continue
        }

        //see the note after this swtich for its importance.
        switch tokenType {
            case html.StartTagToken: {
                switch string(tokenTag) {
                    case "table": {
                        tableDepth++
                    }
                    case "tr": {
                        rowDepth++
                    }
                    case "td": {
                        cellDepth++
                    }
                }
            }

            case html.EndTagToken: {
                switch string(tokenTag) {
                    case "table": {
                        tableDepth--
                        //rowDepth = 0
                    }
                    case "tr": {
                        rowDepth--
                        //cellDepth = 0
                    }
                    case "td": {
                        cellDepth--
                    }
                }
            }

            case html.ErrorToken: {
                fmt.Println("App Error:\n\n" + fmt.Sprint( "%v", z.Err() ))
                return -1
            }
        }

        //the row, cell, and tokenText are most important here.
        if( (tableDepth > 0) && (rowDepth > 0) && (cellDepth > 0) && (tokenText != "") && (string(tokenTag) == "") ) {

            tokenText = strings.TrimRight(tokenText, ":")

            switch tokenText {
                case "Name", "Cost", "Type", "Pow/Tgh", "Rules Text", "Set/Rarity": {
                    currentSwitch = tokenText
                }
                default: {

                    switch currentSwitch {
                        case "Name": {
                            currentCard.Name = tokenText
                        }
                        case "Cost": {
                            currentCard.Cost = tokenText
                        }
                        case "Type": {
                            currentCard.Type = tokenText
                        }
                        case "Pow/Tgh": {
                            currentCard.Power = tokenText
                        }
                        case "Rules Text": {
                            //for range append()
                            currentCard.Rules = strings.Split(tokenText, "\n")
                            for i, _ := range currentCard.Rules {
                                //trim the spaces
                                if currentCard.Rules[i][0] == 32 {
                                    currentCard.Rules[i] = currentCard.Rules[i][1:]
                                }
                            }
                        }
                        case "Set/Rarity": {
                            //at this stage the card is complete, commit to the file and clear the data

                            currentCard.Sets = strings.Split(tokenText, ",")
                            for i, _ := range currentCard.Sets {
                                //trim the spaces
                                if currentCard.Sets[i][0] == 32 {
                                    currentCard.Sets[i] = currentCard.Sets[i][1:]
                                }
                            }

                            //commit the card to the output file...
                            //_, e := output.WriteString( fmt.Sprintf("%s\r\n", currentCard.GetCardData() ) )
                            s, e := output.Write( currentCard.GetCardData() )
                            size += s
                            if(e != nil) {
                                fmt.Println("App Error:\n\n" + e.Error())
                                return -1
                            }

                            output.Flush()
                            //currentCard = new(CardObject)
                        }
                    }
                }
            }
            //fmt.Println( fmt.Sprintf("%dx%dx%d [%s] [%s]", tableDepth, rowDepth, cellDepth, tokenText, string(tokenTag) ) )
        }
    }

    return size
}

func main() {

    i := 0
    size := 0

    fmt.Println("\n\nThom is working... be patient; this may take a very long time.")

    size = GetGathererSource()
    fmt.Println(fmt.Sprintf("Ok: the source material is %d megabytes.", (size / (1024 * 1024) ) ) )

    size = AssembleSourceMaterial()
    fmt.Println(fmt.Sprintf("Ok: the assembled json data is %d megabytes.", (size / (1024 * 1024) ) ) )

    fmt.Println("Done: You may see the data mtg_data.txt.  Press ENTER to exit.")
    fmt.Scanln(&i)

    //len(os.Args) is always atleast 1 since the first argument is the name of the executable

    /*
    if(len(os.Args) < 2) {
        fmt.Println("Thom says:  What do you want to do?  update or assemble?")
        return
    }

    switch os.Args[1] {
        case "update": {
            GetGathererSource()
        }

        case "assemble": {
            AssembleSourceMaterial()
            fmt.Println("Thom says:  Done.  Have a look in mtg_data.txt")
            //fmt.Println("Thom says:  Assembling isnt currently imlemented.  Lazy ass developer.")
        }

        case "test": {

        }

        default: {
            fmt.Println(fmt.Sprintf("Thom says:  I don't understand '%s'", os.Args[1]))
            return

        }
    }
    */

}
