package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/gofiber/fiber/v2"
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

var conf Conf

func main() {
	log.Printf("cicd: Start gitcicd system.......\n\n\n")
	processConfig()
	app := fiber.New(fiber.Config{
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

	log.Printf("cicd: start!!!\n")

	data := make(map[string]any)
	body := c.Body()
	header := c.GetReqHeaders()
	err := json.Unmarshal(body, &data)
	if err != nil {
		log.Printf("cicd: error -> %v\n", err)
		return err
	}
	repo, ok := data["repository"].(map[string]any)["homepage"].(string)
	if !ok {
		gittype = "github"
		if len(header["X-Hub-Signature-256"]) > 0 {
			secret = header["X-Hub-Signature-256"][0]
		}
		ref = data["ref"].(string)
		repo = data["repository"].(map[string]any)["html_url"].(string)
		message = data["head_commit"].(map[string]any)["message"].(string)

	} else {
		gittype = "gitlab"
		if len(header["X-Gitlab-Token"]) > 0 {
			secret = header["X-Gitlab-Token"][0]
		}
		ref = data["ref"].(string)
		lx := len(data["commits"].([]any))
		if lx > 0 {
			message = data["commits"].([]any)[lx-1].(map[string]any)["title"].(string)
		}
	}
	runx := false
	log.Printf("cicd: process repo:%s - ref:%s\n", repo, ref)
	for i := range conf.Condition {
		if repo == conf.Condition[i].Repo && message == conf.Condition[i].Message && ref == conf.Condition[i].Ref {
			log.Printf("cicd: activated!\n")
			if !checksecret(body, conf.Condition[i].Secret, secret, gittype) {
				log.Printf("cicd: secret error!!!\n")
				return nil
			}
			runx = true
			for i2 := range conf.Condition[i].Script {
				log.Printf("cicd: exec %s\n", conf.Condition[i].Script[i2])
				_, err := runCom(conf.Condition[i].Script[i2])
				if err != nil {
					log.Printf("cicd: error -> %v\n", err)
					return err
				}
			}
		}
	}
	if runx {
		log.Printf("cicd: operation finished\n")
	} else {
		log.Printf("cicd: no operation!!")
	}

	return c.SendString("ok")
}

func logcicd(c *fiber.Ctx) error {
	command := "sudo systemctl status gitcicd -n 100 | grep 'cicd:'"
	output, err := runCom("bash", "-c", command)
	if err != nil {
		log.Printf("cicd: error -> %v \n", err)
		return err
	}
	ostring := strings.Split(string(output), "\n")
	if len(ostring) > 2 {
		nstring := ostring[2:]
		output = []byte(strings.Join(nstring, "\n"))
	} else {
		return c.SendString("")
	}
	return c.SendString(string(output))
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
