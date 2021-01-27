/*
Copyright 2016 Medcl (m AT medcl.net)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/parnurzeal/gorequest"
	"github.com/valyala/fasthttp"
	//"infini.sh/framework/lib/fasthttp"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

func BasicAuth(req *fasthttp.Request,user,pass string) {
	msg := fmt.Sprintf("%s:%s",user,pass)
	encoded := base64.StdEncoding.EncodeToString([]byte(msg))
	req.Header.Add("Authorization", "Basic "+encoded)
}

func Get(url string,auth *Auth,proxy string) (*http.Response, string, []error) {
	request := gorequest.New()

	tr := &http.Transport{
		DisableKeepAlives: true,
		DisableCompression: false,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	request.Transport=tr


	if auth!=nil {
		request.SetBasicAuth(auth.User,auth.Pass)
	}
	s := make([]string, 1)
	s[0] = "application/json"
	request.Header["Content-Type"]= s

	if len(proxy)>0 {
		request.Proxy(proxy)
	}

	resp, body, errs := request.Get(url).End()
	return resp, body, errs

}

func Post(url string,auth *Auth, body string,proxy string)(*http.Response, string, []error)  {
	request := gorequest.New()
	tr := &http.Transport{
		DisableKeepAlives: true,
		DisableCompression: false,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	request.Transport=tr

	s := make([]string, 1)
	s[0] = "application/json"
	if auth!=nil {
		request.SetBasicAuth(auth.User,auth.Pass)
	}

	request.Header["Content-Type"]=s
	
	if len(proxy)>0 {
		request.Proxy(proxy)
	}

	request.Post(url)

	if len(body)>0 {
		request.Send(body)
	}

	return request.End()
}

func newDeleteRequest(client *http.Client,method, urlStr string) (*http.Request, error) {
	if method == "" {
		// We document that "" means "GET" for Request.Method, and people have
		// relied on that from NewRequest, so keep that working.
		// We still enforce validMethod for non-empty methods.
		method = "GET"
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	req := &http.Request{
		Method:     method,
		URL:        u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       u.Host,
	}
	return req, nil
}

//
//func GzipHandler(req *http.Request) {
//	var b bytes.Buffer
//	var buf bytes.Buffer
//	g := gzip.NewWriter(&buf)
//
//	_, err := io.Copy(g, &b)
//	if err != nil {
//		panic(err)
//		//slog.Error(err)
//		return
//	}
//}

var client *http.Client=&http.Client{
	Transport: &http.Transport{
		DisableKeepAlives: true,
		DisableCompression: false,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}
var fastHttpClient = &fasthttp.Client{
	TLSConfig: &tls.Config{InsecureSkipVerify: true},
}

func DoRequest(compress bool,method string,loadUrl string,auth *Auth,body []byte,proxy string) (string,error)  {

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	req.SetRequestURI(loadUrl)
	req.Header.SetMethod(method)

	req.Header.Set("Content-Type", "application/json")


	if auth!=nil{
		req.URI().SetUsername(auth.User)
		req.URI().SetPassword(auth.Pass)
	}

	if len(body)>0{
		if compress{
			_, err := fasthttp.WriteGzipLevel(req.BodyWriter(), body, fasthttp.CompressBestSpeed)
			if err != nil {
				panic(err)
			}
		}else{
			req.SetBody(body)

		}
	}

	err:=fastHttpClient.Do(req, resp)

	if err != nil {
			panic(err)
	}
	if resp == nil {
		panic("empty response")
	}

	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusCreated {

	} else {
		//fmt.Println("received status code", resp.StatusCode, "from", string(resp.Header.Header()), "content", string(resp.Body()), req)
	}



	//if compress{
	//	data,err:= resp.BodyGunzip()
	//	return string(data),err
	//}

	return string(resp.Body()),nil
}


func Request(method string,r string,auth *Auth,body *bytes.Buffer,proxy string)(string,error)  {

	//TODO use global client
	//client = &http.Client{}
	//
	//if(len(proxy)>0){
	//	proxyURL, err := url.Parse(proxy)
	//	if(err!=nil){
	//		log.Error(err)
	//	}else{
	//		transport := &http.Transport{
	//			Proxy: http.ProxyURL(proxyURL),
	//			DisableKeepAlives: true,
	//			DisableCompression: false,
	//		}
	//		client = &http.Client{Transport: transport}
	//	}
	//}
	//
	//tr := &http.Transport{
	//	DisableKeepAlives: true,
	//	DisableCompression: false,
	//	TLSClientConfig: &tls.Config{
	//		InsecureSkipVerify: true,
	//},
	//}
	//
	//client.Transport=tr

	var err error
	var reqest *http.Request
	if body!=nil {
		reqest, err =http.NewRequest(method,r,body)
	}else{
		reqest, err = newDeleteRequest(client,method,r)
	}

	if err!=nil {
		panic(err)
	}

	if auth!=nil {
		reqest.SetBasicAuth(auth.User,auth.Pass)
	}

	reqest.Header.Set("Content-Type", "application/json")

	//enable gzip
	//reqest.Header.Set("Content-Encoding", "gzip")
	//GzipHandler(reqest)
	//

	resp,errs := client.Do(reqest)
	if errs != nil {
		log.Error(errs)
		return "",errs
	}

	if resp!=nil&& resp.Body!=nil{
		//io.Copy(ioutil.Discard, resp.Body)
		defer resp.Body.Close()
	}

	if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		return "",errors.New("server error: "+string(b))
	}

	respBody,err:=ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Error(err)
		return string(respBody),err
	}

	//if err != nil {
	//	return string(respBody),err
	//}
	io.Copy(ioutil.Discard, resp.Body)
	defer resp.Body.Close()
	return string(respBody),nil
}

func DecodeJson(jsonStream string, o interface{}) error {
	decoder := json.NewDecoder(strings.NewReader(jsonStream))
	// UseNumber causes the Decoder to unmarshal a number into an interface{} as a Number instead of as a float64.
	decoder.UseNumber()

	if err := decoder.Decode(o); err != nil {
		fmt.Println("error:", err)
		return err
	}
	return nil
}

func DecodeJsonBytes(jsonStream []byte, o interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(jsonStream))
	// UseNumber causes the Decoder to unmarshal a number into an interface{} as a Number instead of as a float64.
	decoder.UseNumber()

	if err := decoder.Decode(o); err != nil {
		fmt.Println("error:", err)
		return err
	}
	return nil
}
