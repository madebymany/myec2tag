package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/ec2"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const infoUrlBase = `http://169.254.169.254`
const metadataUrlBase = infoUrlBase + `/latest/meta-data`

var splitCommas bool

type InstanceIdentity struct {
	InstanceId string
	Region     string
}

func init() {
	flag.BoolVar(&splitCommas, "s", false, "splits multiple values by comma")
}

func getInfo(url string) (body []byte) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}
	if resp.StatusCode != 200 {
		log.Fatalf("%s returned %s\n", url, resp.Status)
	}
	defer resp.Body.Close()

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	return
}

func getInstanceIdentity() (id InstanceIdentity) {
	body := getInfo(infoUrlBase + `/latest/dynamic/instance-identity/document`)
	err := json.Unmarshal(body, &id)
	if err != nil {
		log.Fatalln(err)
	}
	return
}

func tagValue(i ec2.Instance, tagKey string) string {
	for _, tag := range i.Tags {
		if tag.Key == tagKey {
			return tag.Value
		}
	}
	return ""
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	var splitCommasRegexp *regexp.Regexp
	if splitCommas {
		splitCommasRegexp = regexp.MustCompile(`[^,\s]+`)
	}

	id := getInstanceIdentity()
	var region aws.Region
	var ok bool
	if region, ok = aws.Regions[id.Region]; !ok {
		log.Fatalf("Instance is in unknown region: %s\n", id.Region)
	}

	auth, err := aws.GetAuth("", "", "", time.Time{})
	if err != nil {
		log.Fatalln(err)
	}
	e := ec2.New(auth, region)

	instancesResp, err := e.Instances([]string{id.InstanceId}, nil)
	if err != nil {
		log.Fatalln(err)
	}

	instances := make([]ec2.Instance, 0, 1)
	for _, res := range instancesResp.Reservations {
		for _, inst := range res.Instances {
			instances = append(instances, inst)
		}
	}

	if len(instances) == 0 {
		log.Fatalln("Instance not found in API")
	}
	if len(instances) > 1 {
		log.Fatalln("More than one instance returned!")
	}

	instance := instances[0]

	for _, tagKey := range flag.Args() {
		val := tagValue(instance, tagKey)
		if splitCommas {
			vals := splitCommasRegexp.FindAllString(val, -1)
			val = strings.Join(vals, " ")
		}
		fmt.Println(val)
	}
}
