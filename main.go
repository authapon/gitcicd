package main

import (
	"encoding/json"
	"log"
	"os"
	"os/exec"

	"github.com/gofiber/fiber/v2"
	"gopkg.in/yaml.v3"
)

type (
	ConditionBuild struct {
		Typex   string   `json:"type" yaml:"type"`
		Owner   string   `json:"owner" yaml:"owner"`
		Name    string   `json:"name" yaml:"name"`
		Message string   `json:"message" yaml:"message"`
		Ref     string   `json:"ref" yaml:"ref"`
		Script  []string `json:"script" yaml:"script"`
	}
	Conf struct {
		Listen    string           `json:"listen" yaml:"listen"`
		Condition []ConditionBuild `json:"condition" yaml:"condition"`
	}
)

var conf Conf

func main() {
	processConfig()
	log.Printf("cicd: %v\n", conf)
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
	name := ""
	owner := ""
	ref := ""
	message := ""
	typex := ""

	log.Printf("cicd: start!!!\n")

	data := make(map[string]any)
	err := json.Unmarshal(c.Body(), &data)
	if err != nil {
		log.Printf("cicd: error -> %v\n", err)
		return err
	}
	owner, ok := data["user_username"].(string)
	if !ok {
		typex = "github"
		ref = data["ref"].(string)
		name = data["repository"].(map[string]any)["name"].(string)
		owner = data["repository"].(map[string]any)["owner"].(map[string]any)["name"].(string)
		message = data["head_commit"].(map[string]any)["message"].(string)

	} else {
		typex = "gitlab"
		ref = data["ref"].(string)
		name = data["repository"].(map[string]any)["name"].(string)
		lx := len(data["commits"].([]any))
		if lx > 0 {
			message = data["commits"].([]any)[lx-1].(map[string]any)["title"].(string)
		}
	}

	log.Printf("cicd: process type:%s - owner:%s - name:%s - ref:%s - message:%s\n", typex, owner, name, ref, message)
	for i := range conf.Condition {
		log.Printf("cicd: test type:%s - owner:%s - name:%s - ref:%s - message:%s\n", conf.Condition[i].Typex, conf.Condition[i].Owner, conf.Condition[i].Name, conf.Condition[i].Ref, conf.Condition[i].Message)
		if owner == conf.Condition[i].Owner && name == conf.Condition[i].Name && message == conf.Condition[i].Message && ref == conf.Condition[i].Ref && typex == conf.Condition[i].Typex {
			log.Printf("cicd: activated!\n")
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
	log.Printf("cicd: finished\n")

	return c.SendString("ok")
}

func logcicd(c *fiber.Ctx) error {
	output, err := runCom("sudo", "systemctl", "status", "gitcicd", "-n", "100")
	if err != nil {
		log.Printf("cicd: error -> %v \n", err)
		return err
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
