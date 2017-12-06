package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"testbox"
)

type TestResponse struct {
	Id          id     `json:"id"`
	Description string `json:"description"`
}

type SubmissionRequest struct {
	Id       id     `json:"id"`
	Language string `json:"language"`
	Code     string `json:"code"`
	Input    string `json:"input"`
}

type SubmissionResponse struct {
	Output map[string]string `json:"passFail"`
	Error  testbox.Message   `json:"error"`
}

type LanguagesResponse struct {
	Languages []string `json:"languages"`
}

var box testbox.TestBox

func main() {
	port := os.Getenv("TEST_BOX_PORT")

	box = testbox.New("data/compilers.json")

	http.HandleFunc("/", getTest)
	http.HandleFunc("/submit/", submitTest)
	http.HandleFunc("/stdout/", getStdout)
	http.HandleFunc("/languages/", getLangs)

	log.Println("TestBox listening on " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getTest(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for test...")
	testids := make([]id, len(challenges))
	i := 0
	for k := range challenges {
		testids[i] = k
		i++
	}

	n := rand.Intn(len(testids))
	testid := testids[n]
	test := challenges[testid]

	tr := TestResponse{testid, test.Description}
	json, _ := json.MarshalIndent(tr, "", "    ")

	log.Printf("Handing out test %s:\n%s", testid, test.Description)

	w.Header().Set("Content-Type", "application/json")
	w.Write(json)
}

func getStdout(w http.ResponseWriter, r *http.Request) {
	log.Println("Received request for stdout")
	decoder := json.NewDecoder(r.Body)
	var submission SubmissionRequest
	err := decoder.Decode(&submission)
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()

	output, msg := box.StdOut(submission.Language, submission.Code, submission.Input)
	log.Println(output, msg)

	buf, _ := json.MarshalIndent(SubmissionResponse{output, msg}, "", "   ")

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func submitTest(w http.ResponseWriter, r *http.Request) {
	log.Println("Received challenge submission")
	decoder := json.NewDecoder(r.Body)
	var submission SubmissionRequest
	err := decoder.Decode(&submission)
	if err != nil {
		panic(err)
	}
	defer r.Body.Close()

	test := challenges[submission.Id]
	stdin, stdout := test.StdIO()

	passed, msg := box.Test(submission.Language, submission.Code, stdin, stdout)
	log.Println(passed, msg)

	buf, _ := json.MarshalIndent(SubmissionResponse{passed, msg}, "", "   ")

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

func getLangs(w http.ResponseWriter, r *http.Request) {
	log.Println("Received languages request")
	langs := make([]string, 0)

	for k := range box.LanguageMap {
		langs = append(langs, k)
	}

	log.Println(langs)
	buf, _ := json.MarshalIndent(LanguagesResponse{langs}, "", "   ")

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf)
}

var challenges map[id]Test

func init() {
	log.Println("Reading challenges file...")
	bytes, err := ioutil.ReadFile("data/challenges.json")
	if err != nil {
		panic(err)
	}

	challenges = make(map[id]Test)
	err = json.Unmarshal(bytes, &challenges)
	if err != nil {
		panic(err)
	}
	log.Println("Challenges file loaded.")
}
