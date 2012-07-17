package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"pkg/util"
	"strconv"
)

var (
	server  = flag.String("server", "localhost:9333", "weedfs master location")
	IsDebug = flag.Bool("debug", false, "verbose debug information")
)

type AssignResult struct {
	Fid       string "fid"
	Url       string "url"
	PublicUrl string "publicUrl"
	Count     int    `json:",string"`
	Error     string "error"
}

func assign(count int) (AssignResult, error) {
	values := make(url.Values)
	values.Add("count", strconv.Itoa(count))
	jsonBlob := util.Post("http://"+*server+"/dir/assign", values)
	var ret AssignResult
	err := json.Unmarshal(jsonBlob, &ret)
	if err != nil {
		return ret, err
	}
	if ret.Count <= 0 {
		return ret, errors.New(ret.Error)
	}
	return ret, nil
}

type UploadResult struct {
	Size int
}

func upload(filename string, uploadUrl string) (int, string) {
	body_buf := bytes.NewBufferString("")
	body_writer := multipart.NewWriter(body_buf)
	file_writer, err := body_writer.CreateFormFile("file", filename)
	if err != nil {
		panic(err.Error())
	}
	fh, err := os.Open(filename)
	if err != nil {
		panic(err.Error())
	}
	io.Copy(file_writer, fh)
	content_type := body_writer.FormDataContentType()
	body_writer.Close()
	resp, err := http.Post(uploadUrl, content_type, body_buf)
	if err != nil {
		panic(err.Error())
	}
	defer resp.Body.Close()
	resp_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err.Error())
	}
	var ret UploadResult
	err = json.Unmarshal(resp_body, &ret)
	if err != nil {
		panic(err.Error())
	}
	//fmt.Println("Uploaded " + strconv.Itoa(ret.Size) + " Bytes to " + uploadUrl)
	return ret.Size, uploadUrl
}

type SubmitResult struct {
	Fid  string "fid"
	Size int    "size"
}

func submit(files []string)([]SubmitResult) {
	ret, err := assign(len(files))
	if err != nil {
		panic(err)
	}
	results := make([]SubmitResult, len(files))
	for index, file := range files {
		fid := ret.Fid
		if index > 0 {
			fid = fid + "_" + strconv.Itoa(index)
		}
		uploadUrl := "http://" + ret.PublicUrl + "/" + fid
		results[index].Size, _ = upload(file, uploadUrl)
		results[index].Fid = fid
	}
	return results
}

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
    fmt.Fprintln(os.Stderr, "Submit one file or multiple version of the same file.")
    fmt.Fprintf(os.Stderr, "Usage: %s -server=<host>:<port> file1 [file2 file3 ...]\n", os.Args[0])
		flag.Usage()
		return
	}
	results := submit(flag.Args())
	bytes, _ := json.Marshal(results)
	fmt.Print(string(bytes))
}
