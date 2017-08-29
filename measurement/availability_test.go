package measurement_test

import (
	"fmt"
	"net/http"
	"sync"

	. "github.com/cloudfoundry/uptimer/measurement"
	"github.com/cloudfoundry/uptimer/measurement/measurementfakes"

	"bytes"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Availability", func() {
	var (
		url              string
		fakeRoundTripper *FakeRoundTripper
		successResponse  *http.Response
		failResponse     *http.Response
		client           *http.Client
		fakeResultSet    *measurementfakes.FakeResultSet

		am BaseMeasurement
	)

	BeforeEach(func() {
		url = "https://example.com/foo"
		fakeRoundTripper = &FakeRoundTripper{}
		successResponse = &http.Response{
			StatusCode: 200,
			Body:       ioutil.NopCloser(bytes.NewBufferString("")),
		}
		failResponse = &http.Response{
			StatusCode: 400,
			Body:       ioutil.NopCloser(bytes.NewBufferString("")),
		}
		fakeRoundTripper.RoundTripReturns(successResponse, nil)
		client = &http.Client{
			Transport: fakeRoundTripper,
		}
		fakeResultSet = &measurementfakes.FakeResultSet{}

		am = NewAvailability(url, client)
	})

	Describe("Name", func() {
		It("returns the name", func() {
			Expect(am.Name()).To(Equal("HTTP availability"))
		})
	})

	Describe("PerformMeasurement", func() {
		It("makes a get request to the url", func() {
			am.PerformMeasurement()

			req := fakeRoundTripper.RoundTripArgsForCall(0)
			Expect(req.Method).To(Equal(http.MethodGet))
			Expect(req.URL.String()).To(Equal("https://example.com/foo"))
		})

		It("records 200 results as success", func() {
			_, _, _, res := am.PerformMeasurement()

			Expect(res).To(BeTrue())
		})

		It("records the non-200 results as failed", func() {
			fakeRoundTripper.RoundTripReturns(failResponse, nil)

			_, _, _, res := am.PerformMeasurement()

			Expect(res).To(BeFalse())
		})

		It("records the error results as failed", func() {
			fakeRoundTripper.RoundTripReturns(nil, fmt.Errorf("error"))

			_, _, _, res := am.PerformMeasurement()

			Expect(res).To(BeFalse())
		})

		It("returns error output when there is a non-200 response", func() {
			fakeRoundTripper.RoundTripReturns(failResponse, nil)

			msg, _, _, _ := am.PerformMeasurement()

			Expect(msg).To(Equal("response had status 400"))
		})

		It("returns error output when there is an error", func() {
			fakeRoundTripper.RoundTripReturns(nil, fmt.Errorf("error"))

			msg, _, _, _ := am.PerformMeasurement()

			Expect(msg).To(Equal("Get https://example.com/foo: error"))
		})

		It("closes the body of the response when there is a 200 response", func() {
			fakeRC := &fakeReadCloser{}
			fakeRoundTripper.RoundTripReturns(
				&http.Response{
					StatusCode: 200,
					Body:       fakeRC,
				},
				nil,
			)

			am.PerformMeasurement()

			Expect(fakeRC.Closed).To(BeTrue())
		})

		It("closes the body of the response when there is a non-200 response", func() {
			fakeRC := &fakeReadCloser{}
			fakeRoundTripper.RoundTripReturns(
				&http.Response{
					StatusCode: 400,
					Body:       fakeRC,
				},
				nil,
			)

			am.PerformMeasurement()

			Expect(fakeRC.Closed).To(BeTrue())
		})

		It("does not close the body of the response when there is an error", func() {
			fakeRoundTripper.RoundTripReturns(
				nil,
				fmt.Errorf("foobar"),
			)

			Expect(func() { am.PerformMeasurement() }).NotTo(Panic())
		})
	})
})

type fakeReadCloser struct {
	Closed bool
}

func (rc *fakeReadCloser) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (rc *fakeReadCloser) Close() error {
	rc.Closed = true

	return nil
}

type FakeRoundTripper struct {
	RoundTripStub        func(*http.Request) (*http.Response, error)
	roundTripMutex       sync.RWMutex
	roundTripArgsForCall []struct {
		arg1 *http.Request
	}
	roundTripReturns struct {
		result1 *http.Response
		result2 error
	}
}

func (fake *FakeRoundTripper) RoundTrip(arg1 *http.Request) (*http.Response, error) {
	fake.roundTripMutex.Lock()
	fake.roundTripArgsForCall = append(fake.roundTripArgsForCall, struct {
		arg1 *http.Request
	}{arg1})
	fake.roundTripMutex.Unlock()
	if fake.RoundTripStub != nil {
		return fake.RoundTripStub(arg1)
	} else {
		return fake.roundTripReturns.result1, fake.roundTripReturns.result2
	}
}

func (fake *FakeRoundTripper) RoundTripCallCount() int {
	fake.roundTripMutex.RLock()
	defer fake.roundTripMutex.RUnlock()
	return len(fake.roundTripArgsForCall)
}

func (fake *FakeRoundTripper) RoundTripArgsForCall(i int) *http.Request {
	fake.roundTripMutex.RLock()
	defer fake.roundTripMutex.RUnlock()
	return fake.roundTripArgsForCall[i].arg1
}

func (fake *FakeRoundTripper) RoundTripReturns(result1 *http.Response, result2 error) {
	fake.RoundTripStub = nil
	fake.roundTripReturns = struct {
		result1 *http.Response
		result2 error
	}{result1, result2}
}
