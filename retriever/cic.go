package retriever

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/peterjiz/gocic/email"
	"io/ioutil"
	"labix.org/v2/pipe"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"strings"
	"github.com/spf13/viper"
)

var cicBaseURL string = "https://services3.cic.gc.ca/ecas"
var casePrefix string = `<a href="viewcasehistory.do?id=`

//ID Type:
// 1 = Client ID # / UCI; 2 = Receipt Number (IMM 5401); 3 = Application Number / Case Number; 4 = Record of Landing Number;
// 5 = PR Card Number; 6 = Citizenship Receipt Number; 7 = Citizenship File Number / Group Number; 8 = Confirmation of PR Number

//Country Code:
// Go to https://services3.cic.gc.ca/ecas/authenticate.do?app=ecas
// View Page Source
// Find Country; <option value="COUNTRY_CODE_HERE">YOUR_COUNTRY</option>
type CICRequest struct {
	Id_Type  string
	ID       string // Identifier of that type (so the UCI number if '1' above, etc.)
	LastName string // Surname/Family Name:
	Dob      string // Date of birth (in YYYY-MM-DD format):
	Country  string
	Emails   []string
}

type CICFile struct {
	ID      string
	Name    string
	Status  string
	Details string
	Emails  []string
}

func getCookie() (string, error) {
	//Get Cookie
	requestURL := cicBaseURL + "/authenticate.do?app=ecas"
	request, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return "", err
	}
	client := &http.Client{}
	//client.Timeout = time.Duration(5 * time.Second)
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}

	//Cookie should look like "JSESSIONID=################################;"
	var cookie *http.Cookie = resp.Cookies()[0]
	var cookieString string = cookie.String()

	cookieValues, err := url.ParseQuery(cookieString)
	if err != nil {
		return "", err
	}

	cookieString = "JSESSIONID=" + cookieValues.Get("JSESSIONID") + ";"
	if cookieString == "" {
		return "", err
	}

	return cookieString, nil
}

func (requester *CICRequest) authenticate(cookieString string) error {
	//Authenticate
	auth_data := fmt.Sprintf("lang=&_page=_target0&app=ecas&identifierType=%v&identifier=%v&surname=%v&dateOfBirth=%v&countryOfBirth=%v&_submit=Continue", requester.Id_Type, requester.ID, requester.LastName, requester.Dob, requester.Country)

	reader := strings.NewReader(auth_data)
	requestURL := cicBaseURL + "/authenticate.do"
	request, err := http.NewRequest("POST", requestURL, reader)
	request.Header.Set("Cookie", cookieString)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if err != nil {
		return err
	}
	client := &http.Client{}
	_, err = client.Do(request)
	if err != nil {
		return err
	}

	return nil
}

func retrieveStatusPageHTML(cookieString string) (string, error) {
	//Retrieve Status Page as HTML
	var statusPageHTML string
	requestURL := cicBaseURL + "/viewcasestatus.do?app=ecas"
	request, err := http.NewRequest("GET", requestURL, nil)
	request.Header.Set("Cookie", cookieString)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if err != nil {
		return "", err
	}
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}

	//Extract Status Page
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	statusPageHTML = string(body)

	return statusPageHTML, nil
}

func retrieveDetailsPageHTML(cookieString, caseID string) (string, error) {
	//Request Details
	var data string
	requestURL := fmt.Sprintf("%v/viewcasehistory.do?id=%v&type=citCases&source=db&app=ecas&lang=en", cicBaseURL, caseID)
	request, err := http.NewRequest("GET", requestURL, nil)
	request.Header.Set("Cookie", cookieString)
	if err != nil {
		return "", err
	}
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	//Read data as HTML
	bs, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	data = string(bs)

	return data, nil
}

func extractCaseID(statusPageHTML string) (string, error) {
	// Run the pipeline
	line := pipe.Line(
		pipe.Exec("echo", statusPageHTML),
		pipe.Exec("grep", "-i", casePrefix),
		pipe.Exec("cut", "-d", "=", "-f", "3"),
		pipe.Exec("cut", "-d", "&", "-f", "1"),
	)
	out, err := pipe.Output(line)
	if err != nil {
		log.Printf("err: %#v", err.Error())
		return "", err
	}
	caseid := string(out)
	caseid = strings.TrimSpace(caseid)

	return caseid, nil
}

func extractCaseStatus(statusPageHTML, caseprefixAndcaseID string) (string, error) {
	//Extract Status
	line := pipe.Line(
		pipe.Exec("echo", statusPageHTML),
		pipe.Exec("grep", "-i", caseprefixAndcaseID),
		pipe.Exec("cut", "-d", ">", "-f", "2-99"),
		pipe.Exec("sed", "s|</b>|-|g"),
		pipe.Exec("sed", "s|<[^>]*>||g"),
	)
	out, err := pipe.Output(line)
	if err != nil {
		log.Printf("err: %#v", err.Error())
		return "", err
	}
	caseStatus := string(out)
	return caseStatus, nil
}

func extractCaseName(statusPageHTML, caseprefixAndcaseID string) (string, error) {
	//Extract Name
	line := pipe.Line(
		pipe.Exec("echo", statusPageHTML),
		pipe.Exec("grep", "-B", "4", caseprefixAndcaseID),
		pipe.Exec("head", "-n", "2"),
		pipe.Exec("awk", "{$2=$2};1"),
	)
	out, err := pipe.Output(line)
	if err != nil {
		log.Printf("err: %#v", err.Error())
		return "", err
	}
	whitespaceRegex, _ := regexp.Compile("\\n+")
	caseName := string(whitespaceRegex.ReplaceAll(out, []byte("")))
	caseName = strings.ToTitle(caseName)

	return caseName, nil

}

func extractDetails(detailsPageHTML string) (string, error) {
	//Process & Extract data
	line := pipe.Line(
		pipe.Exec("echo", detailsPageHTML),
		pipe.Exec("grep", "-i", `<li class="margin-bottom-medium">`),
		pipe.Exec("sed", "s|.*<li class=\"margin-bottom-medium\">|* |"),
		pipe.Exec("sed", "s|</b>|-|g"),
		pipe.Exec("sed", "s|<[^>]*>||g"),
	)
	out, err := pipe.Output(line)
	if err != nil {
		return "", err
	}
	details := string(out)
	return details, nil
}

func (requester *CICRequest) RetrieveCitizenshipFile() (CICFile, error) {

	cicFile := CICFile{}

	cookieString, err := getCookie()
	if err != nil {
		return cicFile, err
	}
	err = requester.authenticate(cookieString)
	if err != nil {
		return cicFile, err
	}
	statusPageHTML, err := retrieveStatusPageHTML(cookieString)
	if err != nil {
		return cicFile, err
	}

	caseid, err := extractCaseID(statusPageHTML)
	if err != nil {
		return cicFile, err
	}

	if caseid == "" {
		return cicFile, errors.New("Could not get Case ID - most likely authentication failed")
	} else {

		caseprefixAndcaseID := fmt.Sprintf("%v%v", casePrefix, caseid)
		caseStatus, err := extractCaseStatus(statusPageHTML, caseprefixAndcaseID)
		if err != nil {
			return cicFile, err
		}

		caseName, err := extractCaseName(statusPageHTML, caseprefixAndcaseID)
		if err != nil {
			return cicFile, err
		}

		detailsPageHTML, err := retrieveDetailsPageHTML(cookieString, caseid)
		if err != nil {
			return cicFile, err
		}
		details, err := extractDetails(detailsPageHTML)
		if err != nil {
			return cicFile, err
		}
		if details == "" {
			return cicFile, errors.New("Could not parse the details of case" + caseid)
		} else {

			cicFile.ID = caseid
			cicFile.Name = caseName
			cicFile.Status = caseStatus
			cicFile.Details = details
			cicFile.Emails = requester.Emails
			return cicFile, nil
		}
	}
}

func LoadFromFile(id string) (CICFile, error) {

	cicFile := CICFile{}

	fh, err := os.Open(id)
	if err != nil {
		return cicFile, err
	}
	dec := gob.NewDecoder(fh)
	err = dec.Decode(&cicFile)
	if err != nil {
		return cicFile, err
	}
	return cicFile, nil
}

func (cicFile *CICFile) SaveToFile() error {
	b := new(bytes.Buffer)
	enc := gob.NewEncoder(b)
	err := enc.Encode(*cicFile)
	if err != nil {
		return err
	}

	fh, eopen := os.OpenFile(cicFile.ID, os.O_CREATE|os.O_WRONLY, 0666)
	defer fh.Close()
	if eopen != nil {
		return eopen
	}
	_, e := fh.Write(b.Bytes())
	if e != nil {
		return e
	}

	return nil
}

func (cicFile *CICFile) CompareCitizenshipFiles(newFile *CICFile) bool {
	return cicFile.Status == newFile.Status && cicFile.Details == newFile.Details
}

func (cicFile *CICFile) SendEmail() error {

	viper.SetConfigType("json")
	viper.SetConfigName("mailserver") // name of config file (without extension)
	viper.AddConfigPath(".")          // optionally look for config in the working directory
	err := viper.ReadInConfig()       // Find and read the config file
	if err != nil {                   // Handle errors reading the config file
		return err
	}

	var sender email.EmailUser
	sender.Username = viper.GetString("Username")
	sender.Password = viper.GetString("Password")
	sender.EmailServer = viper.GetString("EmailServer")
	sender.Port = viper.GetInt("Port")

	//From
	from := mail.Address{fmt.Sprintf("%v Citizenship [Automated]", (strings.SplitN(cicFile.Name, " ", 1))[0]), sender.Username}

	//Subject
	subject := "Citizenship Update"

	//Body
	body := fmt.Sprintf("Name: %v\n\nStatus: %v\n\nDetails:\n%v", cicFile.Name, cicFile.Status, cicFile.Details)

	//Endname
	endname := "Peter's Automated System"

	err = sender.SendEmail(from, cicFile.Emails, subject, body, endname)
	if err != nil {
		return err
	}

	return nil

}

func (requester *CICRequest) TimedRefresh() error {

	newFile, err := requester.RetrieveCitizenshipFile()
	if err != nil {
		return err
	}

	//Retrieve Previously Saved File
	previousFile, loadErr := LoadFromFile(newFile.ID)

	//Save New File
	saveErr := newFile.SaveToFile()
	if saveErr != nil {
		return saveErr
	}

	//If previousfile load error, just send an email with new file
	if loadErr != nil {
		err = newFile.SendEmail()
		if err != nil {
			return err
		}
		return nil
	}

	//Compare 2 files
	equalFiles := newFile.CompareCitizenshipFiles(&previousFile)

	//If updated, notify me by email
	if !equalFiles {
		//Send an email
		err = newFile.SendEmail()
		if err != nil {
			return err
		}
		return nil

	}


	return nil

}

func (requester *CICRequest) ForcedRefresh() error {

	newFile, err := requester.RetrieveCitizenshipFile()
	if err != nil {
		return err
	}

	//Save New File
	err = newFile.SaveToFile()
	if err != nil {
		return err
	}

	//Send an email
	err = newFile.SendEmail()
	if err != nil {
		return err
	}
	return nil

}
