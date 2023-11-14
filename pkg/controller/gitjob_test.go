//go:generate mockgen --build_flags=--mod=mod -destination=../mocks/poller_mock.go -package=mocks github.com/rancher/gitjob/pkg/controller GitPoller
//go:generate mockgen --build_flags=--mod=mod -destination=../mocks/client_mock.go -package=mocks sigs.k8s.io/controller-runtime/pkg/client Client,SubResourceWriter

package controller

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	gitjobv1 "github.com/rancher/gitjob/pkg/apis/gitjob.cattle.io/v1"
	"github.com/rancher/gitjob/pkg/mocks"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestReconcile_AddOrModifyGitRepoWatchIsCalled_WhenGitRepoIsCreatedOrModified(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	scheme := runtime.NewScheme()
	utilruntime.Must(gitjobv1.AddToScheme(scheme))
	utilruntime.Must(batchv1.AddToScheme(scheme))
	gitJob := gitjobv1.GitJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gitjob",
			Namespace: "default",
		},
	}
	namespacedName := types.NamespacedName{Name: gitJob.Name, Namespace: gitJob.Namespace}
	ctx := context.TODO()
	client := mocks.NewMockClient(mockCtrl)
	statusClient := mocks.NewMockSubResourceWriter(mockCtrl)
	statusClient.EXPECT().Update(ctx, gomock.Any())
	client.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	client.EXPECT().Status().Return(statusClient)
	poller := mocks.NewMockGitPoller(mockCtrl)
	poller.EXPECT().AddOrModifyGitRepoWatch(ctx, gomock.Any()).Times(1)
	poller.EXPECT().CleanUpWatches(ctx).Times(0)

	r := GitJobReconciler{
		Client:    client,
		Scheme:    scheme,
		Image:     "",
		GitPoller: poller,
	}
	_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}

func TestReconcile_PurgeWatchesIsCalled_WhenGitRepoIsCreatedOrModified(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	scheme := runtime.NewScheme()
	utilruntime.Must(gitjobv1.AddToScheme(scheme))
	utilruntime.Must(batchv1.AddToScheme(scheme))
	ctx := context.TODO()
	namespacedName := types.NamespacedName{Name: "gitJob", Namespace: "default"}
	client := mocks.NewMockClient(mockCtrl)
	client.EXPECT().Get(ctx, namespacedName, gomock.Any()).Times(1).Return(errors.NewNotFound(schema.GroupResource{}, "NotFound"))
	poller := mocks.NewMockGitPoller(mockCtrl)
	poller.EXPECT().AddOrModifyGitRepoWatch(ctx, gomock.Any()).Times(0)
	poller.EXPECT().CleanUpWatches(ctx).Times(1)

	r := GitJobReconciler{
		Client:    client,
		Scheme:    scheme,
		Image:     "",
		GitPoller: poller,
	}
	_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
}
