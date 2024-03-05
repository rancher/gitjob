package webhook

import (
	"bytes"

	"github.com/golang/mock/gomock"
	v1 "github.com/rancher/gitjob/pkg/apis/gitjob.cattle.io/v1"
	"github.com/rancher/gitjob/pkg/webhook/azuredevops"
	"github.com/rancher/wrangler/v2/pkg/generic/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"net/http"
	"testing"

	"gotest.tools/assert"
)

func TestGetBranchTagFromRef(t *testing.T) {
	inputs := []string{
		"refs/heads/master",
		"refs/heads/test",
		"refs/head/foo",
		"refs/tags/v0.1.1",
		"refs/tags/v0.1.2",
		"refs/tag/v0.1.3",
	}

	outputs := [][]string{
		{"master", ""},
		{"test", ""},
		{"", ""},
		{"", "v0.1.1"},
		{"", "v0.1.2"},
		{"", ""},
	}

	for i, input := range inputs {
		branch, tag := getBranchTagFromRef(input)
		assert.Equal(t, branch, outputs[i][0])
		assert.Equal(t, tag, outputs[i][1])
	}
}

func TestAzureDevopsWebhook(t *testing.T) {
	const commit = "f00c3a181697bb3829a6462e931c7456bbed557b"
	const repoURL = "https://dev.azure.com/fleet/git-test/_git/git-test"
	ctrl := gomock.NewController(t)
	mockGitjob := fake.NewMockControllerInterface[*v1.GitJob, *v1.GitJobList](ctrl)
	mockGitjobCache := fake.NewMockCacheInterface[*v1.GitJob](ctrl)
	gitjob := &v1.GitJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: v1.GitJobSpec{
			Git: v1.GitInfo{
				Repo:   repoURL,
				Branch: "main",
			},
		},
	}
	gitjobs := []*v1.GitJob{gitjob}
	w := &Webhook{gitjobs: mockGitjob}
	w.azureDevops, _ = azuredevops.New()
	jsonBody := []byte(`{"subscriptionId":"xxx","notificationId":1,"id":"xxx","eventType":"git.push","publisherId":"tfs","message":{"text":"commit pushed","html":"commit pushed"},"detailedMessage":{"text":"pushed a commit to git-test"},"resource":{"commits":[{"commitId":"` + commit + `","author":{"name":"fleet","email":"fleet@suse.com","date":"2024-01-05T10:16:56Z"},"committer":{"name":"fleet","email":"fleet@suse.com","date":"2024-01-05T10:16:56Z"},"comment":"test commit","url":"https://dev.azure.com/fleet/_apis/git/repositories/xxx/commits/f00c3a181697bb3829a6462e931c7456bbed557b"}],"refUpdates":[{"name":"refs/heads/main","oldObjectId":"135f8a827edae980466f72eef385881bb4e158d8","newObjectId":"` + commit + `"}],"repository":{"id":"xxx","name":"git-test","url":"https://dev.azure.com/fleet/_apis/git/repositories/xxx","project":{"id":"xxx","name":"git-test","url":"https://dev.azure.com/fleet/_apis/projects/xxx","state":"wellFormed","visibility":"unchanged","lastUpdateTime":"0001-01-01T00:00:00"},"defaultBranch":"refs/heads/main","remoteUrl":"` + repoURL + `"},"pushedBy":{"displayName":"Fleet","url":"https://spsprodneu1.vssps.visualstudio.com/xxx/_apis/Identities/xxx","_links":{"avatar":{"href":"https://dev.azure.com/fleet/_apis/GraphProfile/MemberAvatars/msa.xxxx"}},"id":"xxx","uniqueName":"fleet@suse.com","imageUrl":"https://dev.azure.com/fleet/_api/_common/identityImage?id=xxx","descriptor":"xxxx"},"pushId":22,"date":"2024-01-05T10:17:18.735088Z","url":"https://dev.azure.com/fleet/_apis/git/repositories/xxx/pushes/22","_links":{"self":{"href":"https://dev.azure.com/fleet/_apis/git/repositories/xxx/pushes/22"},"repository":{"href":"https://dev.azure.com/fleet/xxx/_apis/git/repositories/xxx"},"commits":{"href":"https://dev.azure.com/fleet/_apis/git/repositories/xxx/pushes/22/commits"},"pusher":{"href":"https://spsprodneu1.vssps.visualstudio.com/xxx/_apis/Identities/xxx"},"refs":{"href":"https://dev.azure.com/fleet/xxx/_apis/git/repositories/xxx/refs/heads/main"}}},"resourceVersion":"1.0","resourceContainers":{"collection":{"id":"xxx","baseUrl":"https://dev.azure.com/fleet/"},"account":{"id":"ec365173-fce3-4dfc-8fc2-950f0b5728b1","baseUrl":"https://dev.azure.com/fleet/"},"project":{"id":"xxx","baseUrl":"https://dev.azure.com/fleet/"}},"createdDate":"2024-01-05T10:17:26.0098694Z"}`)
	bodyReader := bytes.NewReader(jsonBody)
	req, err := http.NewRequest(http.MethodPost, repoURL, bodyReader)
	if err != nil {
		t.Errorf("unexpected err %v", err)
	}
	h := http.Header{}
	h.Add("X-Vss-Activityid", "xxx")
	req.Header = h

	expectedStatusUpdate := gitjob.DeepCopy()
	expectedStatusUpdate.Status.Commit = commit
	mockGitjobCache.EXPECT().List("", labels.Everything()).Return(gitjobs, nil)
	mockGitjob.EXPECT().Cache().Return(mockGitjobCache)
	mockGitjob.EXPECT().UpdateStatus(expectedStatusUpdate).Return(gitjob, nil)
	mockGitjob.EXPECT().Update(gitjob)

	w.ServeHTTP(&responseWriter{}, req)
}

func TestAzureDevopsWebhookWithSSHURL(t *testing.T) {
	const (
		commit            = "f00c3a181697bb3829a6462e931c7456bbed557b"
		gitRepoURL        = "git@ssh.dev.azure.com:v3/fleet/git-test/git-test"
		responseRemoteURL = "https://dev.azure.com/fleet/git-test/_git/git-test"
	)

	ctrl := gomock.NewController(t)
	mockGitjob := fake.NewMockControllerInterface[*v1.GitJob, *v1.GitJobList](ctrl)
	mockGitjobCache := fake.NewMockCacheInterface[*v1.GitJob](ctrl)
	gitjob := &v1.GitJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: v1.GitJobSpec{
			Git: v1.GitInfo{
				Repo:   gitRepoURL,
				Branch: "main",
			},
		},
	}
	gitjobs := []*v1.GitJob{gitjob}
	w := &Webhook{gitjobs: mockGitjob}
	w.azureDevops, _ = azuredevops.New()
	jsonBody := []byte(`{"subscriptionId":"xxx","notificationId":1,"id":"xxx","eventType":"git.push","publisherId":"tfs","message":{"text":"commit pushed","html":"commit pushed"},"detailedMessage":{"text":"pushed a commit to git-test"},"resource":{"commits":[{"commitId":"` + commit + `","author":{"name":"fleet","email":"fleet@suse.com","date":"2024-01-05T10:16:56Z"},"committer":{"name":"fleet","email":"fleet@suse.com","date":"2024-01-05T10:16:56Z"},"comment":"test commit","url":"https://dev.azure.com/fleet/_apis/git/repositories/xxx/commits/f00c3a181697bb3829a6462e931c7456bbed557b"}],"refUpdates":[{"name":"refs/heads/main","oldObjectId":"135f8a827edae980466f72eef385881bb4e158d8","newObjectId":"` + commit + `"}],"repository":{"id":"xxx","name":"git-test","url":"https://dev.azure.com/fleet/_apis/git/repositories/xxx","project":{"id":"xxx","name":"git-test","url":"https://dev.azure.com/fleet/_apis/projects/xxx","state":"wellFormed","visibility":"unchanged","lastUpdateTime":"0001-01-01T00:00:00"},"defaultBranch":"refs/heads/main","remoteUrl":"` + responseRemoteURL + `"},"pushedBy":{"displayName":"Fleet","url":"https://spsprodneu1.vssps.visualstudio.com/xxx/_apis/Identities/xxx","_links":{"avatar":{"href":"https://dev.azure.com/fleet/_apis/GraphProfile/MemberAvatars/msa.xxxx"}},"id":"xxx","uniqueName":"fleet@suse.com","imageUrl":"https://dev.azure.com/fleet/_api/_common/identityImage?id=xxx","descriptor":"xxxx"},"pushId":22,"date":"2024-01-05T10:17:18.735088Z","url":"https://dev.azure.com/fleet/_apis/git/repositories/xxx/pushes/22","_links":{"self":{"href":"https://dev.azure.com/fleet/_apis/git/repositories/xxx/pushes/22"},"repository":{"href":"https://dev.azure.com/fleet/xxx/_apis/git/repositories/xxx"},"commits":{"href":"https://dev.azure.com/fleet/_apis/git/repositories/xxx/pushes/22/commits"},"pusher":{"href":"https://spsprodneu1.vssps.visualstudio.com/xxx/_apis/Identities/xxx"},"refs":{"href":"https://dev.azure.com/fleet/xxx/_apis/git/repositories/xxx/refs/heads/main"}}},"resourceVersion":"1.0","resourceContainers":{"collection":{"id":"xxx","baseUrl":"https://dev.azure.com/fleet/"},"account":{"id":"ec365173-fce3-4dfc-8fc2-950f0b5728b1","baseUrl":"https://dev.azure.com/fleet/"},"project":{"id":"xxx","baseUrl":"https://dev.azure.com/fleet/"}},"createdDate":"2024-01-05T10:17:26.0098694Z"}`)
	bodyReader := bytes.NewReader(jsonBody)
	req, err := http.NewRequest(http.MethodPost, responseRemoteURL, bodyReader)
	if err != nil {
		t.Errorf("unexpected err %v", err)
	}
	h := http.Header{}
	h.Add("X-Vss-Activityid", "xxx")
	req.Header = h

	expectedStatusUpdate := gitjob.DeepCopy()
	expectedStatusUpdate.Status.Commit = commit
	mockGitjobCache.EXPECT().List("", labels.Everything()).Return(gitjobs, nil)
	mockGitjob.EXPECT().Cache().Return(mockGitjobCache)
	mockGitjob.EXPECT().UpdateStatus(expectedStatusUpdate).Return(gitjob, nil)
	mockGitjob.EXPECT().Update(gitjob)

	w.ServeHTTP(&responseWriter{}, req)
}

type responseWriter struct{}

func (r *responseWriter) Header() http.Header {
	return http.Header{}
}
func (r *responseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (r *responseWriter) WriteHeader(statusCode int) {}
