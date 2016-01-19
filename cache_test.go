package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestSetKey(t *testing.T) {
	server, dbClient := testTools(201, `{'message': 'here'}`)
	defer server.Close()

	k := []byte("randomkeyhere")
	v := []byte("value")

	err := dbClient.cache.Set(k, v)
	expect(t, err, nil)

	value, err := dbClient.cache.Get(k)
	expect(t, err, nil)
	expect(t, string(value), string(v))
	dbClient.cache.DeleteBucket(dbClient.cache.requestsBucket)
}

func TestPayloadSetGet(t *testing.T) {
	server, dbClient := testTools(201, `{'message': 'here'}`)
	defer server.Close()

	key := []byte("keySetGetCache")
	resp := response{
		Status: 200,
		Body:   "body here",
	}

	payload := Payload{Response: resp}
	bts, err := json.Marshal(payload)
	expect(t, err, nil)

	err = dbClient.cache.Set(key, bts)
	expect(t, err, nil)

	var p Payload
	payloadBts, err := dbClient.cache.Get(key)
	err = json.Unmarshal(payloadBts, &p)
	expect(t, err, nil)
	expect(t, payload.Response.Body, p.Response.Body)

	dbClient.cache.DeleteBucket(dbClient.cache.requestsBucket)
}

func TestGetNonExistingBucket(t *testing.T) {
	server, dbClient := testTools(201, `{'message': 'here'}`)
	defer server.Close()

	dbClient.cache.requestsBucket = []byte("some_random_bucket")

	_, err := dbClient.cache.Get([]byte("whatever"))
	expect(t, err.Error(), "Bucket \"some_random_bucket\" not found!")
}

func TestDeleteBucket(t *testing.T) {
	server, dbClient := testTools(201, `{'message': 'here'}`)
	defer server.Close()

	k := []byte("randomkeyhere")
	v := []byte("value")
	// checking whether bucket is okay
	err := dbClient.cache.Set(k, v)
	expect(t, err, nil)

	value, err := dbClient.cache.Get(k)
	expect(t, err, nil)
	expect(t, string(value), string(v))

	// deleting bucket
	err = dbClient.cache.DeleteBucket(dbClient.cache.requestsBucket)
	expect(t, err, nil)

	// deleting it again
	err = dbClient.cache.DeleteBucket(dbClient.cache.requestsBucket)
	refute(t, err, nil)
}

func TestGetAllRequestNoBucket(t *testing.T) {
	server, dbClient := testTools(201, `{'message': 'here'}`)
	defer server.Close()

	dbClient.cache.requestsBucket = []byte("no_bucket_for_TestGetAllRequestNoBucket")
	_, err := dbClient.cache.GetAllRequests()
	// expecting nil since this would mean that records were wiped
	expect(t, err, nil)
}

func TestGetMultipleRecords(t *testing.T) {
	server, dbClient := testTools(201, `{'message': 'here'}`)
	defer server.Close()

	// inserting some payloads
	for i := 0; i < 5; i++ {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://example.com/q=%d", i), nil)
		expect(t, err, nil)
		dbClient.captureRequest(req)
	}

	// getting requests
	payloads, err := dbClient.cache.GetAllRequests()
	expect(t, err, nil)

	for _, payload := range payloads {
		expect(t, payload.Request.Method, "GET")
		expect(t, payload.Response.Status, 201)
	}

	dbClient.cache.DeleteBucket(dbClient.cache.requestsBucket)
}
