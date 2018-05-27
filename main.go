package main

import (
	"bytes"
	hm "crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codecommit"
	git "gopkg.in/libgit2/git2go.v26"
)

const (
	cloneStore = "/tmp/store"
)

var (
	region = os.Getenv("REGION")
	bucket = os.Getenv("S3_BUCKET")
	branch = os.Getenv("BRANCH")
)

func initSession() (*session.Session, error) {
	fmt.Println("init session")
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))

	return sess, nil
}

func makeCanonicalRequest(fullURL string, key string) (string, error) {
	u, _ := url.Parse(fullURL)
	t := time.Now()
	cr := new(bytes.Buffer)
	fmt.Fprintf(cr, "%s\n", "GIT")         // HTTPRequestMethod
	fmt.Fprintf(cr, "%s\n", u.Path)        // CanonicalURI
	fmt.Fprintf(cr, "%s\n", "")            // CanonicalQueryString
	fmt.Fprintf(cr, "host:%s\n\n", u.Host) // CanonicalHeaders
	fmt.Fprintf(cr, "%s\n", "host")        // SignedHeaders
	fmt.Fprintf(cr, "%s", "")              // HexEncode(Hash(Payload))

	fmt.Println("Canonical Request: " + cr.String())

	sts := new(bytes.Buffer)
	fmt.Fprint(sts, "AWS4-HMAC-SHA256\n")                                                   // Algorithm
	fmt.Fprintf(sts, "%s\n", t.Format("20060102T150405"))                                   // RequestDate
	fmt.Fprintf(sts, "%s/%s/%s/aws4_request\n", t.Format("20060102"), region, "codecommit") // CredentialScope
	fmt.Fprintf(sts, "%s", hash(cr.String()))

	fmt.Println("String to Sign: " + sts.String())

	dsk := hmac([]byte("AWS4"+key), []byte(t.Format("20060102")))
	dsk = hmac(dsk, []byte(region))
	dsk = hmac(dsk, []byte("codecommit"))
	dsk = hmac(dsk, []byte("aws4_request"))
	h := hmac(dsk, []byte(sts.String()))
	sig := fmt.Sprintf("%x", h) // HexEncode(HMAC(derived-signing-key, string-to-sign))

	fmt.Println("Signature: " + sig)

	// codecommmit smart http password to use with AWS_ACCESS
	output := fmt.Sprintf("%sZ%s", t.Format("20060102T150405"), sig)
	fmt.Println("Magic Password: " + output)
	return output, nil
}

func hash(in string) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s", in)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func hmac(key, data []byte) []byte {
	h := hm.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

type CloneInfo struct {
	URL       string
	AccessID  string
	AccessKey string
	Token     string
}

func getCloneInfo(repo string) (*CloneInfo, error) {

	r := repo

	sess, err := initSession()
	if err != nil {
		errors.New("session failed to initialize")
	}

	svc := codecommit.New(sess)

	result, err := svc.GetRepository(
		&codecommit.GetRepositoryInput{
			RepositoryName: aws.String(r)})
	if err != nil {
		errors.New("get codecommit repository input failed")
	}

	allCreds, _ := sess.Config.Credentials.Get()

	output := &CloneInfo{
		URL:       *result.RepositoryMetadata.CloneUrlHttp,
		AccessKey: allCreds.SecretAccessKey,
		Token:     allCreds.SessionToken,
		AccessID:  allCreds.AccessKeyID,
	}

	return output, nil
}

func credentialsCallback(url string, username string, allowedTypes git.CredType) (git.ErrorCode, *git.Cred) {
	sess, err := initSession()
	if err != nil {
		errors.New("credentials callback: no session initialized")
	}
	allCreds, err := sess.Config.Credentials.Get()
	if err != nil {
		errors.New("credentials callback: no credentials to get")
	}

	username = allCreds.AccessKeyID + "%" + allCreds.SessionToken

	password, err := makeCanonicalRequest(url, allCreds.SecretAccessKey)
	if err != nil {
		errors.New("credentials callback: no password in conincal request")
	}

	ret, cred := git.NewCredUserpassPlaintext(username, password)
	return git.ErrorCode(ret), &cred
}

func clone(url string, path string) *git.Repository {

	cloneOptions := &git.CloneOptions{}

	cloneOptions.FetchOptions = &git.FetchOptions{
		RemoteCallbacks: git.RemoteCallbacks{
			CredentialsCallback: credentialsCallback,
		},
	}
	repo, err := git.Clone(url, path, cloneOptions)
	if err != nil {
		errors.New("clone failed")
	}
	return repo
}

func HandleRequest(evt events.CodeCommitEvent) error {

	var repo string

	os.Remove("/tmp")

	for _, record := range evt.Records {
		if record.EventSourceARN != "" {
			repo = strings.Split(record.EventSourceARN, ":")[5]
		} else {
			errors.New("could not resolve repo name, source ARN is empty")
		}
	}

	cloneInfo, err := getCloneInfo(repo)
	if err != nil {
		errors.New("cloneInfo: failed getting clone info")
	}

	clone(cloneInfo.URL, cloneStore)

	return nil

}

func main() {
	lambda.Start(HandleRequest)
}
