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
	"encoding/json"
	"github.com/cheggaaa/pb"
	log "github.com/cihub/seelog"
)


type ScrollAPI interface{
	GetScrollId()string
	GetHitsTotal()int
	GetDocs() []interface{}
	ProcessScrollResult(c *Migrator, bar *pb.ProgressBar)
	Next(c *Migrator, bar *pb.ProgressBar) (done bool)
}


func (scroll *Scroll) GetHitsTotal()int{
	return scroll.Hits.Total
}

func (scroll *Scroll) GetScrollId()string{
	return scroll.ScrollId
}

func (scroll *Scroll) GetDocs()[]interface{}{
	return scroll.Hits.Docs
}

func (scroll *ScrollV7) GetHitsTotal()int{
	return scroll.Hits.Total.Value
}


func (scroll *ScrollV7) GetScrollId()string{
	return scroll.ScrollId
}

func (scroll *ScrollV7) GetDocs()[]interface{}{
	return scroll.Hits.Docs
}


// Stream from source es instance. "done" is an indicator that the stream is
// over
func (s *Scroll) ProcessScrollResult(c *Migrator, bar *pb.ProgressBar){

	//update progress bar
	bar.Add(len(s.Hits.Docs))

	// show any failures
	for _, failure := range s.Shards.Failures {
		reason, _ := json.Marshal(failure.Reason)
		log.Errorf(string(reason))
	}

	// write all the docs into a channel
	for _, docI := range s.Hits.Docs {
		c.DocChan <- docI.(map[string]interface{})
	}
}

func (s *Scroll) Next(c *Migrator, bar *pb.ProgressBar) (done bool) {

	scroll,err:=c.SourceESAPI.NextScroll(c.Config.ScrollTime,s.ScrollId)
	if err != nil {
		log.Error(err)
		return false
	}

	docs:=scroll.(ScrollAPI).GetDocs()
	if docs == nil || len(docs) <= 0 {
		log.Debug("scroll result is empty")
		return true
	}

	scroll.(ScrollAPI).ProcessScrollResult(c,bar)

	//update scrollId
	s.ScrollId=scroll.(ScrollAPI).GetScrollId()

	return
}

// Stream from source es instance. "done" is an indicator that the stream is
// over
func (s *ScrollV7) ProcessScrollResult(c *Migrator, bar *pb.ProgressBar){

	//update progress bar
	bar.Add(len(s.Hits.Docs))

	// show any failures
	for _, failure := range s.Shards.Failures {
		reason, _ := json.Marshal(failure.Reason)
		log.Errorf(string(reason))
	}

	// write all the docs into a channel
	for _, docI := range s.Hits.Docs {
		c.DocChan <- docI.(map[string]interface{})
	}
}

func (s *ScrollV7) Next(c *Migrator, bar *pb.ProgressBar) (done bool) {

	scroll,err:=c.SourceESAPI.NextScroll(c.Config.ScrollTime,s.ScrollId)
	if err != nil {
		log.Error(err)
		return false
	}

	docs:=scroll.(ScrollAPI).GetDocs()
	if docs == nil || len(docs) <= 0 {
		log.Debug("scroll result is empty")
		return true
	}

	scroll.(ScrollAPI).ProcessScrollResult(c,bar)

	//update scrollId
	s.ScrollId=scroll.(ScrollAPI).GetScrollId()

	return
}


