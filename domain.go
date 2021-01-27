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

import "sync"

type Indexes map[string]interface{}

type Document struct {
	Index   string                 `json:"_index,omitempty"`
	Type    string                 `json:"_type,omitempty"`
	Id      string                 `json:"_id,omitempty"`
	source  map[string]interface{} `json:"_source,omitempty"`
	Routing string                 `json:"routing,omitempty"` //after 6, only `routing` was supported
}

type Scroll struct {
	Took     int    `json:"took,omitempty"`
	ScrollId string `json:"_scroll_id,omitempty"`
	TimedOut bool   `json:"timed_out,omitempty"`
	Hits     struct {
		MaxScore float32       `json:"max_score,omitempty"`
		Total    int           `json:"total,omitempty"`
		Docs     []interface{} `json:"hits,omitempty"`
	} `json:"hits"`
	Shards struct {
		Total      int `json:"total,omitempty"`
		Successful int `json:"successful,omitempty"`
		Skipped    int `json:"skipped,omitempty"`
		Failed     int `json:"failed,omitempty"`
		Failures   []struct {
			Shard  int         `json:"shard,omitempty"`
			Index  string      `json:"index,omitempty"`
			Status int         `json:"status,omitempty"`
			Reason interface{} `json:"reason,omitempty"`
		} `json:"failures,omitempty"`
	} `json:"_shards,omitempty"`
}

type ScrollV7 struct {
	Scroll
	Hits struct {
		MaxScore float32 `json:"max_score,omitempty"`
		Total    struct {
			Value    int    `json:"value,omitempty"`
			Relation string `json:"relation,omitempty"`
		} `json:"total,omitempty"`
		Docs []interface{} `json:"hits,omitempty"`
	} `json:"hits"`
}

type ClusterVersion struct {
	Name        string `json:"name,omitempty"`
	ClusterName string `json:"cluster_name,omitempty"`
	Version     struct {
		Number        string `json:"number,omitempty"`
		LuceneVersion string `json:"lucene_version,omitempty"`
	} `json:"version,omitempty"`
}

type ClusterHealth struct {
	Name   string `json:"cluster_name,omitempty"`
	Status string `json:"status,omitempty"`
}

//{"took":23,"errors":true,"items":[{"create":{"_index":"mybank3","_type":"my_doc2","_id":"AWz8rlgUkzP-cujdA_Fv","status":409,"error":{"type":"version_conflict_engine_exception","reason":"[AWz8rlgUkzP-cujdA_Fv]: version conflict, document already exists (current version [1])","index_uuid":"w9JZbJkfSEWBI-uluWorgw","shard":"0","index":"mybank3"}}},{"create":{"_index":"mybank3","_type":"my_doc4","_id":"AWz8rpF2kzP-cujdA_Fx","status":400,"error":{"type":"illegal_argument_exception","reason":"Rejecting mapping update to [mybank3] as the final mapping would have more than 1 type: [my_doc2, my_doc4]"}}},{"create":{"_index":"mybank3","_type":"my_doc1","_id":"AWz8rjpJkzP-cujdA_Fu","status":400,"error":{"type":"illegal_argument_exception","reason":"Rejecting mapping update to [mybank3] as the final mapping would have more than 1 type: [my_doc2, my_doc1]"}}},{"create":{"_index":"mybank3","_type":"my_doc3","_id":"AWz8rnbckzP-cujdA_Fw","status":400,"error":{"type":"illegal_argument_exception","reason":"Rejecting mapping update to [mybank3] as the final mapping would have more than 1 type: [my_doc2, my_doc3]"}}},{"create":{"_index":"mybank3","_type":"my_doc5","_id":"AWz8rrsEkzP-cujdA_Fy","status":400,"error":{"type":"illegal_argument_exception","reason":"Rejecting mapping update to [mybank3] as the final mapping would have more than 1 type: [my_doc2, my_doc5]"}}},{"create":{"_index":"mybank3","_type":"doc","_id":"3","status":400,"error":{"type":"illegal_argument_exception","reason":"Rejecting mapping update to [mybank3] as the final mapping would have more than 1 type: [my_doc2, doc]"}}}]}
type BulkResponse struct {
	Took   int                 `json:"took,omitempty"`
	Errors bool                `json:"errors,omitempty"`
	Items  []map[string]Action `json:"items,omitempty"`
}

type Action struct {
	Index  string      `json:"_index,omitempty"`
	Type   string      `json:"_type,omitempty"`
	Id     string      `json:"_id,omitempty"`
	Status int         `json:"status,omitempty"`
	Error  interface{} `json:"error,omitempty"`
}

type Migrator struct {
	FlushLock   sync.Mutex
	DocChan     chan map[string]interface{}
	SourceESAPI ESAPI
	TargetESAPI ESAPI
	SourceAuth  *Auth
	TargetAuth  *Auth
	Config      *Config
}

type Config struct {

	// config options
	SourceEs            string `short:"s" long:"source"  description:"source elasticsearch instance, ie: http://localhost:9200"`
	Query               string `short:"q" long:"query"  description:"query against source elasticsearch instance, filter data before migrate, ie: name:medcl"`
	TargetEs            string `short:"d" long:"dest"    description:"destination elasticsearch instance, ie: http://localhost:9201"`
	SourceEsAuthStr     string `short:"m" long:"source_auth"  description:"basic auth of source elasticsearch instance, ie: user:pass"`
	TargetEsAuthStr     string `short:"n" long:"dest_auth"  description:"basic auth of target elasticsearch instance, ie: user:pass"`
	DocBufferCount      int    `short:"c" long:"count"   description:"number of documents at a time: ie \"size\" in the scroll request" default:"10000"`
	BufferCount      int    `long:"buffer_count"   description:"number of buffered documents in memory" default:"1000000"`
	Workers             int    `short:"w" long:"workers" description:"concurrency number for bulk workers" default:"1"`
	BulkSizeInMB        int    `short:"b" long:"bulk_size" description:"bulk size in MB" default:"5"`
	ScrollTime          string `short:"t" long:"time"    description:"scroll time" default:"10m"`
	ScrollSliceSize     int    `long:"sliced_scroll_size"    description:"size of sliced scroll, to make it work, the size should be > 1" default:"1"`
	RecreateIndex       bool   `short:"f" long:"force"   description:"delete destination index before copying"`
	CopyAllIndexes      bool   `short:"a" long:"all"     description:"copy indexes starting with . and _"`
	CopyIndexSettings   bool   `long:"copy_settings"          description:"copy index settings from source"`
	CopyIndexMappings   bool   `long:"copy_mappings"          description:"copy index mappings from source"`
	ShardsCount         int    `long:"shards"            description:"set a number of shards on newly created indexes"`
	SourceIndexNames    string `short:"x" long:"src_indexes" description:"indexes name to copy,support regex and comma separated list" default:"_all"`
	TargetIndexName     string `short:"y" long:"dest_index" description:"indexes name to save, allow only one indexname, original indexname will be used if not specified" default:""`
	OverrideTypeName    string `short:"u" long:"type_override" description:"override type name" default:""`
	WaitForGreen        bool   `long:"green"             description:"wait for both hosts cluster status to be green before dump. otherwise yellow is okay"`
	LogLevel            string `short:"v" long:"log"            description:"setting log level,options:trace,debug,info,warn,error"  default:"INFO"`
	DumpOutFile         string `short:"o" long:"output_file"            description:"output documents of source index into local file" `
	DumpInputFile       string `short:"i" long:"input_file"            description:"indexing from local dump file" `
	InputFileType       string `long:"input_file_type"                 description:"the data type of input file, options: dump, json_line, json_array, log_line" default:"dump" `
	SourceProxy         string `long:"source_proxy"            description:"set proxy to source http connections, ie: http://127.0.0.1:8080"`
	TargetProxy         string `long:"dest_proxy"            description:"set proxy to target http connections, ie: http://127.0.0.1:8080"`
	Refresh             bool   `long:"refresh"                 description:"refresh after migration finished"`
	Fields              string `long:"fields"                 description:"filter source fields, comma separated, ie: col1,col2,col3,..." `
	RenameFields        string `long:"rename"                 description:"rename source fields, comma separated, ie: _type:type, name:myname" `
	LogstashEndpoint    string `short:"l"  long:"logstash_endpoint"    description:"target logstash tcp endpoint, ie: 127.0.0.1:5055" `
	LogstashSecEndpoint bool   `long:"secured_logstash_endpoint"    description:"target logstash tcp endpoint was secured by TLS" `

	RepeatOutputTimes         int  `long:"repeat_times"            description:"repeat the data from source N times to dest output, use align with parameter regenerate_id to amplify the data size "`
	RegenerateID              bool `short:"r" long:"regenerate_id"   description:"regenerate id for documents, this will override the exist document id in data source"`
	Compress                  bool `long:"compress"            description:"use gzip to compress traffic"`
	SleepSecondsAfterEachBulk int  `short:"p" long:"sleep" description:"sleep N seconds after each bulk request" default:"-1"`
}

type Auth struct {
	User string
	Pass string
}
