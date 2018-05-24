package jtl

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	. "github.com/Comcast/eel/util"
)

const (
	OAUTH_VERSION              = "1.0"
	SIGNATURE_METHOD_HMAC_SHA1 = "HMAC-SHA1"
	SIGNATURE_METHOD_RSA_SHA1  = "RSA-SHA1"

	CALLBACK_PARAM         = "oauth_callback"
	CONSUMER_KEY_PARAM     = "oauth_consumer_key"
	NONCE_PARAM            = "oauth_nonce"
	SESSION_HANDLE_PARAM   = "oauth_session_handle"
	SIGNATURE_METHOD_PARAM = "oauth_signature_method"
	SIGNATURE_PARAM        = "oauth_signature"
	TIMESTAMP_PARAM        = "oauth_timestamp"
	TOKEN_PARAM            = "oauth_token"
	TOKEN_SECRET_PARAM     = "oauth_token_secret"
	VERIFIER_PARAM         = "oauth_verifier"
	VERSION_PARAM          = "oauth_version"
)

type OrderedParams struct {
	allParams   map[string]string
	keyOrdering []string
}

func NewOrderedParams() *OrderedParams {
	return &OrderedParams{
		allParams:   make(map[string]string),
		keyOrdering: make([]string, 0),
	}
}

func (o *OrderedParams) Get(key string) string {
	return o.allParams[key]
}

func (o *OrderedParams) Keys() []string {
	sort.Sort(o)
	return o.keyOrdering
}

func (o *OrderedParams) Add(key, value string) {
	o.AddUnescaped(key, escape(value))
}

func (o *OrderedParams) AddUnescaped(key, value string) {
	o.allParams[key] = value
	o.keyOrdering = append(o.keyOrdering, key)
}

func (o *OrderedParams) Len() int {
	return len(o.keyOrdering)
}

func (o *OrderedParams) Less(i int, j int) bool {
	return o.keyOrdering[i] < o.keyOrdering[j]
}

func (o *OrderedParams) Swap(i int, j int) {
	o.keyOrdering[i], o.keyOrdering[j] = o.keyOrdering[j], o.keyOrdering[i]
}

func (o *OrderedParams) Clone() *OrderedParams {
	clone := NewOrderedParams()
	for _, key := range o.Keys() {
		clone.AddUnescaped(key, o.Get(key))
	}
	return clone
}

type SHA1Signer struct {
	consumerSecret string
	debug          bool
}

func (s *SHA1Signer) Debug(enabled bool) {
	s.debug = enabled
}

func (s *SHA1Signer) Sign(message string, tokenSecret string) (string, error) {
	key := escape(s.consumerSecret) + "&" + escape(tokenSecret)
	if s.debug {
		fmt.Println("Signing:", message)
		fmt.Println("Key:", key)
	}
	hashfun := hmac.New(sha1.New, []byte(key))
	hashfun.Write([]byte(message))
	rawSignature := hashfun.Sum(nil)
	base64signature := base64.StdEncoding.EncodeToString(rawSignature)
	if s.debug {
		fmt.Println("Base64 signature:", base64signature)
	}
	return base64signature, nil
}

func (s *SHA1Signer) SignatureMethod() string {
	return SIGNATURE_METHOD_HMAC_SHA1
}

type OAuthConsumer struct {
	serviceProvider string
}

func NewOAuthConsumer(serviceProvider string) *OAuthConsumer {
	return &OAuthConsumer{
		serviceProvider: serviceProvider,
	}
}

func (c *OAuthConsumer) BaseParams(consumerKey string, additionalParams map[string]string) *OrderedParams {
	params := NewOrderedParams()
	params.Add(VERSION_PARAM, OAUTH_VERSION)
	params.Add(SIGNATURE_METHOD_PARAM, SIGNATURE_METHOD_HMAC_SHA1)
	params.Add(TIMESTAMP_PARAM, strconv.FormatInt(time.Now().Unix(), 10))
	params.Add(NONCE_PARAM, strconv.FormatInt(rand.New(rand.NewSource(time.Now().UnixNano())).Int63(), 10))
	params.Add(CONSUMER_KEY_PARAM, consumerKey)
	for key, value := range additionalParams {
		params.Add(key, value)
	}
	return params
}

func (c *OAuthConsumer) RequestString(method string, url string, params *OrderedParams) string {
	result := method + "&" + escape(url)
	for pos, key := range params.Keys() {
		if pos == 0 {
			result += "&"
		} else {
			result += escape("&")
		}
		result += escape(fmt.Sprintf("%s=%s", key, params.Get(key)))
	}
	return result
}

func (c *OAuthConsumer) LoadConfig(filename string, conf interface{}) error {
	if _, err := os.Stat(filename); nil != err {
		return err
	}

	bs, err := ioutil.ReadFile(filename)
	if nil != err {
		return err
	}

	if err = json.Unmarshal(bs, conf); nil != err {
		return err
	}
	return nil
}

func (c *OAuthConsumer) GetOAuth1Header(ctx Context, method string, reqURL string) (string, error) {

	//Read consumer key/secret from config file
	var keyConfig map[string](map[string]string)
	err := c.LoadConfig("config-eel/oauth_curl_key.json", &keyConfig)
	if err != nil {
		ctx.Log().Error("op", "getConfigFromFile", "error", err)
		return "", err
	}

	providerPair, ok := keyConfig[c.serviceProvider]
	if !ok {
		ctx.Log().Error("op", "getConfigFromFile", "error", "cannotGetProviderPair")
		return "", fmt.Errorf("No OAuth Provider Key/Secret Pair Found")
	}

	key, ok := providerPair["key"]
	if !ok {
		ctx.Log().Error("op", "getConfigFromFile", "error", "cannotGetConsumerKey")
		return "", fmt.Errorf("No OAuth Consumer Key Found")
	}

	secret, ok := providerPair["secret"]
	if !ok {
		ctx.Log().Error("op", "getConfigFromFile", "error", "cannotGetConsumerSecret")
		return "", fmt.Errorf("No OAuth Consumer Secret Found")
	}

	//Prepare oauth params before signature
	oauthParams := c.BaseParams(key, make(map[string]string))

	ss := strings.Split(reqURL, "?")

	//Add query params to prepare for signature
	allParams := oauthParams.Clone()
	if 1 < len(ss) {
		vals, err := url.ParseQuery(ss[1])
		if nil != err {
			return "", err
		}

		for k, v := range vals {
			allParams.Add(k, v[0])
		}
	}

	//Prepare signature string
	baseString := c.RequestString(method, ss[0], allParams)

	signer := &SHA1Signer{
		consumerSecret: secret,
	}

	//signature
	signature, err := signer.Sign(baseString, "")
	if err != nil {
		return "", err
	}

	//Oauth params with signature
	oauthParams.Add(SIGNATURE_PARAM, signature)

	//concat oauth params as OAuth header
	oauthHdr := "OAuth "
	for pos, key := range oauthParams.Keys() {
		if pos > 0 {
			oauthHdr += ","
		}
		oauthHdr += key + "=\"" + oauthParams.Get(key) + "\""
	}
	ctx.Log().Debug("op", "GetOAuth1Header", "OAuth1Header", oauthHdr)
	return oauthHdr, nil
}

func escape(s string) string {
	t := make([]byte, 0, 3*len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isEscapable(c) {
			t = append(t, '%')
			t = append(t, "0123456789ABCDEF"[c>>4])
			t = append(t, "0123456789ABCDEF"[c&15])
		} else {
			t = append(t, s[i])
		}
	}
	return string(t)
}

func isEscapable(b byte) bool {
	return !('A' <= b && b <= 'Z' || 'a' <= b && b <= 'z' || '0' <= b && b <= '9' || b == '-' || b == '.' || b == '_' || b == '~')
}
