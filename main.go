package main

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/mail"
	"os"
	"strings"

	"github.com/emersion/go-smtp"
	"github.com/google/go-github/v44/github"
	"github.com/namsral/flag"
	"golang.org/x/oauth2"
)

const (
	name      = "smtp2workflow"
	envPrefix = "SMTP2WORKFLOW"
	version   = "1.0.0"
)

var (
	fs           *flag.FlagSet
	domain       string
	code         string
	githubToken  string
	tlsCertPath  string
	tlsKeyPath   string
	healthcheck  bool
	printVersion bool
)

type Workflow struct {
	Owner            string
	Repo             string
	Ref              string
	WorkflowFileName string
}

func (w Workflow) String() string {
	return fmt.Sprintf("{%s/%s/blob/%s/.github/workflows/%s}", w.Owner, w.Repo, w.Ref, w.WorkflowFileName)
}

var ghClient *github.Client
var workflows = make(map[string]Workflow)

func main() {
	ctx := context.Background()

	fs = flag.NewFlagSetWithEnvPrefix(name, envPrefix, flag.ExitOnError)
	fs.StringVar(&domain, "domain", "localhost", "domain")
	fs.StringVar(&code, "code", "", "secret code")
	fs.StringVar(&githubToken, "github-token", "", "github personal access token")
	fs.StringVar(&tlsCertPath, "tls-cert", "", "TLS certificate path")
	fs.StringVar(&tlsKeyPath, "tls-key", "", "TLS key path")
	fs.BoolVar(&healthcheck, "healthcheck", false, "run healthcheck")
	fs.BoolVar(&printVersion, "version", false, "print version")
	fs.Parse(os.Args[1:])

	if printVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	ghClient = github.NewClient(tc)

	for _, s := range os.Environ() {
		kv := strings.SplitN(s, "=", 2)
		if strings.HasPrefix(kv[0], "SMTP2WORKFLOW_REPOSITORY_") {
			key := code + "+" + strings.ToLower(kv[0][25:]) + "@"
			value := kv[1]

			workflow, ok := workflows[key]
			if !ok {
				workflow = Workflow{}
			}

			nwo := strings.SplitN(value, "/", 2)
			workflow.Owner = nwo[0]
			workflow.Repo = nwo[1]

			ref, err := GetDefaultBranch(ctx, workflow.Owner, workflow.Repo)
			if workflow.Ref == "" && err == nil {
				workflow.Ref = *ref
			}

			workflows[key] = workflow
		}

		if strings.HasPrefix(kv[0], "SMTP2WORKFLOW_REF_") {
			key := code + "+" + strings.ToLower(kv[0][18:]) + "@"
			value := kv[1]

			workflow, ok := workflows[key]
			if !ok {
				workflow = Workflow{}
			}
			workflow.Ref = value
			workflows[key] = workflow
		}

		if strings.HasPrefix(kv[0], "SMTP2WORKFLOW_WORKFLOW_") {
			key := code + "+" + strings.ToLower(kv[0][23:]) + "@"
			value := kv[1]

			workflow, ok := workflows[key]
			if !ok {
				workflow = Workflow{}
			}
			workflow.WorkflowFileName = value
			workflows[key] = workflow
		}
	}

	if healthcheck {
		client, err := smtp.Dial("127.0.0.1:25")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		err = client.Hello("localhost")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		os.Exit(0)
	}

	s := smtp.NewServer(&Backend{
		Context: ctx,
	})

	if tlsCertPath != "" && tlsKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(tlsCertPath, tlsKeyPath)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		s.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
	}

	s.Domain = domain
	s.AllowInsecureAuth = true
	s.AuthDisabled = true
	s.EnableSMTPUTF8 = false

	go func() {
		if s.TLSConfig != nil {
			log.Printf("Listening on :465")
			s.Addr = "[::]:465"
			if err := s.ListenAndServeTLS(); err != nil {
				log.Fatal(err)
			}
		}
	}()

	log.Printf("Listening on :25")
	s.Addr = "[::]:25"
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

type Backend struct {
	Context context.Context
}

func (bkd *Backend) Login(state *smtp.ConnectionState, username, password string) (smtp.Session, error) {
	return &Session{Context: bkd.Context}, nil
}

func (bkd *Backend) AnonymousLogin(state *smtp.ConnectionState) (smtp.Session, error) {
	return &Session{Context: bkd.Context}, nil
}

type Session struct {
	Context  context.Context
	From     string
	To       string
	Workflow Workflow
}

func (s *Session) Mail(from string, opts smtp.MailOptions) error {
	s.From = from
	return nil
}

func (s *Session) Rcpt(to string) error {
	s.To = to

	e, err := mail.ParseAddress(to)
	if err != nil {
		log.Println(s.From, "->", s.To, "501")
		log.Println(err)
		return err
	}

	for prefix, workflow := range workflows {
		if strings.HasPrefix(e.Address, prefix) {
			s.Workflow = workflow
			return nil
		}
	}

	log.Println(s.From, "->", s.To, "550")
	return &smtp.SMTPError{
		Code:         550,
		EnhancedCode: smtp.EnhancedCode{5, 5, 0},
		Message:      "No mailbox",
	}
}

func (s *Session) Data(r io.Reader) error {
	log.Println(s.From, "->", s.To)

	buf, err := ioutil.ReadAll(r)
	if err != nil {
		log.Println(err)
		return err
	}

	err = RelayToWorkflow(s.Context, s.Workflow, buf)

	if err != nil {
		log.Println("ERROR", s.Workflow, err)

		return &smtp.SMTPError{
			Code:         450,
			EnhancedCode: smtp.EnhancedCode{4, 5, 0},
			Message:      "Failed to relay message",
		}
	}

	log.Println("OK", s.Workflow, 200)

	return nil
}

func GetDefaultBranch(ctx context.Context, owner string, repo string) (*string, error) {
	repository, _, err := ghClient.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, err
	}

	return repository.DefaultBranch, nil
}

func RelayToWorkflow(ctx context.Context, workflow Workflow, buf []byte) error {
	blob := &github.Blob{
		Content:  github.String(base64.StdEncoding.EncodeToString(buf)),
		Encoding: github.String("base64"),
	}
	blob, _, err := ghClient.Git.CreateBlob(ctx, workflow.Owner, workflow.Repo, blob)
	if err != nil {
		log.Println(err)
		return err
	}

	event := github.CreateWorkflowDispatchEventRequest{
		Ref: workflow.Ref,
		Inputs: map[string]interface{}{
			"email_sha": blob.SHA,
		},
	}
	resp, err := ghClient.Actions.CreateWorkflowDispatchEventByFileName(ctx, workflow.Owner, workflow.Repo, workflow.WorkflowFileName, event)
	if err != nil {
		log.Println(err)
		return err
	} else if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		return nil
	} else {
		return fmt.Errorf("github workflow dispatch failed: %d", resp.StatusCode)
	}
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}
