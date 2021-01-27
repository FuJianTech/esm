package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/cheggaaa/pb"
	log "github.com/cihub/seelog"
	goflags "github.com/jessevdk/go-flags"
	"github.com/mattn/go-isatty"
	"io"
	"io/ioutil"
	_ "net/http/pprof"
	"os"
	"runtime"
	_ "runtime/pprof"
	"strings"
	"sync"
	"time"
)

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	//go func() {
	//	//log.Infof("pprof listen at: http://%s/debug/pprof/", app.httpprof)
	//	mux := http.NewServeMux()
	//
	//	// register pprof handler
	//	mux.HandleFunc("/debug/pprof/", func(w http.ResponseWriter, r *http.Request) {
	//		http.DefaultServeMux.ServeHTTP(w, r)
	//	})
	//
	//	// register metrics handler
	//	//mux.HandleFunc("/debug/vars", app.metricsHandler)
	//
	//	endpoint := http.ListenAndServe("0.0.0.0:6060", mux)
	//	log.Debug("stop pprof server: %v", endpoint)
	//}()

	var err error
	c := &Config{}
	migrator := Migrator{}
	migrator.Config = c

	// parse args
	_, err = goflags.Parse(c)
	if err != nil {
		log.Error(err)
		return
	}

	setInitLogging(c.LogLevel)

	if len(c.SourceEs) == 0 && len(c.DumpInputFile) == 0 {
		log.Error("no input, type --help for more details")
		return
	}
	if len(c.TargetEs) == 0 && len(c.DumpOutFile) == 0 {
		log.Error("no output, type --help for more details")
		return
	}

	if c.SourceEs == c.TargetEs && c.SourceIndexNames == c.TargetIndexName {
		log.Error("migration output is the same as the output")
		return
	}

	var showBar bool = false
	if isatty.IsTerminal(os.Stdout.Fd()) {
		showBar = true
	} else if isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		showBar = true
	} else {
		showBar = false
	}

	//至少输出一次
	if c.RepeatOutputTimes < 1 {
		c.RepeatOutputTimes=1
	}else{
		log.Info("source data will repeat send to target: ", c.RepeatOutputTimes, " times, the document id will be regenerated.")
	}

	if c.RepeatOutputTimes > 0 {

		for i := 0; i < c.RepeatOutputTimes; i++ {

			if c.RepeatOutputTimes>1 {
				log.Info("repeat round: ", i+1)
			}

			// enough of a buffer to hold all the search results across all workers
			migrator.DocChan = make(chan map[string]interface{}, c.BufferCount)

			var srcESVersion *ClusterVersion
			// create a progressbar and start a docCount
			var outputBar *pb.ProgressBar = pb.New(1).Prefix("Output ")

			var fetchBar = pb.New(1).Prefix("Scroll")

			wg := sync.WaitGroup{}

			//dealing with input
			if len(c.SourceEs) > 0 {
				//dealing with basic auth
				if len(c.SourceEsAuthStr) > 0 && strings.Contains(c.SourceEsAuthStr, ":") {
					authArray := strings.Split(c.SourceEsAuthStr, ":")
					auth := Auth{User: authArray[0], Pass: authArray[1]}
					migrator.SourceAuth = &auth
				}

				//get source es version
				srcESVersion, errs := migrator.ClusterVersion(c.SourceEs, migrator.SourceAuth, migrator.Config.SourceProxy)
				if errs != nil {
					return
				}
				if strings.HasPrefix(srcESVersion.Version.Number, "7.") {
					log.Debug("source es is V7,", srcESVersion.Version.Number)
					api := new(ESAPIV7)
					api.Host = c.SourceEs
					api.Compress=c.Compress
					api.Auth = migrator.SourceAuth
					api.HttpProxy = migrator.Config.SourceProxy
					migrator.SourceESAPI = api
				} else if strings.HasPrefix(srcESVersion.Version.Number, "6.") {
					log.Debug("source es is V6,", srcESVersion.Version.Number)
					api := new(ESAPIV6)
					api.Compress=c.Compress
					api.Host = c.SourceEs
					api.Auth = migrator.SourceAuth
					api.HttpProxy = migrator.Config.SourceProxy
					migrator.SourceESAPI = api
				} else if strings.HasPrefix(srcESVersion.Version.Number, "5.") {
					log.Debug("source es is V5,", srcESVersion.Version.Number)
					api := new(ESAPIV5)
					api.Host = c.SourceEs
					api.Compress=c.Compress
					api.Auth = migrator.SourceAuth
					api.HttpProxy = migrator.Config.SourceProxy
					migrator.SourceESAPI = api
				} else {
					log.Debug("source es is not V5,", srcESVersion.Version.Number)
					api := new(ESAPIV0)
					api.Host = c.SourceEs
					api.Compress=c.Compress
					api.Auth = migrator.SourceAuth
					api.HttpProxy = migrator.Config.SourceProxy
					migrator.SourceESAPI = api
				}

				if c.ScrollSliceSize < 1 {
					c.ScrollSliceSize = 1
				}

				totalSize := 0
				finishedSlice := 0
				for slice := 0; slice < c.ScrollSliceSize; slice++ {
					scroll, err := migrator.SourceESAPI.NewScroll(c.SourceIndexNames, c.ScrollTime, c.DocBufferCount, c.Query, slice, c.ScrollSliceSize, c.Fields)
					if err != nil {
						log.Error(err)
						return
					}

					temp := scroll.(ScrollAPI)

					totalSize += temp.GetHitsTotal()

					if scroll != nil && temp.GetDocs() != nil {

						if temp.GetHitsTotal() == 0 {
							log.Error("can't find documents from source.")
							return
						}

						go func() {
							wg.Add(1)
							//process input
							// start scroll
							temp.ProcessScrollResult(&migrator, fetchBar)

							// loop scrolling until done
							for temp.Next(&migrator, fetchBar) == false {
							}

							if showBar {
								fetchBar.Finish()
							}

							// finished, close doc chan and wait for goroutines to be done
							wg.Done()
							finishedSlice++

							//clean up final results
							if finishedSlice == c.ScrollSliceSize {
								log.Debug("closing doc chan")
								close(migrator.DocChan)
							}
						}()
					}
				}

				if totalSize > 0 {
					fetchBar.Total = int64(totalSize)
					outputBar.Total = int64(totalSize)
				}

			} else if len(c.DumpInputFile) > 0 {
				//read file stream
				wg.Add(1)
				f, err := os.Open(c.DumpInputFile)
				if err != nil {
					log.Error(err)
					return
				}
				//get file lines
				lineCount := 0
				defer f.Close()
				r := bufio.NewReader(f)
				for {
					_, err := r.ReadString('\n')
					if io.EOF == err || nil != err {
						break
					}
					lineCount += 1
				}
				log.Trace("file line,", lineCount)

				fetchBar := pb.New(lineCount).Prefix("Read")
				outputBar = pb.New(lineCount).Prefix("Output ")

				f.Close()

				go migrator.NewFileReadWorker(fetchBar, &wg)

			}

			var pool *pb.Pool
			if showBar {

				// start pool
				pool, err = pb.StartPool(fetchBar, outputBar)
				if err != nil {
					panic(err)
				}
			}

			//dealing with output
			if len(c.TargetEs) > 0 {
				if len(c.TargetEsAuthStr) > 0 && strings.Contains(c.TargetEsAuthStr, ":") {
					authArray := strings.Split(c.TargetEsAuthStr, ":")
					auth := Auth{User: authArray[0], Pass: authArray[1]}
					migrator.TargetAuth = &auth
				}

				//get target es version
				descESVersion, errs := migrator.ClusterVersion(c.TargetEs, migrator.TargetAuth, migrator.Config.TargetProxy)
				if errs != nil {
					return
				}

				if strings.HasPrefix(descESVersion.Version.Number, "7.") {
					log.Debug("target es is V7,", descESVersion.Version.Number)
					api := new(ESAPIV7)
					api.Host = c.TargetEs
					api.Auth = migrator.TargetAuth
					api.HttpProxy = migrator.Config.TargetProxy
					migrator.TargetESAPI = api
				} else if strings.HasPrefix(descESVersion.Version.Number, "6.") {
					log.Debug("target es is V6,", descESVersion.Version.Number)
					api := new(ESAPIV6)
					api.Host = c.TargetEs
					api.Auth = migrator.TargetAuth
					api.HttpProxy = migrator.Config.TargetProxy
					migrator.TargetESAPI = api
				} else if strings.HasPrefix(descESVersion.Version.Number, "5.") {
					log.Debug("target es is V5,", descESVersion.Version.Number)
					api := new(ESAPIV5)
					api.Host = c.TargetEs
					api.Auth = migrator.TargetAuth
					api.HttpProxy = migrator.Config.TargetProxy
					migrator.TargetESAPI = api
				} else {
					log.Debug("target es is not V5,", descESVersion.Version.Number)
					api := new(ESAPIV0)
					api.Host = c.TargetEs
					api.Auth = migrator.TargetAuth
					api.HttpProxy = migrator.Config.TargetProxy
					migrator.TargetESAPI = api

				}

				log.Debug("start process with mappings")
				if srcESVersion != nil && c.CopyIndexMappings && descESVersion.Version.Number[0] != srcESVersion.Version.Number[0] {
					log.Error(srcESVersion.Version, "=>", descESVersion.Version, ",cross-big-version mapping migration not avaiable, please update mapping manually :(")
					return
				}

				// wait for cluster state to be okay before moving
				timer := time.NewTimer(time.Second * 3)

				for {
					if len(c.SourceEs) > 0 {
						if status, ready := migrator.ClusterReady(migrator.SourceESAPI); !ready {
							log.Infof("%s at %s is %s, delaying migration ", status.Name, c.SourceEs, status.Status)
							<-timer.C
							continue
						}
					}

					if len(c.TargetEs) > 0 {
						if status, ready := migrator.ClusterReady(migrator.TargetESAPI); !ready {
							log.Infof("%s at %s is %s, delaying migration ", status.Name, c.TargetEs, status.Status)
							<-timer.C
							continue
						}
					}
					timer.Stop()
					break
				}

				if len(c.SourceEs) > 0 {
					// get all indexes from source
					indexNames, indexCount, sourceIndexMappings, err := migrator.SourceESAPI.GetIndexMappings(c.CopyAllIndexes, c.SourceIndexNames)

					if err != nil {
						log.Error(err)
						return
					}

					sourceIndexRefreshSettings := map[string]interface{}{}

					log.Debugf("indexCount: %d", indexCount)

					if indexCount > 0 {
						//override indexnames to be copy
						c.SourceIndexNames = indexNames

						// copy index settings if user asked
						if c.CopyIndexSettings || c.ShardsCount > 0 {
							log.Info("start settings/mappings migration..")

							//get source index settings
							var sourceIndexSettings *Indexes
							sourceIndexSettings, err := migrator.SourceESAPI.GetIndexSettings(c.SourceIndexNames)
							log.Debug("source index settings:", sourceIndexSettings)
							if err != nil {
								log.Error(err)
								return
							}

							//get target index settings
							targetIndexSettings, err := migrator.TargetESAPI.GetIndexSettings(c.TargetIndexName)
							if err != nil {
								//ignore target es settings error
								log.Debug(err)
							}
							log.Debug("target IndexSettings", targetIndexSettings)

							//if there is only one index and we specify the dest indexname
							if c.SourceIndexNames != c.TargetIndexName && (len(c.TargetIndexName) > 0) && indexCount == 1 {
								log.Debugf("only one index,so we can rewrite indexname, src:%v, dest:%v ,indexCount:%d", c.SourceIndexNames, c.TargetIndexName, indexCount)
								(*sourceIndexSettings)[c.TargetIndexName] = (*sourceIndexSettings)[c.SourceIndexNames]
								delete(*sourceIndexSettings, c.SourceIndexNames)
								log.Debug(sourceIndexSettings)
							}

							// dealing with indices settings
							for name, idx := range *sourceIndexSettings {
								log.Debug("dealing with index,name:", name, ",settings:", idx)
								tempIndexSettings := getEmptyIndexSettings()

								targetIndexExist := false
								//if target index settings is exist and we don't copy settings, we use target settings
								if targetIndexSettings != nil {
									//if target es have this index and we dont copy index settings
									if val, ok := (*targetIndexSettings)[name]; ok {
										targetIndexExist = true
										tempIndexSettings = val.(map[string]interface{})
									}

									if c.RecreateIndex {
										migrator.TargetESAPI.DeleteIndex(name)
										targetIndexExist = false
									}
								}

								//copy index settings
								if c.CopyIndexSettings {
									tempIndexSettings = ((*sourceIndexSettings)[name]).(map[string]interface{})
								}

								//check map elements
								if _, ok := tempIndexSettings["settings"]; !ok {
									tempIndexSettings["settings"] = map[string]interface{}{}
								}

								if _, ok := tempIndexSettings["settings"].(map[string]interface{})["index"]; !ok {
									tempIndexSettings["settings"].(map[string]interface{})["index"] = map[string]interface{}{}
								}

								sourceIndexRefreshSettings[name] = ((*sourceIndexSettings)[name].(map[string]interface{}))["settings"].(map[string]interface{})["index"].(map[string]interface{})["refresh_interval"]

								//set refresh_interval
								tempIndexSettings["settings"].(map[string]interface{})["index"].(map[string]interface{})["refresh_interval"] = -1
								tempIndexSettings["settings"].(map[string]interface{})["index"].(map[string]interface{})["number_of_replicas"] = 0

								//clean up settings
								delete(tempIndexSettings["settings"].(map[string]interface{})["index"].(map[string]interface{}), "number_of_shards")

								//copy indexsettings and mappings
								if targetIndexExist {
									log.Debug("update index with settings,", name, tempIndexSettings)
									//override shard settings
									if c.ShardsCount > 0 {
										tempIndexSettings["settings"].(map[string]interface{})["index"].(map[string]interface{})["number_of_shards"] = c.ShardsCount
									}
									err := migrator.TargetESAPI.UpdateIndexSettings(name, tempIndexSettings)
									if err != nil {
										log.Error(err)
									}
								} else {

									//override shard settings
									if c.ShardsCount > 0 {
										tempIndexSettings["settings"].(map[string]interface{})["index"].(map[string]interface{})["number_of_shards"] = c.ShardsCount
									}

									log.Debug("create index with settings,", name, tempIndexSettings)
									err := migrator.TargetESAPI.CreateIndex(name, tempIndexSettings)
									if err != nil {
										log.Error(err)
									}

								}

							}

							if c.CopyIndexMappings {

								//if there is only one index and we specify the dest indexname
								if c.SourceIndexNames != c.TargetIndexName && (len(c.TargetIndexName) > 0) && indexCount == 1 {
									log.Debugf("only one index,so we can rewrite indexname, src:%v, dest:%v ,indexCount:%d", c.SourceIndexNames, c.TargetIndexName, indexCount)
									(*sourceIndexMappings)[c.TargetIndexName] = (*sourceIndexMappings)[c.SourceIndexNames]
									delete(*sourceIndexMappings, c.SourceIndexNames)
									log.Debug(sourceIndexMappings)
								}

								for name, mapping := range *sourceIndexMappings {
									err := migrator.TargetESAPI.UpdateIndexMapping(name, mapping.(map[string]interface{})["mappings"].(map[string]interface{}))
									if err != nil {
										log.Error(err)
									}
								}
							}

							log.Info("settings/mappings migration finished.")
						}

					} else {
						log.Error("index not exists,", c.SourceIndexNames)
						return
					}

					defer migrator.recoveryIndexSettings(sourceIndexRefreshSettings)
				} else if len(c.DumpInputFile) > 0 {
					//check shard settings
					//TODO support shard config
				}

			}

			log.Info("start data migration..")

			//start es bulk thread
			if len(c.TargetEs) > 0 {
				log.Debug("start es bulk workers")
				outputBar.Prefix("Bulk")
				var docCount int
				wg.Add(c.Workers)
				for i := 0; i < c.Workers; i++ {
					go migrator.NewBulkWorker(&docCount, outputBar, &wg)
				}
			} else if len(c.DumpOutFile) > 0 {
				// start file write
				outputBar.Prefix("Write")
				wg.Add(1)
				go migrator.NewFileDumpWorker(outputBar, &wg)
			}

			wg.Wait()

			if showBar {

				outputBar.Finish()
				// close pool
				pool.Stop()

			}
		}

	}

	log.Info("data migration finished.")
}

func (c *Migrator) recoveryIndexSettings(sourceIndexRefreshSettings map[string]interface{}) {
	//update replica and refresh_interval
	for name, interval := range sourceIndexRefreshSettings {
		tempIndexSettings := getEmptyIndexSettings()
		tempIndexSettings["settings"].(map[string]interface{})["index"].(map[string]interface{})["refresh_interval"] = interval
		//tempIndexSettings["settings"].(map[string]interface{})["index"].(map[string]interface{})["number_of_replicas"] = 1
		c.TargetESAPI.UpdateIndexSettings(name, tempIndexSettings)
		if c.Config.Refresh {
			c.TargetESAPI.Refresh(name)
		}
	}
}

func (c *Migrator) ClusterVersion(host string, auth *Auth, proxy string) (*ClusterVersion, []error) {

	url := fmt.Sprintf("%s", host)
	resp, body, errs := Get(url, auth, proxy)

	if resp != nil && resp.Body != nil {
		io.Copy(ioutil.Discard, resp.Body)
		defer resp.Body.Close()
	}

	if errs != nil {
		log.Error(errs)
		return nil, errs
	}

	log.Debug(body)

	version := &ClusterVersion{}
	err := json.Unmarshal([]byte(body), version)

	if err != nil {
		log.Error(body, errs)
		return nil, errs
	}
	return version, nil
}

func (c *Migrator) ClusterReady(api ESAPI) (*ClusterHealth, bool) {
	health := api.ClusterHealth()

	if !c.Config.WaitForGreen {
		return health, true
	}

	if health.Status == "red" {
		return health, false
	}

	if c.Config.WaitForGreen == false && health.Status == "yellow" {
		return health, true
	}

	if health.Status == "green" {
		return health, true
	}

	return health, false
}
