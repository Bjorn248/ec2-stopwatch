package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/hashicorp/vault/api"
	"github.com/robfig/cron"
)

// Declare global cron scheduler
var cronScheduler *cron.Cron

func loadSchedulesFromRedis() error {
	redisConn := pool.Get()
	defer redisConn.Close()

	// Initialize Vault Client
	vconfig := api.DefaultConfig()
	vclient, verr := api.NewClient(vconfig)
	if verr != nil {
		fmt.Printf("Problem creating vault client, '%s'", verr)
		return errors.New("problem creating vault client")
	}

	// Check Vault Connection
	_, authError := vclient.Logical().Read("auth/token/lookup-self")
	if authError != nil {
		fmt.Printf("Something went wrong connecting to Vault! Error is '%s'", authError)
		return errors.New("something went wrong connecting to Vault")
	}

	// Get all keys containing the @ symbol (i.e. emails)
	RedisKeys, redisError := redis.Strings(redisConn.Do("KEYS", "*@*"))
	if redisError != nil {
		fmt.Printf("Error when looking up email: '%s'", redisError)
		return redisError
	}

	// var jsonData map[string]interface{}

	for _, email := range RedisKeys {
		UserFromRedis, redisError := redis.String(redisConn.Do("GET", email))
		if redisError != nil {
			fmt.Printf("Error when looking up email: '%s'", redisError)
			return errors.New("error looking up email in redis")
		}

		var jsonData map[string]interface{}

		if err := json.Unmarshal([]byte(UserFromRedis), &jsonData); err != nil {
			fmt.Println(err)
			return errors.New("error parsing json from redis data")
		}

		// If the user is not verified this will return nil
		if jsonData["aws"] == nil {
			continue
		}

		// TODO Get rid of all these type assertions
		// TODO maybe make this loop over all cloud providers instead of just aws
		awsSchedules := jsonData["aws"].(map[string]interface{})
		for awsAccountID, InstanceIDs := range awsSchedules {
			// Get AWS Secret Key for this account
			path := fmt.Sprintf("secret/%s/aws/%s", email, awsAccountID)

			secret, err := vclient.Logical().Read(path)
			if err != nil {
				fmt.Printf("error reading %s: %s", path, err)
				return errors.New("error reading vault secrets")
			}

			awsSecret := secret.Data["secret_key"]

			for InstanceID, _ := range InstanceIDs.(map[string]interface{}) {
				start := InstanceIDs.(map[string]interface{})[InstanceID].(map[string]interface{})["start"].(map[string]interface{})
				stop := InstanceIDs.(map[string]interface{})[InstanceID].(map[string]interface{})["stop"].(map[string]interface{})
				region := InstanceIDs.(map[string]interface{})[InstanceID].(map[string]interface{})["region"].(string)
				ddns_enabled := InstanceIDs.(map[string]interface{})[InstanceID].(map[string]interface{})["ddns"].(map[string]interface{})["enabled"].(string)
				var ddns_struct DDNS
				if InstanceIDs.(map[string]interface{})[InstanceID].(map[string]interface{})["ddns"].(map[string]interface{})["domain"] == nil && InstanceIDs.(map[string]interface{})[InstanceID].(map[string]interface{})["ddns"].(map[string]interface{})["hosted_zone_id"] == nil {
					domain := InstanceIDs.(map[string]interface{})[InstanceID].(map[string]interface{})["ddns"].(map[string]interface{})["domain"].(string)
					hosted_zone_id := InstanceIDs.(map[string]interface{})[InstanceID].(map[string]interface{})["ddns"].(map[string]interface{})["hosted_zone_id"].(string)
					ddns_struct = DDNS{Enabled: ddns_enabled, Domain: domain, HostedZoneID: hosted_zone_id}
				} else {
					ddns_struct = DDNS{Enabled: ddns_enabled}
				}

				cronStringStart := fmt.Sprintf("0 %s %s %s %s %s", start["minute"], start["hour"], start["day_of_month"],
					start["month"], start["day_of_week"])
				cronStringEnd := fmt.Sprintf("0 %s %s %s %s %s", stop["minute"], stop["hour"], stop["day_of_month"],
					stop["month"], stop["day_of_week"])

				cronScheduler.AddFunc(cronStringStart, func() {
					startInstance(awsAccountID, awsSecret.(string), InstanceID, region, ddns_struct)
				})
				cronScheduler.AddFunc(cronStringEnd, func() {
					stopInstance(awsAccountID, awsSecret.(string), InstanceID, region)
				})
			}
		}
	}

	// Dividing by 2 here because there are two entries for every instance (one to start, one to stop)
	fmt.Printf("%d Schedule(s) loaded from Redis\n", len(cronScheduler.Entries())/2)

	return nil
}
