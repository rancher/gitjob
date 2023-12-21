package webhook

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/go-playground/webhooks/v6/azuredevops"
	gogsclient "github.com/gogits/go-gogs-client"
	"github.com/gorilla/mux"
	v1controller "github.com/rancher/gitjob/pkg/generated/controllers/gitjob.cattle.io/v1"
	"github.com/rancher/gitjob/pkg/types"
	"github.com/rancher/steve/pkg/aggregation"
	corev1controller "github.com/rancher/wrangler/pkg/generated/controllers/core/v1"
	"github.com/sirupsen/logrus"
	"gopkg.in/go-playground/webhooks.v5/bitbucket"
	bitbucketserver "gopkg.in/go-playground/webhooks.v5/bitbucket-server"
	"gopkg.in/go-playground/webhooks.v5/github"
	"gopkg.in/go-playground/webhooks.v5/gitlab"
	"gopkg.in/go-playground/webhooks.v5/gogs"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	webhookSecretName = "gitjob-webhook" //nolint:gosec // this is a resource name

	githubKey          = "github"
	gitlabKey          = "gitlab"
	bitbucketKey       = "bitbucket"
	bitbucketServerKey = "bitbucket-server"
	gogsKey            = "gogs"

	branchRefPrefix = "refs/heads/"
	tagRefPrefix    = "refs/tags/"
)

type Webhook struct {
	gitjobs   v1controller.GitJobController
	secrets   corev1controller.SecretController
	namespace string

	github          *github.Webhook
	gitlab          *gitlab.Webhook
	bitbucket       *bitbucket.Webhook
	bitbucketServer *bitbucketserver.Webhook
	gogs            *gogs.Webhook
	azureDevops     *azuredevops.Webhook
}

func New(ctx context.Context, rContext *types.Context) *Webhook {
	webhook := &Webhook{
		gitjobs:   rContext.Gitjob.Gitjob().V1().GitJob(),
		secrets:   rContext.Core.Core().V1().Secret(),
		namespace: rContext.Namespace,
	}

	rContext.Core.Core().V1().Secret().OnChange(ctx, "webhook-secret", webhook.onSecretChange)
	return webhook
}

func (w *Webhook) onSecretChange(_ string, secret *corev1.Secret) (*corev1.Secret, error) {
	if secret == nil || secret.DeletionTimestamp != nil {
		return nil, nil
	}

	if secret.Name != webhookSecretName && secret.Namespace != w.namespace {
		return nil, nil
	}

	var err error
	w.github, err = github.New(github.Options.Secret(string(secret.Data[githubKey])))
	if err != nil {
		return nil, err
	}
	w.gitlab, err = gitlab.New(gitlab.Options.Secret(string(secret.Data[gitlabKey])))
	if err != nil {
		return nil, err
	}
	w.bitbucket, err = bitbucket.New(bitbucket.Options.UUID(string(secret.Data[bitbucketKey])))
	if err != nil {
		return nil, err
	}
	w.bitbucketServer, err = bitbucketserver.New(bitbucketserver.Options.Secret(string(secret.Data[bitbucketServerKey])))
	if err != nil {
		return nil, err
	}
	w.gogs, err = gogs.New(gogs.Options.Secret(string(secret.Data[gogsKey])))
	if err != nil {
		return nil, err
	}
	w.azureDevops, err = azuredevops.New()
	if err != nil {
		return nil, err
	}
	return nil, nil
}

/*
{"subscriptionId":"00000000-0000-0000-0000-000000000000","notificationId":1,"id":"03c164c2-8912-4d5e-8009-3707d5f83734","eventType":"git.push","publisherId":"tfs","message":{"text":"Jamal Hartnett pushed updates to Fabrikam-Fiber-Git:master.","html":"Jamal Hartnett pushed updates to Fabrikam-Fiber-Git:master.","markdown":"Jamal Hartnett pushed updates to `Fabrikam-Fiber-Git`:`master`."},"detailedMessage":{"text":"Jamal Hartnett pushed a commit to Fabrikam-Fiber-Git:master.\n - Fixed bug in web.config file 33b55f7c","html":"Jamal Hartnett pushed a commit to <a href=\"https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_git/Fabrikam-Fiber-Git/\">Fabrikam-Fiber-Git</a>:<a href=\"https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_git/Fabrikam-Fiber-Git/#version=GBmaster\">master</a>.\n<ul>\n<li>Fixed bug in web.config file <a href=\"https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_git/Fabrikam-Fiber-Git/commit/33b55f7cb7e7e245323987634f960cf4a6e6bc74\">33b55f7c</a>\n</ul>","markdown":"Jamal Hartnett pushed a commit to [Fabrikam-Fiber-Git](https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_git/Fabrikam-Fiber-Git/):[master](https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_git/Fabrikam-Fiber-Git/#version=GBmaster).\n* Fixed bug in web.config file [33b55f7c](https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_git/Fabrikam-Fiber-Git/commit/33b55f7cb7e7e245323987634f960cf4a6e6bc74)"},"resource":{"commits":[{"commitId":"33b55f7cb7e7e245323987634f960cf4a6e6bc74","author":{"name":"Jamal Hartnett","email":"fabrikamfiber4@hotmail.com","date":"2015-02-25T19:01:00Z"},"committer":{"name":"Jamal Hartnett","email":"fabrikamfiber4@hotmail.com","date":"2015-02-25T19:01:00Z"},"comment":"Fixed bug in web.config file","url":"https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_git/Fabrikam-Fiber-Git/commit/33b55f7cb7e7e245323987634f960cf4a6e6bc74"}],"refUpdates":[{"name":"refs/heads/master","oldObjectId":"aad331d8d3b131fa9ae03cf5e53965b51942618a","newObjectId":"33b55f7cb7e7e245323987634f960cf4a6e6bc74"}],"repository":{"id":"278d5cd2-584d-4b63-824a-2ba458937249","name":"Fabrikam-Fiber-Git","url":"https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_apis/git/repositories/278d5cd2-584d-4b63-824a-2ba458937249","project":{"id":"6ce954b1-ce1f-45d1-b94d-e6bf2464ba2c","name":"Fabrikam-Fiber-Git","url":"https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_apis/projects/6ce954b1-ce1f-45d1-b94d-e6bf2464ba2c","state":"wellFormed","visibility":"unchanged","lastUpdateTime":"0001-01-01T00:00:00"},"defaultBranch":"refs/heads/master","remoteUrl":"https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_git/Fabrikam-Fiber-Git"},"pushedBy":{"displayName":"Jamal Hartnett","id":"00067FFED5C7AF52@Live.com","uniqueName":"fabrikamfiber4@hotmail.com"},"pushId":14,"date":"2014-05-02T19:17:13.3309587Z","url":"https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_apis/git/repositories/278d5cd2-584d-4b63-824a-2ba458937249/pushes/14"},"resourceVersion":"1.0","resourceContainers":{"collection":{"id":"c12d0eb8-e382-443b-9f9c-c52cba5014c2"},"account":{"id":"f844ec47-a9db-4511-8281-8b63f4eaf94e"},"project":{"id":"be9b3917-87e6-42a4-a549-2bc06a7a878f"}},"createdDate":"2023-12-20T15:34:35.9881845Z"}

minimal {"subscriptionId":"00000000-0000-0000-0000-000000000000","notificationId":2,"id":"03c164c2-8912-4d5e-8009-3707d5f83734","eventType":"git.push","publisherId":"tfs","message":{"text":"Jamal Hartnett pushed updates to Fabrikam-Fiber-Git:master.","html":"Jamal Hartnett pushed updates to Fabrikam-Fiber-Git:master.","markdown":"Jamal Hartnett pushed updates to `Fabrikam-Fiber-Git`:`master`."},"detailedMessage":{"text":"Jamal Hartnett pushed a commit to Fabrikam-Fiber-Git:master.\n - Fixed bug in web.config file 33b55f7c","html":"Jamal Hartnett pushed a commit to <a href=\"https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_git/Fabrikam-Fiber-Git/\">Fabrikam-Fiber-Git</a>:<a href=\"https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_git/Fabrikam-Fiber-Git/#version=GBmaster\">master</a>.\n<ul>\n<li>Fixed bug in web.config file <a href=\"https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_git/Fabrikam-Fiber-Git/commit/33b55f7cb7e7e245323987634f960cf4a6e6bc74\">33b55f7c</a>\n</ul>","markdown":"Jamal Hartnett pushed a commit to [Fabrikam-Fiber-Git](https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_git/Fabrikam-Fiber-Git/):[master](https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_git/Fabrikam-Fiber-Git/#version=GBmaster).\n* Fixed bug in web.config file [33b55f7c](https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_git/Fabrikam-Fiber-Git/commit/33b55f7cb7e7e245323987634f960cf4a6e6bc74)"},"resource":{"url":"https://fabrikam-fiber-inc.visualstudio.com/DefaultCollection/_apis/git/repositories/278d5cd2-584d-4b63-824a-2ba458937249/pushes/14","pushId":14},"resourceVersion":"1.0","resourceContainers":{"collection":{"id":"c12d0eb8-e382-443b-9f9c-c52cba5014c2"},"account":{"id":"f844ec47-a9db-4511-8281-8b63f4eaf94e"},"project":{"id":"be9b3917-87e6-42a4-a549-2bc06a7a878f"}},"createdDate":"2023-12-20T15:36:15.2554757Z"}
*/
func (w *Webhook) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	// credit from https://github.com/argoproj/argo-cd/blob/97003caebcaafe1683e71934eb483a88026a4c33/util/webhook/webhook.go#L327-L350
	/*b, err2 := httputil.DumpRequest(r.Clone(context.TODO()), true)
	if err2 != nil {
		log.Fatalln(err2)
	}
	fmt.Println("request" + string(b))*/

	var payload interface{}
	var err error

	switch {
	//Gogs needs to be checked before Github since it carries both Gogs and (incompatible) Github headers
	case r.Header.Get("X-Gogs-Event") != "":
		payload, err = w.gogs.Parse(r, gogs.PushEvent)
	case r.Header.Get("X-GitHub-Event") != "":
		payload, err = w.github.Parse(r, github.PushEvent)
	case r.Header.Get("X-Gitlab-Event") != "":
		payload, err = w.gitlab.Parse(r, gitlab.PushEvents, gitlab.TagEvents)
	case r.Header.Get("X-Hook-UUID") != "":
		payload, err = w.bitbucket.Parse(r, bitbucket.RepoPushEvent)
	case r.Header.Get("X-Event-Key") != "":
		payload, err = w.bitbucketServer.Parse(r, bitbucketserver.RepositoryReferenceChangedEvent)
	case r.Header.Get("X-Vss-Activityid") != "" || r.Header.Get("X-Vss-Subscriptionid") != "":
		payload, err = w.azureDevops.Parse(r, azuredevops.GitPushEventType)
	default:
		logrus.Debug("Ignoring unknown webhook event")
		return
	}

	logrus.Debugf("Webhook payload %+v", payload)

	if err != nil {
		logAndReturn(rw, err)
		return
	}

	var revision, branch, tag string
	var repoURLs []string
	// credit from https://github.com/argoproj/argo-cd/blob/97003caebcaafe1683e71934eb483a88026a4c33/util/webhook/webhook.go#L84-L87
	switch t := payload.(type) {
	case github.PushPayload:
		branch, tag = getBranchTagFromRef(t.Ref)
		revision = t.After
		repoURLs = append(repoURLs, t.Repository.HTMLURL)
	case gitlab.PushEventPayload:
		branch, tag = getBranchTagFromRef(t.Ref)
		revision = t.CheckoutSHA
		repoURLs = append(repoURLs, t.Project.WebURL)
	case gitlab.TagEventPayload:
		branch, tag = getBranchTagFromRef(t.Ref)
		revision = t.CheckoutSHA
		repoURLs = append(repoURLs, t.Project.WebURL)
	// https://support.atlassian.com/bitbucket-cloud/docs/event-payloads/#Push
	case bitbucket.RepoPushPayload:
		repoURLs = append(repoURLs, t.Repository.Links.HTML.Href)
		for _, change := range t.Push.Changes {
			revision = change.New.Target.Hash
			if change.New.Type == "branch" {
				branch = change.New.Name
			} else if change.New.Type == "tag" {
				tag = change.New.Name
			}
			break
		}
	case bitbucketserver.RepositoryReferenceChangedPayload:
		for _, l := range t.Repository.Links["clone"].([]interface{}) {
			link := l.(map[string]interface{})
			if link["name"] == "http" {
				repoURLs = append(repoURLs, link["href"].(string))
			}
			if link["name"] == "ssh" {
				repoURLs = append(repoURLs, link["href"].(string))
			}
		}
		for _, change := range t.Changes {
			revision = change.ToHash
			branch, tag = getBranchTagFromRef(change.ReferenceId)
			break
		}
	case gogsclient.PushPayload:
		repoURLs = append(repoURLs, t.Repo.HTMLURL)
		branch, tag = getBranchTagFromRef(t.Ref)
		revision = t.After
	case azuredevops.GitPushEvent:
		repoURLs = append(repoURLs, t.Resource.URL)
		branch, tag = getBranchTagFromRef(t.Resource.RefUpdates[0].Name) // TODO check newest in array?
		revision = t.Resource.Commits[0].CommitID                        // TODO check newest in array?
	}

	fmt.Printf("revision %s, branch %s, tag %s\n", revision, branch, tag)

	gitjobs, err := w.gitjobs.Cache().List("", labels.Everything())
	if err != nil {
		logAndReturn(rw, err)
		return
	}

	for _, repo := range repoURLs {
		u, err := url.Parse(repo)
		if err != nil {
			logAndReturn(rw, err)
			return
		}
		regexpStr := `(?i)(http://|https://|\w+@|ssh://(\w+@)?)` + u.Hostname() + "(:[0-9]+|)[:/]" + u.Path[1:] + "(\\.git)?"
		repoRegexp, err := regexp.Compile(regexpStr)
		if err != nil {
			logAndReturn(rw, err)
			return
		}
		for _, gitjob := range gitjobs {
			if gitjob.Spec.Git.Revision != "" {
				continue
			}

			if !repoRegexp.MatchString(gitjob.Spec.Git.Repo) {
				continue
			}

			// if onTag is enabled, we only watch tag event, as it can be coming from any branch
			if gitjob.Spec.Git.OnTag != "" {
				// skipping if gitjob is watching tag only and tag is empty(not a tag event)
				if tag == "" {
					continue
				}
				contraints, err := semver.NewConstraint(gitjob.Spec.Git.OnTag)
				if err != nil {
					logrus.Warnf("Failed to parsing onTag semver from %s/%s, err: %v, skipping", gitjob.Namespace, gitjob.Name, err)
					continue
				}
				v, err := semver.NewVersion(tag)
				if err != nil {
					logrus.Warnf("Failed to parsing semver on incoming tag, err: %v, skipping", err)
					continue
				}
				if !contraints.Check(v) {
					continue
				}
			} else if gitjob.Spec.Git.Branch != "" {
				// else we check if the branch from webhook matches gitjob's branch
				if branch == "" || branch != gitjob.Spec.Git.Branch {
					continue
				}
			}

			dp := gitjob.DeepCopy()
			if dp.Status.Commit != revision && revision != "" {
				dp.Status.Commit = revision
				newObj, err := w.gitjobs.UpdateStatus(dp)
				if err != nil {
					logAndReturn(rw, err)
					return
				}
				// if syncInterval is not set and webhook is configured, set it to 1 hour
				if newObj.Spec.SyncInterval == 0 {
					newObj.Spec.SyncInterval = 3600
					if _, err := w.gitjobs.Update(newObj); err != nil {
						logAndReturn(rw, err)
						return
					}
				}
			}
		}
	}
	rw.WriteHeader(200)
	rw.Write([]byte("succeeded"))
}

func logAndReturn(rw http.ResponseWriter, err error) {
	logrus.Errorf("Webhook processing failed: %s", err)
	rw.WriteHeader(500)
	rw.Write([]byte(err.Error()))
	return
}

func HandleHooks(ctx context.Context, rContext *types.Context) http.Handler {
	root := mux.NewRouter()
	webhook := New(ctx, rContext)
	root.UseEncodedPath()
	root.Handle("/", webhook)
	aggregation.Watch(ctx,
		rContext.Core.Core().V1().Secret(),
		rContext.Namespace,
		"steve-aggregation",
		root)
	return root
}

// git ref docs: https://git-scm.com/book/en/v2/Git-Internals-Git-References
func getBranchTagFromRef(ref string) (string, string) {
	if strings.HasPrefix(ref, branchRefPrefix) {
		return strings.TrimPrefix(ref, branchRefPrefix), ""
	}

	if strings.HasPrefix(ref, tagRefPrefix) {
		return "", strings.TrimPrefix(ref, tagRefPrefix)
	}

	return "", ""
}
