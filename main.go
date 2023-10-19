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
		Owner   string   `json:"owner"`
		Name    string   `json:"name"`
		Message string   `json:"message"`
		Script  []string `json:"script"`
	}
	Conf struct {
		Listen    string           `json:"listen"`
		Condition []ConditionBuild `json:"condition"`
	}
)

var conf Conf

func main() {
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
	output := ""
	data := make(map[string]any)
	err := json.Unmarshal(c.Body(), &data)
	if err != nil {
		log.Printf("cicd: error -> %v\n", err)
		return err
	}
	message := data["head_commit"].(map[string]any)["message"].(string)
	name := data["repository"].(map[string]any)["name"].(string)
	owner := data["repository"].(map[string]any)["owner"].(map[string]any)["name"].(string)
	log.Printf("cicd: process owner:%s - name:%s - message:%s\n", owner, name, message)
	for i := range conf.Condition {
		if owner == conf.Condition[i].Owner && name == conf.Condition[i].Name && message == conf.Condition[i].Message {
			for i2 := range conf.Condition[i].Script {
				odata, err := runCom(conf.Condition[i].Script[i2])
				if err != nil {
					log.Printf("cicd: error -> %v\n", err)
					return err
				}
				log.Printf("cicd: " + string(odata))
				output += string(odata) + "\n"
			}
		}
	}

	return c.SendString(output)
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
