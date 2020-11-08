package guac

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	logger "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	// Create a new Setting
	setting = NewSetting()
	levels  = []string{"panic", "fatal", "error", "warn", "info", "debug", "trace"}
)

// Setting for guca
type Setting struct {
	Guacd struct {
		Address  string
		Override bool
	}
	Log struct {
		Level string
	}
	Server struct {
		Api struct {
			Ids    bool
			List   bool
			Read   bool
			Delete bool
			Update bool
		}
		Auth struct {
			Basic struct {
				Username string
				Password string
				Realm    string
			}
			Header struct {
				Name   string
				Values []string
			}
		}
		Static struct {
			Path string
		}
	}
	// Conns []Conn `yaml:"conns"`
	Conns []map[string]string `yaml:"conns"`
}

// GetConfPath find conf path
func GetConfPath() string {
	var confPath = os.Getenv("CONFIG_PATH")
	if confPath == "" {
		confPath = "config.yaml"
	}
	return confPath
}

// GetSetting get single setting obj
func GetSetting() *Setting {
	return setting
}

// GetGuacd get single setting obj
func GetGuacd() string {
	var guacd = "127.0.0.1:4822"
	if os.Getenv("GUACD") != "" {
		guacd = os.Getenv("GUACD")
	} else if setting.Guacd.Address != "" {
		guacd = setting.Guacd.Address
	}
	return guacd
}

// GetLogLevel Level
func GetLogLevel() logrus.Level {
	var level = "ERROR"
	if os.Getenv("LOG_LEVEL") != "" {
		level = os.Getenv("LOG_LEVEL")
	} else if setting.Log.Level != "" {
		level = setting.Log.Level
	}
	index := indexOf(levels, strings.ToLower(level))
	if index == -1 {
		index = 3
	}
	return logrus.AllLevels[index]
}

// NewSetting  creates a new Setting from file
func NewSetting() *Setting {
	t := Setting{}
	// Open our Setting
	yamlFile, err := os.Open(GetConfPath())
	// if we os.Open returns an error then handle it
	if err != nil {
		logger.Error(err)
	} else {
		logger.Debug("config.yaml Loaded")
		// defer the closing of our jsonFile so that we can parse it later on
		defer yamlFile.Close()
		byteValue, _ := ioutil.ReadAll(yamlFile)
		err := yaml.Unmarshal([]byte(byteValue), &t)
		if err != nil {
			logger.Error(err)
		}
		logger.Debugf("yaml:\n%v\n\n", t)
	}
	return &t
}

// GetIds get all ids in list
func GetIds() []string {
	ids := []string{}
	for i := 0; i < len(setting.Conns); i++ {
		var conn = setting.Conns[i]
		if conn["id"] == "" {
			conn["id"] = conn["scheme"] + "-" + conn["hostname"] + "-" + conn["port"] + "-" + conn["username"]
		}
		ids = append(ids, conn["id"])
	}
	return ids
}
func (s *Setting) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	method := r.Method
	var query url.Values = r.URL.Query()
	if method == "GET" {
		var id = query.Get("id")
		if setting.Server.Api.Ids && query.Get("ids") == "all" {
			render(w, r, GetIds())
		} else if setting.Server.Api.Read && id != "" {
			render(w, r, GetConn(id))
		} else if setting.Server.Api.List {
			render(w, r, GetSetting().Conns)
		}
	} else if setting.Server.Api.Delete && method == "DELETE" {
		var id = query.Get("id")
		if id != "" {
			w.WriteHeader(404)
		} else {
			RemoveConn(id)
			w.WriteHeader(200)
		}
	} else if setting.Server.Api.Update && method == "POST" {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			logrus.Error("Failed to read body ", err)
			w.WriteHeader(400)
		}
		_ = r.Body.Close()
		var item = map[string]string{}
		err = json.Unmarshal(data, &item)
		if err != nil {
			logger.Error(err)
			w.WriteHeader(400)
		} else {
			AddConn(item)
			render(w, r, item)
		}
	}
}
func render(w http.ResponseWriter, r *http.Request, data interface{}) {
	if data == nil {
		w.WriteHeader(404)
	} else {
		jsonString, _ := json.Marshal(data)
		w.Write([]byte(jsonString))
	}
}

// GetConn returns a connection by id
func GetConn(id string) map[string]string {
	for i := 0; i < len(setting.Conns); i++ {
		var conn = setting.Conns[i]
		if conn["id"] == id {
			return conn
		}
	}
	return nil
}

// RemoveConn Remove Conn form yaml
func RemoveConn(id string) {
	for i := 0; i < len(setting.Conns); i++ {
		var conn = setting.Conns[i]
		if conn["id"] == id {
			setting.Conns[i] = setting.Conns[len(setting.Conns)-1] // Copy last element to index i.
			setting.Conns[len(setting.Conns)-1] = nil              // Erase last element (write zero value).
			setting.Conns = setting.Conns[:len(setting.Conns)-1]
			break
		}
	}
	return
}

// AddConn inserts a new connection
func AddConn(conn map[string]string) {
	if conn["id"] == "" {
		conn["id"] = conn["scheme"] + "-" + conn["hostname"] + "-" + conn["port"] + "-" + conn["username"]
	}
	savedConn := GetConn(conn["id"])
	if savedConn == nil {
		setting.Conns = append(setting.Conns, conn)
	} else {
		for k, v := range conn {
			savedConn[k] = v
		}
	}
	SaveSetting()
}

// SaveSetting save setting to file
func SaveSetting() {
	var d, err = yaml.Marshal(&setting)
	if err != nil {
		logger.Error(err)
	} else {
		err = ioutil.WriteFile(GetConfPath(), d, 0644)
		if err != nil {
			logger.Error(err)
		}
	}
}
