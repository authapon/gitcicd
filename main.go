package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"log"
	"os"
	"os/exec"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"gopkg.in/yaml.v3"
)

type (
	ConditionBuild struct {
		Repo    string   `json:"repo" yaml:"repo"`
		Message string   `json:"message" yaml:"message"`
		Ref     string   `json:"ref" yaml:"ref"`
		Secret  string   `json:"secret" yaml:"secret"`
		Script  []string `json:"script" yaml:"script"`
	}
	Conf struct {
		Listen    string           `json:"listen" yaml:"listen"`
		Condition []ConditionBuild `json:"condition" yaml:"condition"`
	}
)

var (
	conf Conf
)

func main() {
	logs.AddLog("Start gitcicd system.......\n\n")
	processConfig()
	engine := html.New("./template", ".html")
	engine.AddFunc(
		// add unescape function
		"unescape", func(s string) template.HTML {
			return template.HTML(s)
		},
	)
	app := fiber.New(fiber.Config{
		Views:          engine,
		ProxyHeader:    fiber.HeaderXForwardedFor,
		ReadBufferSize: 20480,
		BodyLimit:      104857600,
	})

	app.Get("/", cicd)
	app.Post("/", cicd)
	app.Get("/log", logcicd)
	log.Fatal(app.Listen(conf.Listen))
}

func cicd(c *fiber.Ctx) error {
	repo := ""
	ref := ""
	message := ""
	gittype := ""
	secret := ""

	logs.AddLog("start!!!")

	data := make(map[string]any)
	body := c.Body()
	header := c.GetReqHeaders()
	err := json.Unmarshal(body, &data)
	if err != nil {
		logs.AddLog("error -> %v", err)
		return err
	}
	repo, ok := data["repository"].(map[string]any)["homepage"].(string)
	if !ok {
		gittype = "github"
		if len(header["X-Hub-Signature-256"]) > 0 {
			secret = header["X-Hub-Signature-256"][0]
		}
		ref, _ = data["ref"].(string)
		repo, _ = data["repository"].(map[string]any)["html_url"].(string)
		message, _ = data["head_commit"].(map[string]any)["message"].(string)

	} else {
		gittype = "gitlab"
		if len(header["X-Gitlab-Token"]) > 0 {
			secret = header["X-Gitlab-Token"][0]
		}
		ref, _ = data["ref"].(string)
		lx := len(data["commits"].([]any))
		if lx > 0 {
			message, _ = data["commits"].([]any)[lx-1].(map[string]any)["title"].(string)
		}
	}
	runx := false
	logs.AddLog("process repo:%s - ref:%s", repo, ref)
	for i := range conf.Condition {
		if repo == conf.Condition[i].Repo && message == conf.Condition[i].Message && ref == conf.Condition[i].Ref {
			logs.AddLog("activated!")
			if !checksecret(body, conf.Condition[i].Secret, secret, gittype) {
				logs.AddLog("secret error!!!")
				return nil
			}
			runx = true
			for i2 := range conf.Condition[i].Script {
				logs.AddLog("exec %s", conf.Condition[i].Script[i2])
				_, err := runCom(conf.Condition[i].Script[i2])
				if err != nil {
					logs.AddLog("error -> %v", err)
					return err
				}
			}
		}
	}
	if runx {
		logs.AddLog("operation finished")
	} else {
		logs.AddLog("no operation!!")
	}

	return c.SendString("ok")
}

func logcicd(c *fiber.Ctx) error {
	return c.Render("log", fiber.Map{
		"Logs": logs.GetLogsHTML(),
	})
}

func processConfig() {
	confdata := readFile("/etc/cicd.yml")
	yaml.Unmarshal(confdata, &conf)
}

func readFile(f string) []byte {
	content, err := os.ReadFile(f)
	if err != nil {
		return []byte("")
	}
	return content
}

func runCom(c string, arg ...string) ([]byte, error) {
	cmd := exec.Command(c, arg...)
	data, err := cmd.Output()
	return data, err
}

func checksecret(body []byte, confsecret string, headsecret string, gittype string) bool {
	if confsecret == "" {
		return false
	}
	switch gittype {
	case "github":
		key := hmac.New(sha256.New, []byte(confsecret))
		key.Write(body)
		computedSignature := "sha256=" + hex.EncodeToString(key.Sum(nil))
		return computedSignature == headsecret
	case "gitlab":
		return confsecret == headsecret
	default:
		return false
	}
}
