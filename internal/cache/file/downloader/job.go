// Copyright 2023 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package downloader

import (
	"container/list"
	"fmt"
	"reflect"

	"github.com/googlecloudplatform/gcsfuse/internal/cache/data"
	"github.com/googlecloudplatform/gcsfuse/internal/cache/lru"
	"github.com/googlecloudplatform/gcsfuse/internal/locker"
	"github.com/googlecloudplatform/gcsfuse/internal/storage/gcs"
	"golang.org/x/net/context"
)

type jobStatusName string

const (
	NOT_STARTED jobStatusName = "NOT_STARTED"
	DOWNLOADING jobStatusName = "DOWNLOADING"
	COMPLETED   jobStatusName = "COMPLETED"
	FAILED      jobStatusName = "FAILED"
	CANCELLED   jobStatusName = "CANCELLED"
)

// Job downloads the requested object from GCS into the specified local file
// path with given permissions and ownership.
type Job struct {
	/////////////////////////
	// Constant data
	/////////////////////////

	object               *gcs.MinObject
	bucket               gcs.Bucket
	fileInfoCache        *lru.Cache
	sequentialReadSizeMb int32
	fileSpec             data.FileSpec

	/////////////////////////
	// Mutable state
	/////////////////////////

	// Represents the current status of Job.
	status JobStatus

	// list of subscribers waiting on async download.
	//
	// INVARIANT: Each element is of type jobSubscriber
	subscribers list.List

	// Context & its CancelFunc for cancelling async download in progress.
	cancelCtx  context.Context
	cancelFunc context.CancelFunc

	mu locker.Locker
}

// JobStatus represents the status of job.
type JobStatus struct {
	Name   jobStatusName
	Err    error
	Offset int64
}

// jobSubscriber represents a subscriber waiting on async download of job to
// complete downloading at least till the subscribed offset.
type jobSubscriber struct {
	notificationC    chan<- JobStatus
	subscribedOffset int64
}

func NewJob(object *gcs.MinObject, bucket gcs.Bucket, fileInfoCache *lru.Cache,
	sequentialReadSizeMb int32, fileSpec data.FileSpec) (job *Job) {
	job = &Job{
		object:               object,
		bucket:               bucket,
		fileInfoCache:        fileInfoCache,
		sequentialReadSizeMb: sequentialReadSizeMb,
		fileSpec:             fileSpec,
	}
	job.mu = locker.New("Job-"+fileSpec.Path, job.checkInvariants)
	job.init()
	return
}

// checkInvariants panic if any internal invariants have been violated.
func (job *Job) checkInvariants() {
	// INVARIANT: Each subscriber is of type jobSubscriber
	for e := job.subscribers.Front(); e != nil; e = e.Next() {
		switch e.Value.(type) {
		case jobSubscriber:
		default:
			panic(fmt.Sprintf("Unexpected element type: %v", reflect.TypeOf(e.Value)))
		}
	}
}

// init initializes the mutable members of Job corresponding to not started
// state.
func (job *Job) init() {
	job.status = JobStatus{NOT_STARTED, nil, 0}
	job.subscribers = list.List{}
	job.cancelCtx, job.cancelFunc = context.WithCancel(context.Background())
}

// Cancel changes the state of job to cancelled and cancels the async download
// job if there. Also, notifies the subscribers of job if any.
// ToDo (sethiay): Implement this function.
//
// Acquires and releases LOCK(job.mu)
func (job *Job) Cancel() {
	job.mu.Lock()
	defer job.mu.Unlock()
}

// subscribe adds subscriber for download job and returns channel which is
// notified when the download is completed at least till the subscribed offset
// or in case of failure and cancellation.
//
// Not concurrency safe and requires LOCK(job.mu)
func (job *Job) subscribe(subscribedOffset int64) (notificationC <-chan JobStatus) {
	subscriberC := make(chan JobStatus, 1)
	job.subscribers.PushBack(jobSubscriber{subscriberC, subscribedOffset})
	return subscriberC
}

// notifySubscribers notifies all the subscribers of download job in case of
// error/cancellation or when download is completed till the subscribed offset.
//
// Not concurrency safe and requires LOCK(job.mu)
func (job *Job) notifySubscribers() {
	var nextSubItr *list.Element
	for subItr := job.subscribers.Front(); subItr != nil; subItr = nextSubItr {
		subItrValue := subItr.Value.(jobSubscriber)
		nextSubItr = subItr.Next()
		if job.status.Name == FAILED || job.status.Name == CANCELLED || job.status.Offset >= subItrValue.subscribedOffset {
			subItrValue.notificationC <- job.status
			close(subItrValue.notificationC)
			job.subscribers.Remove(subItr)
		}
	}
}

// failWhileDownloading changes the status of job to failed and notifies
// subscribers about the download error.
//
// Acquires and releases LOCK(job.mu)
func (job *Job) failWhileDownloading(downloadErr error) {
	job.mu.Lock()
	job.status.Err = downloadErr
	job.status.Name = FAILED
	job.notifySubscribers()
	job.mu.Unlock()
}

// updateFileInfoCache updates the file info cache with latest offset downloaded
// by job. Returns error in case of failure.
//
// Not concurrency safe and requires LOCK(job.mu)
func (job *Job) updateFileInfoCache() (err error) {
	fileInfoKey := data.FileInfoKey{
		BucketName: job.bucket.Name(),
		ObjectName: job.object.Name,
	}
	fileInfoKeyName, err := fileInfoKey.Key()
	if err != nil {
		err = fmt.Errorf(fmt.Sprintf("init: error while calling FileInfoKey.Key() for bucket %s and object %s %v",
			fileInfoKey.BucketName, fileInfoKey.ObjectName, err))
		return
	}

	updatedFileInfo := data.FileInfo{
		Key: fileInfoKey, ObjectGeneration: job.object.Generation,
		FileSize: job.object.Size, Offset: uint64(job.status.Offset),
	}

	// To-Do(raj-prince): We should not call normal insert here as that internally
	// changes the LRU element which is undesirable given this is not user access.
	_, err = job.fileInfoCache.Insert(fileInfoKeyName, updatedFileInfo)
	if err != nil {
		err = fmt.Errorf(fmt.Sprintf("error while inserting updatedFileInfo to the FileInfoCache %s: %v", updatedFileInfo.Key, err))
		return
	}
	return
}

// Download downloads object till the given offset if not already downloaded
// and waits for download if waitForDownload is true.
// ToDo (sethiay): Implement this function.
//
// Acquires and releases LOCK(job.mu)
func (job *Job) Download(ctx context.Context, offset int64, waitForDownload bool) (jobStatus JobStatus) {
	job.mu.Lock()
	defer job.mu.Unlock()
	return
}
