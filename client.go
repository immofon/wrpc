package wrpc

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"
)

type Client struct {
	httpClient *http.Client

	// read-only after init
	URL   string
	Token string
}

func NewClient(url, token string) *Client {
	return &Client{
		httpClient: http.DefaultClient,
		URL:        url,
		Token:      token,
	}
}

func (c *Client) Call(ctx context.Context, method string, args ...string) (Resp, error) {
	req, err := http.NewRequest("POST", c.URL, strings.NewReader(Req{
		Token:  c.Token,
		Method: method,
		Args:   args,
	}.Encode()))
	if err != nil {
		return Resp{}, err
	}

	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Resp{}, err
	}
	defer resp.Body.Close()

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Resp{}, err
	}

	data := strings.Split(string(raw), "\x1F")
	var status Status
	var rets []string
	if len(data) > 0 {
		status = Status(data[0])
	}
	if len(data) > 1 {
		rets = data[1:]
	}

	return Ret(status, rets...), nil
}
