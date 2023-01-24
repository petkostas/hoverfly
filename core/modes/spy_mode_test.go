package modes_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/SpectoLabs/hoverfly/core/errors"
	"github.com/SpectoLabs/hoverfly/core/models"
	"github.com/SpectoLabs/hoverfly/core/modes"
	. "github.com/onsi/gomega"
)

type hoverflySpyStub struct{}

// DoRequest - Stub implementation of modes.HoverflySpy interface
func (this hoverflySpyStub) DoRequest(request *http.Request) (*http.Response, error) {
	response := &http.Response{}
	if request.Host == "error.com" {
		return nil, fmt.Errorf("Could not reach error.com")
	}

	response.StatusCode = 200
	response.Body = ioutil.NopCloser(bytes.NewBufferString("test"))

	return response, nil
}

func (this hoverflySpyStub) GetResponse(requestDetails models.RequestDetails) (*models.ResponseDetails, *errors.HoverflyError) {
	if requestDetails.Destination == "positive-match.com" {
		return &models.ResponseDetails{
			Status: 200,
		}, nil
	} else {
		return nil, &errors.HoverflyError{
			Message: "matching-error",
		}
	}
}

func (this hoverflySpyStub) ApplyMiddleware(pair models.RequestResponsePair) (models.RequestResponsePair, error) {
	if pair.Request.Path == "middleware-error" {
		return pair, fmt.Errorf("middleware-error")
	}
	return pair, nil
}

func (hf hoverflySpyStub) ApplyPostHooks(pair models.RequestResponsePair) (models.RequestResponsePair, error) {
	if pair.Response.PostActionHooks != nil {
		for _, postActionHook := range pair.Response.PostActionHooks {
			hookMap, ok := postActionHook.(map[string]interface{})
			if !ok {
				fmt.Println("Unable to convert post action hook to map[string]interface{}")
				continue
			}

			postActionHookStruct := &models.PostActionHook{}
			jsonBytes, err := json.Marshal(hookMap)
			if err != nil {
				fmt.Println("Unable to convert post action hook to json")
				continue
			}

			err = json.Unmarshal(jsonBytes, postActionHookStruct)
			if err != nil {
				fmt.Println("Unable to unmarshal json into struct")
				continue
			}
			postActionHookStruct.Execute()
		}
	}
	return pair, nil
}

func Test_SpyMode_WhenGivenAMatchingRequestItReturnsTheCorrectResponse(t *testing.T) {
	RegisterTestingT(t)

	unit := &modes.SpyMode{
		Hoverfly: hoverflySpyStub{},
	}

	request := models.RequestDetails{
		Destination: "positive-match.com",
	}

	result, err := unit.Process(&http.Request{}, request)
	Expect(err).To(BeNil())

	Expect(result.Response.StatusCode).To(Equal(200))
}

func Test_SpyMode_WhenGivenANonMatchingRequestItWillMakeTheRequestAndReturnIt(t *testing.T) {
	RegisterTestingT(t)

	unit := &modes.SpyMode{
		Hoverfly: hoverflySpyStub{},
	}

	requestDetails := models.RequestDetails{
		Scheme:      "http",
		Destination: "negative-match.com",
	}

	request, err := http.NewRequest("GET", "http://positive-match.com", nil)
	Expect(err).To(BeNil())

	result, err := unit.Process(request, requestDetails)
	Expect(err).To(BeNil())

	Expect(result.Response.StatusCode).To(Equal(200))

	responseBody, err := ioutil.ReadAll(result.Response.Body)
	Expect(err).To(BeNil())

	Expect(string(responseBody)).To(Equal("test"))
}

func Test_SpyMode_WhenGivenAMatchingRequesAndMiddlewareFaislItReturnsAnError(t *testing.T) {
	RegisterTestingT(t)

	unit := &modes.SpyMode{
		Hoverfly: hoverflySpyStub{},
	}

	request := models.RequestDetails{
		Destination: "positive-match.com",
		Path:        "middleware-error",
	}

	result, err := unit.Process(&http.Request{}, request)
	Expect(err).ToNot(BeNil())

	Expect(result.Response.StatusCode).To(Equal(http.StatusBadGateway))

	responseBody, err := ioutil.ReadAll(result.Response.Body)
	Expect(err).To(BeNil())

	Expect(string(responseBody)).To(ContainSubstring("There was an error when executing middleware"))
	Expect(string(responseBody)).To(ContainSubstring("middleware-error"))
}

func Test_SpyMode_ShouldReturnErrorOnRemoteServiceCall(t *testing.T) {
	RegisterTestingT(t)

	unit := &modes.SpyMode{
		Hoverfly: hoverflySpyStub{},
	}

	requestDetails := models.RequestDetails{
		Scheme:      "http",
		Destination: "error.com",
	}

	request, err := http.NewRequest("GET", "http://error.com", nil)
	Expect(err).To(BeNil())

	result, err := unit.Process(request, requestDetails)
	Expect(err).ToNot(BeNil())

	Expect(result.Response.StatusCode).To(Equal(http.StatusBadGateway))

	responseBody, err := ioutil.ReadAll(result.Response.Body)
	Expect(err).To(BeNil())

	Expect(string(responseBody)).To(ContainSubstring("There was an error when forwarding the request to the intended destination"))
	Expect(string(responseBody)).To(ContainSubstring("Could not reach error.com"))

}
