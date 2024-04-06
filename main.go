package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"html/template"
	"log"
	"os/exec"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	utils "github.com/ruts48code/utils4ruts"
)

type (
	RepoBuild struct {
		Repo    string   `json:"repo" yaml:"repo" hcl:"repo,label"`
		Branch  string   `json:"branch" yaml:"branch" hcl:"branch,label"`
		Message string   `json:"message" yaml:"message" hcl:"message,label"`
		Secret  string   `json:"secret" yaml:"secret" hcl:"secret"`
		Script  []string `json:"script" yaml:"script" hcl:"script"`
	}
	Conf struct {
		Listen string      `json:"listen" yaml:"listen" hcl:"listen"`
		Repo   []RepoBuild `json:"repo" yaml:"repo" hcl:"repo,block"`
	}
)

var (
	conf Conf
)

func main() {
	logs.AddLog("Start gitcicd system.......\n\n")
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

func readConfig() {
	if utils.FileExist("/etc/cicd.hcl") {
		utils.ProcessConfigHCL("/etc/cicd.hcl", &conf)
		log.Printf("Load /etc/cicd.hcl sucessfully\n")
	} else if utils.FileExist("/etc/cicd.yml") {
		utils.ProcessConfig("/etc/cicd.yml", &conf)
		log.Printf("Load /etc/cicd.yml sucessfully\n")
	} else {
		log.Printf("Error: cannot load configurationfile\n")
		return
	}
}

func cicd(c *fiber.Ctx) error {
	readConfig()

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
		logs.AddLog("Github")
		if len(header["X-Hub-Signature-256"]) > 0 {
			secret = header["X-Hub-Signature-256"][0]
		}
		ref, _ = data["ref"].(string)
		repo, _ = data["repository"].(map[string]any)["html_url"].(string)
		message, _ = data["head_commit"].(map[string]any)["message"].(string)

	} else {
		gittype = "gitlab"
		logs.AddLog("Gitlab")
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
	for i := range conf.Repo {
		if repo == conf.Repo[i].Repo && message == conf.Repo[i].Message && ref == "refs/heads/"+conf.Repo[i].Branch {
			logs.AddLog("activated!")
			if !checksecret(body, conf.Repo[i].Secret, secret, gittype) {
				logs.AddLog("secret error!!!")
				return nil
			}
			runx = true
			for i2 := range conf.Repo[i].Script {
				logs.AddLog("exec %s", conf.Repo[i].Script[i2])
				_, err := runCom(conf.Repo[i].Script[i2])
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
